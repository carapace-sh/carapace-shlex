# V2 Architecture — The Common Token Model

How the v2 lexer is structured: a common token model and tokenizer state machine that each shell format plugs into via the `Format` interface. V1 was POSIX-only; v2 generalizes to multiple shell formats (including non-POSIX).

> **Source of truth**: `shlex.go` (state machine, `Token`, `Split`, `SplitWith`), `format.go` (`Format` interface, `Span`), `completion.go` (`CompletionContext`, `SplitForCompletion`), `tokenslice.go` (`TokenSlice` operations), `wordbreak.go` (`WordbreakType`), `format_*.go` (per-shell formats). For how shells differ lexically, see [comparison.md](comparison.md).

## V1 Recap (POSIX-Only)

V1 was a single lexer hardcoded to POSIX shell lexing. The rune classes and operator set were fixed:

```go
// shlex.go (v1)
const (
	spaceRunes            = " \t\r\n"
	escapingQuoteRunes    = `"`      // double quotes support \ escapes
	nonEscapingQuoteRunes = "'"      // single quotes: literal
	escapeRunes           = `\`
	commentRunes          = "#"
)

const BASH_WORDBREAKS = " \t\r\n" + `"'@><=;|&(:`
```

The classifier (`newDefaultClassifier`) read `COMP_WORDBREAKS` from the environment and merged any custom wordbreak runes that weren't already classified. The tokenizer state machine (`scanStream`) was a single hardcoded switch over `LexerState`.

This worked for bash and bash-like shells (zsh, oil OSH) but could not express:

- **Non-POSIX quote types** — fish's `\'` inside single quotes, nushell's backtick strings, PowerShell's backtick escape, elvish's `''` doubled-quote escaping.
- **Non-POSIX operators** — fish's `and`/`or`/`not` keyword operators, cmd's `&` command separator and `^` escape.
- **Different escape characters** — PowerShell uses backtick (`` ` ``) instead of backslash; cmd uses caret (`^`).
- **Bareword backslash** — elvish treats `\` as a literal bareword character outside quotes.

## V2 Architecture

V2 keeps the proven tokenizer state machine and `TokenSlice` operations but makes the **rune classification**, **operator grammar**, and **quote behavior** configurable per shell format via the `Format` interface. A format is a small struct that implements:

```go
// format.go
type Format interface {
	// Classifier returns a rune classifier mapping runes to runeTokenClass.
	// Called once per tokenizer; should be freshly built (may read env vars).
	Classifier() tokenClassifier

	// ClassifyOperator maps a wordbreak token's RawValue to a WordbreakType.
	ClassifyOperator(raw string) WordbreakType

	// KeywordOperators returns bare-word operators (e.g. fish "and"/"or")
	// that should be treated as WORDBREAK_TOKEN despite being word characters.
	// Returns nil for shells without keyword operators.
	KeywordOperators() map[string]WordbreakType

	// NonEscapingQuoteEscapes returns true if the non-escaping quote (single
	// quote) supports limited escapes: '' (doubled quote) → literal quote.
	// Supported by: fish, elvish, zsh (RC_QUOTES), PowerShell.
	NonEscapingQuoteEscapes() bool

	// NonEscapingQuoteBackslashEscapes returns true if backslash (\) is an
	// escape inside the non-escaping quote (single quotes): \' and \\.
	// Only fish needs this.
	NonEscapingQuoteBackslashEscapes() bool

	// EscapeNotBareword returns false if the escape character (backslash)
	// is a literal bareword character outside quotes rather than an escape.
	// Only elvish needs this (\ is a bareword char in elvish).
	EscapeNotBareword() bool

	// QuoteWord quotes a single word for safe insertion into a command line.
	// Used by JoinWith. Each format uses its shell's preferred quoting style.
	QuoteWord(s string) string
}
```

## The Common Token Model

These types are shared across all formats:

### TokenType

```go
type TokenType int

const (
	UNKNOWN_TOKEN     TokenType = iota
	WORD_TOKEN          // a word (possibly built from quoted/escaped segments)
	SPACE_TOKEN         // whitespace separating words
	COMMENT_TOKEN       // a comment (skipped by the lexer)
	WORDBREAK_TOKEN     // an operator or word-break sequence
)
```

The lexer (`lexer.Next`) yields only `WORD_TOKEN` and `WORDBREAK_TOKEN`, skipping comments. The tokenizer (`tokenizer.Next`) yields all types including `COMMENT_TOKEN` and the empty trailing `WORD_TOKEN`.

### LexerState

```go
type LexerState int

const (
	START_STATE             // no runes seen yet for this token
	IN_WORD_STATE           // processing regular runes in a word
	ESCAPING_STATE          // just consumed an escape rune; next rune is literal
	ESCAPING_QUOTED_STATE   // just consumed an escape rune within an escaping quote
	QUOTING_ESCAPING_STATE  // inside an escaping quote ("..." in POSIX)
	QUOTING_STATE           // inside a non-escaping quote ('...' in POSIX)
	COMMENT_STATE           // inside a comment
	WORDBREAK_STATE         // just consumed a wordbreak/operator rune
)
```

`Token.State` reports the state **after** the token was emitted. For a word that ends with an open quote (cursor inside quotes), the state is `QUOTING_STATE` or `QUOTING_ESCAPING_STATE` — this is the signal completion code uses to know it must close the quote.

### Span and Token

```go
// format.go
type Span struct {
	Start int // rune offset of the first character
	End   int // rune offset after the last character
}

// shlex.go
type Token struct {
	Type           TokenType
	Value          string      // the processed value (quotes/escapes removed)
	RawValue       string      // the raw source text including quote chars
	Span           Span        // rune offsets in the input stream
	State          LexerState  // state after emitting this token
	WordbreakType  WordbreakType `json:",omitempty"`
	WordbreakIndex int         // index of last opening quote in Value
}
```

`Span` replaces v1's `Index` field. `Span.Start` is the rune offset of the first character; `Span.End` is the rune offset after the last character. The `adjoins` check uses `Span.End == other.Span.Start` to detect contiguous tokens that `Words()` should merge.

### TokenSlice Operations

These are format-agnostic and work on the token stream produced by any format's tokenizer:

| Method | Purpose |
|--------|---------|
| `Split(s)` / `SplitWith(s, format)` | Entry point — lexes a string into tokens |
| `Words()` | Merges adjoining tokens (contiguous `Span`) into single words |
| `CurrentPipeline()` | Returns the last pipeline (splits on `\|`, `&&`, `;`, etc.) |
| `FilterRedirects()` | Removes redirect operators and their targets |
| `WordbreakPrefix()` | Extracts the completion prefix up to the cursor |
| `CurrentToken()` | Returns the last token |
| `Strings()` | Returns word values as `[]string` |

### WordbreakType

Operators are classified so that `CurrentPipeline` and `FilterRedirects` can decide what's a pipeline delimiter vs a redirect vs other. The v1 hardcoded `wordbreakType()` function was renamed to `bashWordbreakType()` and is now called via `Format.ClassifyOperator()`:

```go
type WordbreakType int

const (
	WORDBREAK_UNKNOWN
	// redirects
	WORDBREAK_REDIRECT_INPUT          // <
	WORDBREAK_REDIRECT_OUTPUT         // >
	WORDBREAK_REDIRECT_OUTPUT_APPEND  // >>
	WORDBREAK_REDIRECT_OUTPUT_BOTH    // &> or >&
	WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND // &>>
	WORDBREAK_REDIRECT_INPUT_STRING   // <<<
	WORDBREAK_REDIRECT_INPUT_DUPLICATE // <&
	WORDBREAK_REDIRECT_INPUT_OUTPUT   // <>
	// pipeline/list operators
	WORDBREAK_PIPE                    // |
	WORDBREAK_PIPE_WITH_STDERR        // |&
	WORDBREAK_LIST_ASYNC              // &
	WORDBREAK_LIST_SEQUENTIAL         // ;
	WORDBREAK_LIST_AND                // &&
	WORDBREAK_LIST_OR                 // ||
	// custom COMP_WORDBREAKS
	WORDBREAK_CUSTOM
)
```

`IsPipelineDelimiter()` and `IsRedirect()` drive `CurrentPipeline` and `FilterRedirects`.

## CompletionContext

The `SplitForCompletion` function provides a structured completion context, replacing the manual `tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken()` chains that carapace used with v1:

```go
// completion.go
type CompletionContext struct {
	Words          []string    // pipeline words (redirects filtered)
	CurrentWord    string      // word at cursor (dequoted)
	RawCurrentWord string      // raw source of current word (with quotes)
	Prefix         string      // wordbreak prefix up to cursor
	QuotingState   LexerState  // IN_WORD / QUOTING / QUOTING_ESCAPING / ESCAPING
	IsRedirect     bool        // true when completing a redirect target
	Pipeline       TokenSlice  // raw pipeline tokens (escape hatch)
}

func SplitForCompletion(s string, format Format) *CompletionContext
```

This replaces carapace's regex-based quoting detection in `zsh/action.go` (4 regexes on `RawValue`) with `ctx.QuotingState` from the tokenizer directly.

## JoinWith and QuoteWord

`JoinWith(s []string, format Format) string` joins words using the format's `QuoteWord` method. Each format implements `QuoteWord` with its shell's preferred quoting style:

| Format | Quoting style |
|--------|--------------|
| bash/zsh/oil/tcsh | double-quote wrapping with `\$ \` \" \\` escapes |
| fish | double-quote wrapping with `\" \$ \\` + newline escapes |
| elvish | single-quote wrapping with `''` for literal `'` |
| powershell | single-quote wrapping with `''` for literal `'` |
| nushell | double-quote wrapping with `\" \\` escapes |
| xonsh | Python single-quote wrapping with `\' \\` escapes |
| cmd | double-quote wrapping with `^"` for literal `"` |

`Join(s []string) string` delegates to `JoinWith(s, BashFormat())` for backward compatibility. The old v1 `Join` that used Go's `%#v` quoting is gone.

## Implemented Formats

10 formats are implemented, each in a `format_*.go` file:

| Format | File | Key features |
|--------|------|-------------|
| `BashFormat()` | `format_bash.go` | POSIX baseline, reads `COMP_WORDBREAKS` |
| `ZshFormat()` | `format_zsh.go` | RC_QUOTES (`''`→`'`), `NonEscapingQuoteEscapes` |
| `OilFormat()` | `format_oil.go` | bash-compatible (OSH) |
| `TcshFormat()` | `format_tcsh.go` | POSIX-family |
| `FishFormat()` | `format_fish.go` | `\'`/`\\` in single quotes, keyword operators |
| `ElvishFormat()` | `format_elvish.go` | `''` doubled-quote, `\` as bareword (`EscapeNotBareword`) |
| `PowershellFormat()` | `format_powershell.go` | backtick escape, `''`/`""` doubled-quotes |
| `NushellFormat()` | `format_nushell.go` | backtick-as-quote, `$'...'`/`$"..."` |
| `XonshFormat()` | `format_xonsh.go` | Python string prefixes, POSIX operators |
| `CmdFormat()` | `format_cmd.go` | caret escape, `"`-only, `&` separator |

### Deferred format features

These require multi-rune opener support not yet implemented:

- Nushell `r#'...'#` raw strings
- Xonsh triple-quotes (`'''...'''`)
- Cmd `REM`/`::` keyword comments
- PowerShell here-strings (`@'...'@`) and `--%` stop-parsing

The basic quote types (single, double, backtick) cover the vast majority of completion input. These deferred features are for completeness.

## State Machine Extensions

The v1 state machine is extended with three format-configurable behaviors (no new states added):

### NonEscapingQuoteEscapes (`''` doubled-quote)

When `NonEscapingQuoteEscapes()` returns true, the `QUOTING_STATE` handler peeks at the next rune on seeing `'`:
- If next is also `'` → consume both, emit one literal `'`, stay in `QUOTING_STATE`
- Else → close the quote (`IN_WORD_STATE`)

Also extends to `QUOTING_ESCAPING_STATE` for `""` doubled-quote (PowerShell).

Supported by: zsh, elvish, PowerShell, fish.

### NonEscapingQuoteBackslashEscapes (`\'`/`\\` in single quotes)

When `NonEscapingQuoteBackslashEscapes()` returns true, the `QUOTING_STATE` handler treats `\` as an escape:
- `\'` → literal `'`, stay in `QUOTING_STATE`
- `\\` → literal `\`, stay in `QUOTING_STATE`

Only fish needs this.

### EscapeNotBareword (`\` as bareword)

When `EscapeNotBareword()` returns false, the `START_STATE` and `IN_WORD_STATE` handlers treat `\` as a regular word character instead of entering `ESCAPING_STATE`. The `\` still works as an escape inside double quotes (`QUOTING_ESCAPING_STATE`).

Only elvish needs this — `\` is a valid bareword character in elvish.

### Keyword operators (fish `and`/`or`/`not`)

When `KeywordOperators()` returns a non-nil map, the `tokenizer.Next()` method reclassifies `WORD_TOKEN`s whose `RawValue` matches a keyword as `WORDBREAK_TOKEN` with the mapped `WordbreakType`. This lets fish's bare-word operators split pipelines without operator runes.

## API Summary

```go
// Backward compatible (v1)
func Split(s string) (TokenSlice, error)    // defaults to BashFormat()
func Join(s []string) string                // defaults to BashFormat()

// New (v2)
func SplitWith(s string, format Format) (TokenSlice, error)
func SplitForCompletion(s string, format Format) *CompletionContext
func JoinWith(s []string, format Format) string

// Format constructors
func BashFormat() Format
func ZshFormat() Format
func OilFormat() Format
func TcshFormat() Format
func FishFormat() Format
func ElvishFormat() Format
func PowershellFormat() Format
func NushellFormat() Format
func XonshFormat() Format
func CmdFormat() Format
```

`Split(s)` delegates to `SplitWith(s, BashFormat())`, preserving v1 behavior. Existing carapace code using `Split` and `TokenSlice` methods works unchanged (the only breaking change is `Token.Index` → `Token.Span.Start`).

## Adding a New Shell Format

1. **Create `format_<shell>.go`** — implement the `Format` interface with a struct
2. **Configure the classifier** — map runes to `runeTokenClass` values (spaces, quotes, escape, comments, wordbreaks)
3. **Configure the operator grammar** — implement `ClassifyOperator()` mapping operator strings to `WordbreakType`
4. **Set the format flags** — `NonEscapingQuoteEscapes`, `NonEscapingQuoteBackslashEscapes`, `EscapeNotBareword`, `KeywordOperators` as needed
5. **Write tests** — `format_<shell>_test.go` covering quotes, escapes, operators, comments, edge cases (open quote at EOF, escape at EOF, adjacent quoted segments)

See [comparison.md](comparison.md) for the per-shell lexical rules and the `format-*.md` references for details.

## References

- `shlex.go` — tokenizer state machine, `Token`, `LexerState`, `Split`, `SplitWith`, `Join`, `JoinWith`
- `format.go` — `Format` interface, `Span`
- `completion.go` — `CompletionContext`, `SplitForCompletion`
- `quote.go` — per-shell `QuoteWord` implementations
- `tokenslice.go` — `TokenSlice` operations
- `wordbreak.go` — `WordbreakType`, `bashWordbreakType`, `BASH_WORDBREAKS`
- `format_*.go` — per-shell format implementations
- [comparison.md](comparison.md) — cross-shell lexical comparison
- `format-*.md` — per-shell lexical format references

## Related Skills

- **bash**, **zsh**, **fish**, **elvish**, **nushell**, **powershell**, **xonsh**, **tcsh**, **oil**, **cmd-clink** skills — broader shell internals (completion systems, execution, startup)
- **carapace-dev** skill → `references/shell.md` — how carapace formats completion output per shell
