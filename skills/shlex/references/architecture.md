# V2 Architecture — The Common Token Model

How the v2 lexer is structured: a common token model and tokenizer state machine that each shell format plugs into via rune classification and operator sets. V1 is POSIX-only; v2 generalizes to multiple shell formats.

> **Source of truth**: `shlex.go`, `tokenslice.go`, `wordbreak.go` in the repo root. For how shells differ lexically, see [comparison.md](comparison.md).

## V1 Recap (POSIX-Only)

V1 is a single lexer hardcoded to POSIX shell lexing. The rune classes and operator set are fixed:

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

The classifier (`newDefaultClassifier`) reads `COMP_WORDBREAKS` from the environment and merges any custom wordbreak runes that aren't already classified. The tokenizer state machine (`scanStream`) is a single hardcoded switch over `LexerState`.

This works for bash and bash-like shells (zsh, oil OSH) but cannot express:

- **Non-POSIX quote types** — fish's `\'` inside single quotes, nushell's `r#'...'#` raw strings and backtick strings, PowerShell's backtick escape and here-strings, elvish's `''` doubled-quote escaping, xonsh's Python string literals.
- **Non-POSIX operators** — fish's `and`/`or`/`not` keyword operators, ion's `|>` and `=>` pipe/redirect operators, cmd's `&` command separator and `^` escape.
- **Different comment semantics** — fish `#`, PowerShell `#` (and `<# #>` block comments), cmd `REM`/`::`.
- **Different escape characters** — PowerShell uses backtick (`` ` ``) instead of backslash; cmd uses caret (`^`).

## V2 Goal

V2 keeps the proven tokenizer state machine and `TokenSlice` operations but makes the **rune classification** and **operator set** configurable per shell format. A format is a small struct that declares:

- which runes are spaces, quotes, escapes, comments, word breaks
- which quote types support escaping (double-quote-like) vs not (single-quote-like)
- the operator grammar (multi-char operators like `>>`, `&&`, `||`, `|>`)
- comment termination rules

The tokenizer state machine and token model stay common, so `Split`, `Words`, `CurrentPipeline`, `FilterRedirects`, and `WordbreakPrefix` work identically across all formats.

## The Common Token Model

These types are shared across all formats and unchanged from v1 in spirit:

### TokenType

A top-level token classification:

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

The state machine tracks quotation state so that completion can know whether the cursor is inside an open quote:

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

### Token

```go
type Token struct {
	Type           TokenType
	Value          string      // the processed value (quotes/escapes removed)
	RawValue       string      // the raw source text including quote chars
	Index          int         // rune index in the input stream
	State          LexerState  // state after emitting this token
	WordbreakType  WordbreakType `json:",omitempty"`
	WordbreakIndex int         // index of last opening quote in Value
}
```

Key fields for completion:

- **`Value`** — the dequoted word value. Completion matches against this.
- **`RawValue`** — the original source substring. Used to detect quotation state (e.g., zsh's `quoteValue` reads `RawValue` to decide which replacer to use).
- **`State`** — whether the word ended inside an open quote.
- **`WordbreakIndex`** — set when entering a quote, so `WordbreakPrefix()` can extract the prefix before the last opening quote.

## TokenSlice Operations

These are format-agnostic and work on the token stream produced by any format's tokenizer:

### `Split(s string) (TokenSlice, error)`

Entry point. Creates a lexer over the string and collects all tokens.

### `Words() TokenSlice`

Combines adjoining tokens (tokens whose `RawValue` ranges are contiguous in the source) into single words. A word like `foo"bar"'baz'` is three tokens in the raw stream but one word after `Words()`. The merged token's `State` is taken from the last segment.

```go
// tokenslice.go
func (t TokenSlice) Words() TokenSlice {
	words := make(TokenSlice, 0)
	for index, token := range t {
		switch {
		case index == 0:
			words = append(words, token)
		case t[index-1].adjoins(token):
			words[len(words)-1].Value += token.Value
			words[len(words)-1].RawValue += token.RawValue
			words[len(words)-1].State = token.State
		default:
			words = append(words, token)
		}
	}
	return words
}
```

### `CurrentPipeline() TokenSlice`

Splits the token slice on pipeline-delimiter wordbreaks (`|`, `||`, `&`, `;`, `&&`, etc.) and returns the last pipeline. This lets completion focus on the command currently being typed.

### `FilterRedirects() TokenSlice`

Removes redirect operators (`<`, `>`, `>>`, `<<<`, etc.) and their attached file-descriptor prefixes (e.g., the `2` in `2>`). Completion should not treat redirect targets as command arguments.

### `WordbreakPrefix() string`

Extracts the completion prefix: the text from the last wordbreak (or last opening quote) up to the cursor. This mirrors how bash determines the partial word to complete. Special handling:

- When the last token's state is a quoting state, the prefix starts at `WordbreakIndex` (the position of the last opening quote).
- `@` is a wordbreak but is **not** included in the prefix (bash quirk).

## WordbreakType

Operators are classified so that `CurrentPipeline` and `FilterRedirects` can decide what's a pipeline delimiter vs a redirect vs other:

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

`wordbreakType(t Token)` maps a token's `RawValue` to its type. **In v2, the operator grammar is per-format** — POSIX shells share the bash operator set, but ion uses `|>` for pipes and `=>` for redirection, fish uses keyword operators (`and`, `or`), and cmd uses `&` as a command separator.

The `IsPipelineDelimiter()` and `IsRedirect()` predicates drive `CurrentPipeline` and `FilterRedirects` respectively.

## Adding a New Shell Format

A v2 format is a configuration of the common tokenizer. To add one:

### 1. Identify the lexical rules

Determine for the target shell:

| Concern | Questions |
|---------|-----------|
| **Spaces** | Which runes delimit words? (Usually ` \t\r\n`.) |
| **Quotes** | Which quote characters open/close strings? Which support escaping (double-quote-like) and which are literal (single-quote-like)? |
| **Escape** | What is the escape character? (Backslash `\` for POSIX; backtick `` ` `` for PowerShell; caret `^` for cmd.) |
| **Comments** | What starts a comment? How does it terminate? (`#` to end-of-line for most; `REM`/`::` for cmd.) |
| **Word breaks / operators** | Which operator characters break words? What multi-char operators exist (`>>`, `&&`, `||>`, ...)? |
| **Non-POSIX string types** | Does the shell have string types the tokenizer must track? (nushell `r#'...'#`, backtick strings; PowerShell here-strings; elvish `''`.) |

See [comparison.md](comparison.md) for a per-shell summary and the `format-*.md` references for details.

### 2. Configure the classifier

Provide a rune classifier that maps runes to `runeTokenClass` values. V2 makes the hardcoded rune sets in v1 (`spaceRunes`, `escapingQuoteRunes`, etc.) per-format parameters.

### 3. Configure the operator grammar

Provide the set of operator strings and their `WordbreakType` so that `wordbreakType()` and the predicates work. For shells with keyword operators (fish `and`/`or`), the tokenizer must recognize word-boundary-delimited keywords, not just operator runes.

### 4. Handle format-specific string types

If the shell has string types that don't fit the two-quote model (escaping vs non-escaping), extend the state machine or pre-classify. Examples:

- **Nushell raw strings** `r#'...'#` — the `#` inside the opener is part of the delimiter, not a comment.
- **PowerShell here-strings** `@"..."@` / `@'...'@` — multi-line, closing delimiter on its own line.
- **Elvish doubled single quote** `''` — inside single quotes, `''` is an escaped `'`, not a close-then-open.

These may require a small extension to the state machine or a format-specific scan routine.

### 5. Test against the shell's parser

Validate the tokenizer output against the real shell's word splitting where possible. Use the shell's `--headless` mode (oil), test harness, or manual checks. Edge cases to verify:

- Open quote at end of input (cursor inside quotes)
- Escape at end of input
- Empty word after an operator
- Adjacent quoted segments (`a"b"'c'`)
- Comment at end of input vs comment mid-line
- Operator runs (`||`, `>>`, `|>`)

## References

- `shlex.go` — tokenizer state machine, `Token`, `LexerState`, `Split`, `Join`
- `tokenslice.go` — `TokenSlice` operations
- `wordbreak.go` — `WordbreakType`, operator classification, `BASH_WORDBREAKS`
- [comparison.md](comparison.md) — cross-shell lexical comparison
- `format-*.md` — per-shell lexical format references

## Related Skills

- **bash**, **zsh**, **fish**, **elvish**, **nushell**, **powershell**, **xonsh**, **tcsh**, **oil**, **cmd-clink** skills — broader shell internals (completion systems, execution, startup)
- **carapace-dev** skill → `references/shell.md` — how carapace formats completion output per shell
