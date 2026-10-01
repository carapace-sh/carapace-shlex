# Hilbish Lexical Format

Lexical rules a command-line lexer needs for hilbish. Hilbish is a Lua-based shell whose shell commands run through the **snail** interpreter (wrapping mvdan.cc/sh with the parser's default language variant, **LangBash**), so the sh mode follows the full **bash operator grammar**. The default `hybrid` runner may also execute input as **Lua code** (tries Lua first, falls back to sh).

> **Source of truth**: hilbish source — `golibs/snail/snail.go` (sh interpreter), `nature/runner.lua` (runner modes), `complete.go` (`charEscapeMap`, `escapeFilename`). For broader hilbish internals, use the **hilbish** skill. For cross-shell comparison, see [comparison.md](comparison.md). For bash-shared rules, see [format-bash.md](format-bash.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` |
| escape | `\` | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |

## Word Breaks

`HILBISH_WORDBREAKS = "><;|&()"` — the sh metacharacters handled by the interpreter, **fixed** (hilbish has no `COMP_WORDBREAKS` equivalent; its vendored readline hands the full line to the `TabCompleter` without shell-side word splitting). Unlike bash, `@ = :` are **not** wordbreaks.

## Quotes

Same as bash (LangBash):

```sh
echo 'hello world'   # one word, literal
echo '$HOME'         # $HOME (literal)
echo "`cmd`"         # `cmd` (literal in single quotes)
echo "hello world"   # one word
```

- **Single quotes**: literal, no escapes, `''` is close+reopen (two words' segments join), not an escape.
- **Double quotes**: `\` escapes only `$`, `` ` ``, `"`, `\`, and newline (the format's `EscapingQuoteEscapeChars`).

## Escape Character

Backslash `\`:
- **Outside quotes**: escapes the next character; `\` + newline (or `\r`) is a line continuation (`lineContinuationEscaper`).
- **Inside double quotes**: only the escape chars listed above; `\n` (letter n) stays literal.
- **Inside single quotes**: literal.

## Operators

The **full bash grammar** — mvdan/sh's `LangBash` is the zero value of `syntax.NewParser()` and snail passes no variant option, so `|&`, `<<<`, `&>`, `&>>`, `;;&`, `;&` are all valid. Operator classification delegates to `bashWordbreakType`.

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `\|&` | pipe with stderr | pipeline delimiter |
| `<` `>` `>>` `>\|` `<<` `<<<` `<>` `<&` `>&` `&>` `&>>` | redirects | redirect |
| `;` `;;` `;&` `;;&` | command separators / case lists | pipeline delimiter |
| `&` | background | list operator |
| `&&` `\|\|` | logical and/or | list operator |
| `(` `)` | grouping / substitution delimiters | wordbreak (`PostProcess` reclassifies as substitution open/close, merges `$` + `(`) |

## Variables

`$name` and `${name` — POSIX rules via `posixVariableRef`: `$` is literal inside single quotes and after `\`; closed expansions like `${HOME}` are not references.

## Runner Modes and Lua

The active runner decides how input executes; the lexer always applies sh rules:

| Runner | Behavior |
|--------|----------|
| `hybrid` (default) | **Lua first**, fall back to sh on Lua error |
| `hybridRev` | sh first, fall back to Lua |
| `lua` / `sh` | single language only |

Lexer implications:
- Lua code (`print("hi")`) lexes acceptably as sh-like words — Lua strings share `'` and `"` with sh, so word boundaries and quote state are usually right.
- **Lua long strings** (`[[...]]`, `[=[...]=]`) are not lexed — multi-rune openers are a deferred feature (same class as other formats' deferred string types).
- `--`-style Lua comments are not lexed as comments (they are words).

## Completion Insertion

Hilbish's completion layer inserts results as **backslash-escaped barewords**, not quoted strings (`complete.go` `escapeFilename` via `charEscapeMap`):

`"` `'` `` ` `` space `(` `)` `[` `]` `$` `&` `*` `>` `<` `|` are escaped with `\`.

`hilbishQuoteWord` mirrors this and additionally escapes `\` and `;` (omitted from hilbish's map) plus tab and carriage return, so `Join` round-trips: a raw `\` is dropped by the sh parser, a raw `;` splits the command, and a tab would split the bareword (unlike bash, there is no double-quote wrapping to hide it).

## Modifiers

Lines may start with `@name` / `@name=value` modifiers (`@priv`, `@dir=~/x`, `@runner=lua`) that change how the command runs. They lex as plain words (`@` is not a wordbreak) — resolving them is a consumer concern.

## Edge Cases

- **`@` and `=` are not wordbreaks**: `@dir=~/x ls` lexes as two words; `--opt=val` stays one word (no bash `COMP_WORDBREAKS` prefix splitting).
- **Mid-word `#`**: `a#b` is one word; `#` only starts a comment at a word boundary.
- **`<<-`**: tokenizes as `<<` + `-` (same as the bash format; the delimiter `-` only matters mid-heredoc, which is never completed).
- **Multibyte input**: `Span` offsets are rune offsets — `café` spans 4 runes.
- **Trailing `\` at EOF**: reports `ESCAPING_STATE` (hilbish itself would prompt for continuation via `snail.Validate`).

## References

- [format-bash.md](format-bash.md) — POSIX/bash baseline (shared operator grammar)
- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **hilbish** skill — hilbish completion, editor, runner modes, Lua API, startup
- **carapace-dev** skill → `references/shell.md` — carapace's output formatting
