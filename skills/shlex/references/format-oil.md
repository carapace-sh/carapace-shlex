# Oil Shell Lexical Format

Lexical rules a command-line lexer needs for Oil (OSH and YSH). OSH is bash-compatible (POSIX format). YSH adds string types (`r'...'`, `u'...'`, `b'...'`, triple-quoted) and "simple word evaluation" which changes variable/expansion semantics but not the core lexing.

> **Source of truth**: Oil docs ([Word Language](https://www.oilshell.org/release/latest/doc/word-language.html), [YSH Strings](https://www.oilshell.org/release/latest/doc/ysh.html#string)). For broader Oil internals, use the **oil** skill. For cross-shell comparison, see [comparison.md](comparison.md). For bash-shared rules, see [format-bash.md](format-bash.md).

## Classification

### OSH (bash-compatible)

Same as bash — see [format-bash.md](format-bash.md). OSH aims for bash lexical compatibility.

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` |
| escape | `\` | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |

### YSH (additional string types)

YSH adds string type prefixes that change escape behavior:

| Type | Syntax | Escapes | Lexer state |
|------|--------|---------|-------------|
| Single-quoted | `'...'` | none | `QUOTING_STATE` |
| Double-quoted | `"..."` | `$`, `` ` ``, `\` | `QUOTING_ESCAPING_STATE` |
| ANSI-C | `$'...'` | C-style | escaping-quote-like |
| Raw string | `r'...'` | none (literal backslash) | `QUOTING_STATE` (raw) |
| Unicode | `u'...'` | `\` escapes | `QUOTING_ESCAPING_STATE` |
| Byte | `b'...'` | `\` escapes | `QUOTING_ESCAPING_STATE` |
| Triple-quoted | `'''...'''` `"""..."""` | depends on base type | multi-line quoting |

```bash
echo r'C:\Program Files\'    # Raw: backslashes are literal
echo '''multi
line string'''               # triple-quoted, strips leading whitespace
```

For the lexer: the `r`, `u`, `b` prefixes are word characters adjoining the quote (`Words()` merges). `r'...'` should enter a non-escaping state even with double-quote-like rules. Triple-quotes need 3-rune lookahead.

## Quotes

### Single quotes `'...'`

Same as bash — literal, no escapes. To include a single quote: `'It'\''s'`.

### Double quotes `"..."`

Same as bash — `$`, `` ` ``, `\` are special.

### ANSI-C `$'...'`

Same as bash.

### YSH raw strings `r'...'`

Backslashes are literal. No escape processing. The `r` prefix makes the following single-quoted string raw.

### YSH triple-quoted `'''...'''` / `"""..."""`

Multi-line strings. Leading whitespace is stripped (dedented). For the lexer, the opener is 3 runes (`'''` or `"""`); the closer must match. This needs a 3-rune lookahead to distinguish from single quotes.

## Escape Character

Backslash `\`:
- **Outside quotes (OSH)**: escapes next rune (bash-compatible).
- **Outside quotes (YSH)**: in simple word evaluation, fewer contexts use backslash.
- **Inside `"..."`**: special before `$`, `` ` ``, `"`, `\`, newline (bash rule).
- **Inside `r'...'`**: literal (raw).
- **Inside `'...'`**: none.

## Simple Word Evaluation (YSH)

YSH introduces "simple word evaluation" — variables are not word-split, and the expansion model is simpler. This affects *semantics* but not *lexing*: the lexer still splits on literal source whitespace and tracks quotes the same way. The difference is that `$var` doesn't produce multiple words even unquoted.

## Word Breaks and Operators

Same operator grammar as bash (`|`, `||`, `|&`, `&`, `;`, `&&`, `<`, `>`, `>>`, `<<<`, `<>`, `<&`, `&>`, `&>>`). OSH is bash-compatible.

YSH differences:
- **`|&`** — in YSH, `|&` is a pipeline operator (stderr merged).
- **No `==`** wordbreak concern (that's comparison, not a word break).

## Comments

`#` to end of line, same as bash.

## Headless Mode and Parser-as-Library

Oil supports `--headless` mode and can be used as a parsing library. This means the lexer's output can be validated against Oil's own parser — useful for testing a v2 oil format. The `OILS_COMP_UI` variable controls completion display.

## Edge Cases

- **YSH `r'...'`**: the `r` prefix changes escape behavior — the classifier/state machine must know the prefix.
- **Triple-quotes**: 3-rune lookahead needed.
- **OSH bash-compat**: OSH format can reuse the bash format almost verbatim.
- **`$'...'` ANSI-C**: same bash treatment (`$` as word char + `'` as quote).
- **Simple word eval**: doesn't change lexing, only post-lexing expansion.

## References

- [format-bash.md](format-bash.md) — POSIX baseline (OSH is bash-compatible)
- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **oil** skill — Oil completion, line editing, quoting/expansion, execution, startup
- **carapace-dev** skill → `references/shell-oil.md` — carapace's oil output formatting
