# carapace-shlex

[![PkgGoDev](https://pkg.go.dev/badge/github.com/carapace-sh/carapace-shlex)](https://pkg.go.dev/github.com/carapace-sh/carapace-shlex)
[![GoReportCard](https://goreportcard.com/badge/github.com/carapace-sh/carapace-shlex)](https://goreportcard.com/report/github.com/carapace-sh/carapace-shlex)
[![Coverage Status](https://coveralls.io/repos/github/github.com/carapace-sh/carapace-shlex/badge.svg?branch=master)](https://coveralls.io/github.com/carapace-sh/carapace-shlex?branch=master)

A command-line lexer that splits and re-joins command lines with quotation-state information for shell completion. Fork of [go-shlex](https://github.com/google/shlex).

V1 was POSIX-only. V2 supports multiple shell formats (including non-POSIX) via the `Format` interface.

[![asciicast](https://asciinema.org/a/599580.svg)](https://asciinema.org/a/599580)

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

Returns a `CompletionContext` with the current word, quoting state, prefix, pipeline words, and redirect detection; `ctx.Span` locates the current word in the input and `ctx.Tokens` carries the raw tokens. — replacing the manual `tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken()` chains.

### Variable references

```go
ctx := shlex.Complete(`echo "text$HO`, shlex.Bash)
// ctx.VariableRef          = &shlex.VariableRef{Name: "HO"}   (nil when not a reference)
// ctx.VariableRef.Replacement = "text$HO"                    (raw text the shell replaces)
```

`VariableRef` is detected on the lexer's final word — quote state, escapes, and the `${` form are handled by the format (`$` inside single quotes or after `\` is literal; closed expansions like `${HOME}` are not references). Its `Replacement` is the raw text the shell's completion interface replaces (bash's naive `COMP_WORDS` split, e.g. `"text $HO` yields `$HO`); insertion replaces the `Name` suffix with the completed name.

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

## Links

- [carapace](https://github.com/carapace-sh/carapace) — shell completion framework that uses this library
- [Split action](https://carapace-sh.github.io/carapace/carapace/action/split.html) — carapace action using `Split`
