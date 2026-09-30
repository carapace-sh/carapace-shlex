# carapace-shlex

[![PkgGoDev](https://pkg.go.dev/badge/github.com/carapace-sh/carapace-shlex)](https://pkg.go.dev/github.com/carapace-sh/carapace-shlex)
[![Coverage Status](https://coveralls.io/repos/github/carapace-sh/carapace-shlex/badge.svg?branch=master)](https://coveralls.io/github/carapace-sh/carapace-shlex?branch=master)

Fork of [go-shlex](https://github.com/google/shlex) aimed to enable completion of complex commands passed as single argument with [Split] in [carapace].

[![asciicast](https://asciinema.org/a/599580.svg)](https://asciinema.org/a/599580)

> **v2** is still undergoing strong changes, stick to **v1** for stability.

## Usage

### Split

```go
tokens, err := shlex.Split(`echo "hello world" | grep foo`, shlex.Bash)
tokens, err = shlex.Split(`echo 'it''s'`, shlex.Elvish)
tokens, err = shlex.Split(`echo foo`, shlex.Default) // Default ("") is the bash format
```

Returns a `TokenSlice` with typed tokens including quotation state. Unknown format names are rejected.

### Complete

```go
ctx := shlex.Complete(`echo foo | grep hel`, shlex.Bash)
// ctx.CurrentWord   = "hel"
// ctx.Words         = ["grep", "hel"]
// ctx.QuotingState  = IN_WORD_STATE
// ctx.Prefix        = ""
// ctx.IsRedirect    = false
```

Returns a `CompletionContext` — current word, quoting state, prefix, pipeline words, and redirect detection; `ctx.Span` locates the current word in the input and `ctx.Tokens` carries the raw tokens. This replaces the manual `tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken()` chains of v1.

### Variable references

```go
ctx := shlex.Complete(`echo "text$HO`, shlex.Bash)
// ctx.Variable = &shlex.Variable{Name: "HO"} (nil when the word does not end in a reference)
// ctx.Variable.Insert("HOME")                 // "text$HOME" - the word with the completed name
```

`Variable` is detected on the lexer's final word — quote state, escapes, and the sigil forms are handled by the format (bash, zsh, tcsh, fish, elvish, nushell, and xonsh implement detection; `$` inside single quotes or after `\` is literal where the shell says so, closed expansions like `${HOME}` are not references). `Insert` composes the replacement: it preserves the raw text the shell's completion interface replaces (bash's naive `COMP_WORDS` split, e.g. `"text $HO` yields `$HOME`) and reconstructs the sigil form (`$HOME` or `${HOME}`).

### Quoting for insertion

```go
ctx := shlex.Complete(`echo "he`, shlex.Bash)
// the completion candidate for the current word (`he`) is `hello world`:
ctx.Quote(`hello world`) // `"hello world"` - replaces the raw word `"he`
```

`Quote` returns the value quoted so it can replace the raw current word: an open quote is closed with the format's own escape rules (per shell: bash `'"'"'`, fish `\'`, zsh/elvish `''`, cmd `""`, PowerShell backtick), and barewords are quoted as complete words. Stop-parsing mode (PowerShell `--%`) passes values through raw.

### Join

```go
shlex.Join([]string{"echo", "hello world"}, shlex.Bash)
// `echo "hello world"`
```

Quotes and escapes words into a single command line using the format's quoting rules.

## Supported Formats

| Format | Constant | Key features |
|--------|----------|-------------|
| Bash | `shlex.Bash` | POSIX baseline, reads `COMP_WORDBREAKS` |
| Zsh | `shlex.Zsh` | RC_QUOTES (`''`→`'`) |
| Oil | `shlex.Oil` | bash-compatible (OSH) |
| Tcsh | `shlex.Tcsh` | POSIX-family |
| Fish | `shlex.Fish` | `\'`/`\\` in single quotes, keyword operators (`and`/`or`) |
| Elvish | `shlex.Elvish` | `''` doubled-quote, `\` as bareword |
| PowerShell | `shlex.Powershell` | backtick escape, `''`/`""` doubled-quotes |
| Nushell | `shlex.Nushell` | backtick-as-quote, `$'...'`/`$"..."` |
| Xonsh | `shlex.Xonsh` | Python string prefixes, POSIX operators |
| Cmd | `shlex.Cmd` | caret escape, `"`-only, `&` separator |

## CLI

```
go run ./cmd/carapace-shlex --format fish --completion-context "echo foo and grep hel"
```

Flags:
- `--format` — shell format (bash, zsh, fish, elvish, nushell, powershell, xonsh, tcsh, oil, cmd)
- `--completion-context` — output `CompletionContext` as JSON
- `--current-pipeline` — show current pipeline only
- `--filter-redirects` — filter redirect operators
- `--words` — combine adjoining tokens
- `--wordbreak-prefix` — show wordbreak prefix
- `--join` — re-join words

## Token Model

```go
type Token struct {
    Type           TokenType    // WORD_TOKEN, WORDBREAK_TOKEN, etc.
    Value          string       // dequoted value
    RawValue       string       // raw source text including quotes
    Span           Span         // rune offsets {Start, End}
    State          LexerState   // quotation state after this token
    WordbreakType  WordbreakType // operator type (pipe, redirect, etc.)
    WordbreakIndex int          // index of last opening quote in Value
}
```

## Migrating from v1

- The module path gained `/v2` (`github.com/carapace-sh/carapace-shlex/v2`)
- `Split(s)` and `Join(s)` now take the format: `shlex.Split(s, shlex.Bash)` — the format is no longer implied
- `Token.Index` became `Token.Span` (rune offsets, `End` exclusive; `Index` was equivalent to `Span.Start`)
- Completion consumers replace the manual `tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken()` composition with `shlex.Complete(s, format)`; the current word's position is `ctx.Span` instead of `CurrentToken().Index`
- Formats are a closed set of constants — new lexing behavior lands as a format in this repository, not as an outside implementation

[Split]:https://carapace-sh.github.io/carapace/carapace/action/split.html
[carapace]:https://github.com/carapace-sh/carapace
