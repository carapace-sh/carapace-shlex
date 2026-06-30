---
name: shlex
description: >
  Use when working with the carapace-shlex v2 lexer — a command-line lexer that splits and
  re-joins command lines with quotation-state information for shell completion. Covers the
  common token model, the Format interface, per-shell lexical formats (quoting, escaping,
  word breaks, operators, comments), CompletionContext, and how to add a new shell format.
  Triggers on: "shlex", "shlex v2", "carapace-shlex", "shell lexer", "command line lexer",
  "quotation state", "wordbreak", "WORDBREAK", "COMP_WORDBREAKS", "TokenSlice", "LexerState",
  "TokenType", "tokenizer", "Split", "SplitWith", "SplitForCompletion", "CompletionContext",
  "Format", "BashFormat", "ZshFormat", "FishFormat", "ElvishFormat", "PowershellFormat",
  "NushellFormat", "XonshFormat", "TcshFormat", "OilFormat", "CmdFormat",
  "Span", "shell format", "lexical format", "POSIX shell lexing", "non-POSIX shell lexing".
user-invocable: true
---

# carapace-shlex v2 — Multi-Shell Command-Line Lexer

In-depth reference for the v2 lexer in [carapace-shlex](https://github.com/carapace-sh/carapace-shlex), a fork of [go-shlex](https://github.com/google/shlex) that splits and re-joins command lines while tracking quotation state for shell completion. V1 was POSIX-only; v2 supports 10 shell formats (including non-POSIX) via the `Format` interface.

## Data Flow

```
command line string
  → SplitWith(s, format) or SplitForCompletion(s, format)
    → format.Classifier() (rune → rune class)
      → tokenizer state machine (rune classes → tokens, format flags for quote behavior)
        → TokenSlice (typed tokens with Span and quotation state)
          → CompletionContext (current word, prefix, quoting state, pipeline)
```

Each shell format plugs its rune classifications, operator grammar, and quote-behavior flags into the common tokenizer state machine via the `Format` interface. The token model, `TokenSlice` operations, and `CompletionContext` are shared across all formats.

## Sub-Resources

Load the reference that matches your task. When in doubt, load multiple references.

| Keywords | Reference |
|----------|----------|
| v2 architecture, Format interface, Span, Token, TokenSlice, tokenizer, tokenClassifier, rune class, runeTokenClass, Split, SplitWith, SplitForCompletion, CompletionContext, QuotingState, IsRedirect, NonEscapingQuoteEscapes, NonEscapingQuoteBackslashEscapes, EscapeNotBareword, KeywordOperators, ClassifyOperator, QuoteWord, JoinWith, bashWordbreakType, Pipelines, Equal, double-quote escape limitation, adding a new format, format registration, implemented formats table, deferred features | [references/architecture.md](references/architecture.md) |
| cross-shell comparison, quoting comparison, escape character comparison, word break comparison, operator comparison, comment syntax comparison, string interpolation, metacharacters table, POSIX vs non-POSIX, shell family | [references/comparison.md](references/comparison.md) |
| bash format, POSIX lexing, single quotes, double quotes, backslash escape, ANSI-C quoting, $'...', #" comment, COMP_WORDBREAKS, pipe, redirect, list operators, &&, \|\|, ;, &, \|, <, >, >>, <<<, wordbreaks | [references/format-bash.md](references/format-bash.md) |
| zsh format, zsh lexing, single quotes, double quotes, $'...' ANSI-C, RC_QUOTES, backslash escape, # comment, WORDCHARS, FIGNORE, pipe, redirect, list operators, process substitution, =<(...), glob qualifiers | [references/format-zsh.md](references/format-zsh.md) |
| fish format, fish lexing, fish quotes, single quotes, double quotes, backslash escape, \' in single quotes, fish word splitting, no IFS, # comment, fish operators, pipe, redirect, ; separator, and, or, not, fish metacharacters | [references/format-fish.md](references/format-fish.md) |
| elvish format, elvish lexing, bareword, single quotes, '' doubled, double quotes, escape sequences, no word splitting, # comment, elvish operators, pipe, redirection, no POSIX operators | [references/format-elvish.md](references/format-elvish.md) |
| nushell format, nushell lexing, single quotes, double quotes, raw string, r#'...'#, backtick, bare word, string interpolation, $'...', $\"...\", # comment, nushell metacharacters, pipe, semicolon | [references/format-nushell.md](references/format-nushell.md) |
| powershell format, powershell lexing, single quotes, '' doubled, double quotes, backtick escape, here-string, @\" \"@, @' '@, stop-parsing --%, # comment, powershell metacharacters, pipe, ; separator | [references/format-powershell.md](references/format-powershell.md) |
| xonsh format, xonsh lexing, Python/shell hybrid, subprocess syntax, string literals, p-strings, f-strings, raw strings, quoting, word splitting, glob, tilde, # comment, xonsh operators | [references/format-xonsh.md](references/format-xonsh.md) |
| tcsh format, csh lexing, single quotes, double quotes, backslash, backslash_quote, $'...' ANSI-C, backtick command substitution, # comment, ! history expansion, tcsh operators, pipe, redirect, ; separator | [references/format-tcsh.md](references/format-tcsh.md) |
| oil format, OSH lexing, YSH lexing, single quotes, double quotes, ANSI-C quoting, YSH strings, r'...', triple-quoted, # comment, oil operators, pipe, redirect, simple word evaluation | [references/format-oil.md](references/format-oil.md) |
| cmd format, cmd.exe lexing, clink lexing, caret escape, ^, double quotes, no single quotes, %variable%, REM, :: comment, command separator, & separator, pipe, redirect, cmd metacharacters | [references/format-cmd.md](references/format-cmd.md) |

## Quick Guide

- **How does the v2 lexer work?** → [references/architecture.md](references/architecture.md)
- **How do I add a new shell format?** → [references/architecture.md](references/architecture.md)
- **What is the common token model?** → [references/architecture.md](references/architecture.md)
- **How do shells differ lexically?** → [references/comparison.md](references/comparison.md)
- **Which shells are POSIX vs non-POSIX?** → [references/comparison.md](references/comparison.md)
- **What are the word break characters per shell?** → [references/comparison.md](references/comparison.md)
- **How does bash quoting work for the lexer?** → [references/format-bash.md](references/format-bash.md)
- **How does zsh differ from bash lexically?** → [references/format-zsh.md](references/format-zsh.md)
- **How does fish tokenize command lines?** → [references/format-fish.md](references/format-fish.md)
- **How does elvish tokenize barewords and quotes?** → [references/format-elvish.md](references/format-elvish.md)
- **How does nushell handle its string types?** → [references/format-nushell.md](references/format-nushell.md)
- **How does PowerShell's backtick escaping work?** → [references/format-powershell.md](references/format-powershell.md)
- **How does xonsh mix Python and shell lexing?** → [references/format-xonsh.md](references/format-xonsh.md)
- **How does tcsh differ from POSIX lexing?** → [references/format-tcsh.md](references/format-tcsh.md)
- **How do OSH and YSH differ lexically?** → [references/format-oil.md](references/format-oil.md)
- **How does cmd.exe/clink tokenize?** → [references/format-cmd.md](references/format-cmd.md)

## Cross-Project References

The per-shell lexical format references here describe only what a **lexer** needs: quote characters, escape semantics, word delimiters, operators, and comment syntax. For the broader shell internals (completion systems, execution model, startup, editor), use the dedicated shell skills:

- **bash** skill — bash completion, Readline, quoting/expansion, execution, startup
- **zsh** skill — compsys, ZLE, expansion/quoting, startup
- **fish** skill — fish completion, editor, language, startup
- **elvish** skill — elvish completion, editor, styling, language, startup
- **nushell** skill — nushell completion, Reedline, quoting, types, externs, config
- **powershell** skill — PowerShell completion, PSReadLine, styling, language, startup
- **xonsh** skill — xonsh completion, prompt-toolkit, language/execution, startup
- **tcsh** skill — tcsh completion, editor, quoting/expansion, execution, startup
- **oil** skill — Oil completion, line editing, quoting/expansion, execution, startup
- **cmd-clink** skill — clink argmatcher, completion, line editing, cmd integration

For how carapace formats completion *output* per shell (snippet, value quoting, nospace), see the **carapace-dev** skill → `references/shell.md`.
