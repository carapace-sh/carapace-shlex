# V2 Architecture — The Common Token Model

How the v2 lexer is structured: a common token model and tokenizer state machine that each shell format plugs into via the `Format` interface. V1 was POSIX-only; v2 generalizes to multiple shell formats (including non-POSIX).

> **Source of truth**: `shlex.go` (state machine, `Token`, `Split`), `format.go` (`Format` constants, `formatImpl` interface, `Span`), `completion.go` (`CompletionContext`, `Complete`), `tokenslice.go` (`TokenSlice` operations), `wordbreak.go` (`WordbreakType`), `format_*.go` (per-shell formats). For how shells differ lexically, see [comparison.md](comparison.md).

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

V2 keeps the proven tokenizer state machine and `TokenSlice` operations but makes the **rune classification**, **operator grammar**, and **quote behavior** configurable per shell format. A `Format` is a string constant naming a shell (`shlex.Bash`, ...); the behavior is a small struct implementing the unexported `formatImpl` interface, registered in the `formatImpls` map in `format.go`:

```go
// format.go
type formatImpl interface {
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
	// Used by Join. Each format uses its shell's preferred quoting style.
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
| `Split(s)` / `Split(s, format)` | Entry point — lexes a string into tokens |
| `Words()` | Merges adjoining tokens (contiguous `Span`) into single words |
| `CurrentPipeline()` | Returns the last pipeline (splits on `\|`, `&&`, `;`, etc.) |
| `pipelines()` | Splits into all pipelines (used by `CurrentPipeline`) |
| `FilterRedirects()` | Removes redirect operators and their targets |
| `WordbreakPrefix()` | Extracts the completion prefix up to the cursor |
| `currentToken()` | Returns the last token |
| `Strings()` | Returns word values as `[]string` |
| `Equal(other *Token)` | Reports whether two tokens are equal |

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

The `Complete` function provides a structured completion context, replacing the manual `tokens.CurrentPipeline().FilterRedirects().Words().currentToken()` chains that carapace used with v1:

```go
// completion.go
type CompletionContext struct {
	Words             []string     // current pipeline's words (redirects filtered)
	CurrentWord       string       // word at cursor (dequoted)
	RawCurrentWord    string       // raw source of current word (with quotes)
	Prefix            string       // wordbreak prefix up to cursor
	QuotingState      LexerState   // IN_WORD / QUOTING / QUOTING_ESCAPING / ESCAPING
	IsRedirect        bool         // true when completing a redirect target
	InLambdaParams    bool         // elvish lambda parameter list (omitempty)
	Variable       *Variable // variable reference ending the word (omitempty)
	Span              Span         // current word's position in the input
	Tokens            TokenSlice   // raw tokens of the whole input (escape hatch)
	SubstitutionDepth int          // unclosed substitution scopes (omitempty)
}

func Complete(s string, format Format) *CompletionContext
```

`Complete` never fails: lexer errors and unknown format names yield an empty context with `START_STATE`.

This replaces carapace's regex-based quoting detection in `zsh/action.go` (4 regexes on `RawValue`) with `ctx.QuotingState` from the tokenizer directly.

## Join and QuoteWord

`Join(s []string, format Format) string` joins words using the format's `QuoteWord` method. Each format implements `QuoteWord` with its shell's preferred quoting style:

| Format | Quoting style |
|--------|--------------|
| bash/zsh/oil/tcsh | double-quote wrapping with `\$ \` \" \\` escapes |
| fish | double-quote wrapping with `\" \$ \\` + newline escapes |
| elvish | single-quote wrapping with `''` for literal `'` |
| powershell | single-quote wrapping with `''` for literal `'` |
| nushell | double-quote wrapping with `\" \\` escapes |
| xonsh | Python single-quote wrapping with `\' \\` escapes |
| cmd | double-quote wrapping with `^"` for literal `"` |

`Join(s []string, format Format) string` joins words using the format's `QuoteWord` method; `shlex.Default` selects the bash rules.

## Implemented Formats

11 formats are implemented, each in a `format_*.go` file:

| Format | File | Key features |
|--------|------|-------------|
| `shlex.Bash` | `format_bash.go` | POSIX baseline, reads `COMP_WORDBREAKS` |
| `shlex.Zsh` | `format_zsh.go` | RC_QUOTES (`''`→`'`), `NonEscapingQuoteEscapes` |
| `shlex.Oil` | `format_oil.go` | bash-compatible (OSH) |
| `shlex.Tcsh` | `format_tcsh.go` | POSIX-family |
| `shlex.Fish` | `format_fish.go` | `\'`/`\\` in single quotes, keyword operators |
| `shlex.Elvish` | `format_elvish.go` | `''` doubled-quote, `\` as bareword (`EscapeNotBareword`) |
| `shlex.Powershell` | `format_powershell.go` | backtick escape, `''`/`""` doubled-quotes |
| `shlex.Nushell` | `format_nushell.go` | backtick-as-quote, `$'...'`/`$"..."` |
| `shlex.Xonsh` | `format_xonsh.go` | Python string prefixes, POSIX operators |
| `shlex.Cmd` | `format_cmd.go` | caret escape, `"`-only, `&` separator |
| `shlex.Hilbish` | `format_hilbish.go` | POSIX sh via snail (mvdan.cc/sh), fixed wordbreaks, no `<<<`/`\|&`/`&>` |

### Deferred format features

These require multi-rune opener support not yet implemented:

- Nushell `r#'...'#` raw strings
- Xonsh triple-quotes (`'''...'''`)
- Cmd `REM`/`::` keyword comments
- PowerShell here-strings (`@'...'@`) and `--%` stop-parsing
- Zsh `WORDCHARS` env var (characters that are NOT word breaks)

The basic quote types (single, double, backtick) cover the vast majority of completion input. These deferred features are for completeness.

### Implemented cmd.exe features

The cmd format now implements:
- Line continuation (`^` + newline, via `lineContinuationEscaper`)
- `(` `)` grouping operators (wordbreak runes, `WORDBREAK_UNKNOWN`)
- `,` as word delimiter (space class, not a command separator)
- Numeric stream redirects `2>`, `2>>`, `2>&1`, `1>&2` (via `postProcessor`)
- `^` is literal inside double quotes (via `EscapeNotInEscapingQuote` flag — cmd's Phase 2 parser only treats `"` and `<LF>` as special inside quotes)
- `cmdQuoteWord` uses close-quote/`^"`/reopen-quote to embed literal `"` (no escape mechanism exists inside cmd double quotes)

## State Machine Extensions

The v1 state machine is extended with three format-configurable behaviors (no new states added):

### NonEscapingQuoteEscapes (`''` doubled-quote)

When `NonEscapingQuoteEscapes()` returns true, the `QUOTING_STATE` handler peeks at the next rune on seeing `'`:
- If next is also `'` → consume both, emit one literal `'`, stay in `QUOTING_STATE`
- Else → close the quote (`IN_WORD_STATE`)

Also extends to `QUOTING_ESCAPING_STATE` for `""` doubled-quote (PowerShell).

Supported by: zsh, elvish, PowerShell, fish.

### NonEscapingQuoteBackslashEscapes (`\'`/`\\` in single quotes)

When `NonEscapingQuoteBackslashEscapes()` returns true, the `QUOTING_STATE` handler treats `\` as a limited escape:
- `\'` → literal `'`, stay in `QUOTING_STATE`
- `\\` → literal `\`, stay in `QUOTING_STATE`
- `\X` (X ≠ `'`, `\`) → literal `\X` (both chars emitted), stay in `QUOTING_STATE`
- `\` at EOF → literal `\`

Requires `NonEscapingQuoteEscapes()` to also be true. Only fish needs this.

### Double-quote escape limitation

The `QUOTING_ESCAPING_STATE` handler unconditionally enters `ESCAPING_QUOTED_STATE` on any escape rune, consuming the backslash and emitting the next rune literally. This means `\n` inside `"..."` produces `n` (backslash consumed), not `hello\nworld`.

In real shells, `\` inside double quotes is only special before specific characters:
- **bash/zsh/oil/tcsh**: `$`, `` ` ``, `"`, `\`, newline — `\n` should be literal `hello\nworld`
- **fish**: `"`, `$`, `\`, newline — `\n` should be literal
- **nushell**: `"`, `\` — `\$` should be literal
- **PowerShell**: backtick (not `\`) — `\` is always literal

This is a known limitation of the lexer. For completion purposes it doesn't affect word splitting or quote-state tracking — the `Value` field may differ from the shell's actual dequoting, but the `State` and word boundaries are correct. The `EscapingQuoteEscapeChars` and `escapingQuoteUnescaper` interfaces narrow this behavior for formats that need it.

### Line continuation (`\<newline>`)

POSIX shells (bash, zsh, tcsh, oil), fish, and xonsh implement `lineContinuationEscaper`. When the escape character (backslash) is followed by `\n` or `\r`, both the escape char and the newline are consumed — they are not added to the token's `Value` or `RawValue`. This applies:
- Outside quotes (`ESCAPING_STATE`): the word continues on the next line
- Inside double quotes (`ESCAPING_QUOTED_STATE`): the string continues on the next line

For CRLF input, the `\r` is consumed and an optional following `\n` is also consumed.

PowerShell and cmd implement the same interface for their escape characters (backtick and caret respectively).

### Line continuation whitespace (`^<newline>` in elvish)

Elvish uses `^` (not `\`) followed by newline as a line continuation. Unlike `lineContinuationEscaper` (which concatenates the word across lines), elvish's `^<newline>` acts as **whitespace** — it ends the current word and the next word starts on the next line. The `lineContinuationWhitespace` interface handles this: when the line-continuation char is seen in `IN_WORD_STATE` or `START_STATE`, the tokenizer peeks ahead. If the next rune is `\n` or `\r`, both the char and newline are consumed and the current word ends (like a space). `^` without a following newline is a regular bareword character.

Supported by: elvish only.

### EscapeNotBareword (`\` as bareword)

When `EscapeNotBareword()` returns false, the `START_STATE` and `IN_WORD_STATE` handlers treat `\` as a regular word character instead of entering `ESCAPING_STATE`. The `\` still works as an escape inside double quotes (`QUOTING_ESCAPING_STATE`).

Only elvish needs this — `\` is a valid bareword character in elvish.

### Keyword operators (fish `and`/`or`/`not`)

When `KeywordOperators()` returns a non-nil map, the `tokenizer.Next()` method reclassifies `WORD_TOKEN`s whose `RawValue` matches a keyword as `WORDBREAK_TOKEN` with the mapped `WordbreakType`. This lets fish's bare-word operators split pipelines without operator runes.

## API Summary

```go
// Backward compatible (v1)
func Split(s string, format Format) (TokenSlice, error)
func Join(s []string, format Format) string

// New (v2)
func Split(s string, format Format) (TokenSlice, error)
func Complete(s string, format Format) *CompletionContext
func Join(s []string, format Format) string

// Format constants
const (
	Default Format = "" // resolves to Bash
	Bash    Format = "bash"
	// ... one per format, see format.go
)
```

`Split(s, Default)` matches v1's default bash lexing. `Token.Index` became `Token.Span` (rune offsets), and `Split`/`Join` now take the format explicitly.

## Adding a New Shell Format

1. **Create `format_<shell>.go`** — implement the `formatImpl` interface with a struct
2. **Configure the classifier** — map runes to `runeTokenClass` values (spaces, quotes, escape, comments, wordbreaks)
3. **Configure the operator grammar** — implement `ClassifyOperator()` mapping operator strings to `WordbreakType`
4. **Set the format flags** — `NonEscapingQuoteEscapes`, `NonEscapingQuoteBackslashEscapes`, `EscapeNotBareword`, `KeywordOperators` as needed
5. **Write tests** — `format_<shell>_test.go` covering quotes, escapes, operators, comments, edge cases (open quote at EOF, escape at EOF, adjacent quoted segments)

See [comparison.md](comparison.md) for the per-shell lexical rules and the `format-*.md` references for details.

## References

- `shlex.go` — tokenizer state machine, `Token`, `LexerState`, `Split`, `Split`, `Join`, `Join`
- `format.go` — `Format` constants, `formatImpl` interface, `Span`
- `completion.go` — `CompletionContext`, `Complete`
- `quote.go` — per-shell `QuoteWord` implementations
- `tokenslice.go` — `TokenSlice` operations
- `wordbreak.go` — `WordbreakType`, `bashWordbreakType`, `BASH_WORDBREAKS`
- `format_*.go` — per-shell format implementations
- [comparison.md](comparison.md) — cross-shell lexical comparison
- `format-*.md` — per-shell lexical format references

## Related Skills

- **bash**, **zsh**, **fish**, **elvish**, **nushell**, **powershell**, **xonsh**, **tcsh**, **oil**, **cmd-clink** skills — broader shell internals (completion systems, execution, startup)
- **carapace-dev** skill → `references/shell.md` — how carapace formats completion output per shell
