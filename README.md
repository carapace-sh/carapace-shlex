# carapace-shlex

[![PkgGoDev](https://pkg.go.dev/badge/github.com/carapace-sh/carapace-shlex)](https://pkg.go.dev/github.com/carapace-sh/carapace-shlex)
[![GoReportCard](https://goreportcard.com/badge/github.com/carapace-sh/carapace-shlex)](https://goreportcard.com/report/github.com/carapace-sh/carapace-shlex)
[![Coverage Status](https://coveralls.io/repos/github/github.com/carapace-sh/carapace-shlex/badge.svg?branch=master)](https://coveralls.io/github.com/carapace-sh/carapace-shlex?branch=master)

A command-line lexer that splits and re-joins command lines with quotation-state information for shell completion. Fork of [go-shlex](https://github.com/google/shlex).

V1 was POSIX-only. V2 supports multiple shell formats (including non-POSIX) via the `Format` interface.

[![asciicast](https://asciinema.org/a/599580.svg)](https://asciinema.org/a/599580)

## Usage

### Split (backward compatible)

```go
tokens, err := shlex.Split(`echo "hello world" | grep foo`)
```

Defaults to the bash (POSIX) format. Returns a `TokenSlice` with typed tokens including quotation state.

### SplitWith (format-specific)

```go
tokens, err := shlex.SplitWith(`echo 'it''s'`, shlex.ElvishFormat())
```

Use a specific shell format for lexing.

### SplitForCompletion

```go
ctx := shlex.SplitForCompletion(`echo foo | grep hel`, shlex.BashFormat())
// ctx.CurrentWord   = "hel"
// ctx.Words         = ["grep", "hel"]
// ctx.QuotingState  = IN_WORD_STATE
// ctx.Prefix        = ""
// ctx.IsRedirect    = false
```

Returns a `CompletionContext` with the current word, quoting state, prefix, pipeline words, and redirect detection — replacing the manual `tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken()` chains.

## Supported Formats

| Format | Function | Key features |
|--------|----------|-------------|
| Bash | `BashFormat()` | POSIX baseline, reads `COMP_WORDBREAKS` |
| Zsh | `ZshFormat()` | RC_QUOTES (`''`→`'`) |
| Oil | `OilFormat()` | bash-compatible (OSH) |
| Tcsh | `TcshFormat()` | POSIX-family |
| Fish | `FishFormat()` | `\'`/`\\` in single quotes, keyword operators (`and`/`or`) |
| Elvish | `ElvishFormat()` | `''` doubled-quote, `\` as bareword |
| PowerShell | `PowershellFormat()` | backtick escape, `''`/`""` doubled-quotes |
| Nushell | `NushellFormat()` | backtick-as-quote, `$'...'`/`$"..."` |
| Xonsh | `XonshFormat()` | Python string prefixes, POSIX operators |
| Cmd | `CmdFormat()` | caret escape, `"`-only, `&` separator |

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
