# Cmd.exe / Clink Lexical Format

Lexical rules a command-line lexer needs for Windows cmd.exe (with clink). Cmd uses caret (`^`) as the escape character, double quotes (`"..."`) with no single-quote support, `%VAR%` variable expansion, `&`/`&&`/`||` command separators, `|` pipe, `>`/`<`/`>>` redirects, and `REM`/`::` comments.

> **Source of truth**: cmd.exe documentation, [clink docs](https://chrisant996.github.io/clink/). For broader clink internals, use the **cmd-clink** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t` | word delimiter (within a command) |
| newline | `\r\n` | command terminator |
| quote | `"` | `QUOTING_ESCAPING_STATE`-like (but no `\` escapes) |
| escape | `^` | `ESCAPING_STATE` |
| comment | `REM` / `::` | `COMMENT_STATE` |

**Key differences from POSIX**:
- **No single quotes** — `'` is a literal character, not a quote.
- **Caret `^` is the escape character**, not backslash.
- **No `\` escaping** — backslash is literal.
- **`%VAR%` variable expansion** — `%` is a sigil, but not a word break.

## Quotes

### Double quotes `"..."`

Cmd uses double quotes for strings containing spaces. **No escape processing inside** — there is no `\"` to include a literal quote inside a quoted string. The `"..."` pair is a simple toggle.

```cmd
echo "hello world"          # hello world
echo "say "hello""          # say hello (quotes toggle, no escaping)
```

For the lexer: cmd double quotes are a simple toggle state — enter on `"`, exit on `"`. No `ESCAPING_QUOTED_STATE` because `\` is not special. The caret `^` **does** escape inside quotes in cmd (`` ^" `` produces a literal `"`), so the state machine needs `^` to work as an escape in the quoting state.

### No single quotes

```cmd
echo 'hello world'          # 'hello world' (single quotes are literal characters)
```

For the lexer: `'` is a regular word character in the cmd format, not a quote.

## Escape Character

Caret `^`:
- **Outside quotes**: escapes the next character (literal). Commonly used to escape operators (`^&`, `^|`, `^>`, `^<`).

```cmd
echo hello^&world           # hello&world (literal &)
echo ^|                     # | (literal pipe)
```

- **Inside double quotes**: `^` still escapes the next character (`` ^" `` → literal `"`).

```cmd
echo "say ^"hello^""        # say "hello"
```

For the lexer: `^` enters `ESCAPING_STATE` (and `ESCAPING_QUOTED_STATE` inside quotes) — the next rune is literal.

## Variable Expansion `%VAR%`

Cmd uses `%NAME%` for variable expansion (and `!NAME!` with enabledelayedexpansion):

```cmd
echo %PATH%
echo %USERPROFILE%\Desktop
```

For the lexer: `%` is a word character (expansion happens post-lexing). `%` is **not** a word break.

## Word Breaks and Operators

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `>` `>>` `<` | redirects | redirect |
| `2>` `2>&1` etc. | stream redirects | redirect |
| `&` | command separator (sequential) | pipeline delimiter |
| `&&` | conditional and (run if success) | list operator |
| `\|\|` | conditional or (run if failure) | list operator |

**Key difference from POSIX**: `&` in cmd is a *command separator* (like `;` in bash), not a background operator. `command1 & command2` runs both sequentially. `&&` and `||` are conditional separators.

The wordbreak rune set for cmd: `|`, `&`, `<`, `>`, and `\r\n` (command terminators). `;` is **not** a separator in cmd (it's a literal character or argument separator within some commands).

## Comments

Cmd has two comment styles:

### `REM` command

`REM` is a command that does nothing — the rest of the line is ignored:

```cmd
REM This is a comment
```

For the lexer: `REM` is a word that, when at command position (first word), starts a comment. This is keyword-based comment detection, unlike `#` rune-based.

### `::` label comment

`::` is a label-style comment (technically a label, but used as a comment):

```cmd
:: This is a comment
```

For the lexer: `::` at a word boundary starts a comment to end of line. This is a two-rune comment opener.

## Carapace Integration

Carapace's cmd-clink integration uses `CARAPACE_COMPLINE` env var and `cmd_clink.Patch()` re-lexes the command line with carapace-shlex, stripping redirects:

```go
// carapace internal/shell/cmd_clink/patch.go (conceptual)
// Re-lexes CARAPACE_COMPLINE; strips redirects via FilterRedirects()
```

The clink snippet is a Lua function that invokes carapace with the completion line. Clink's match generation pipeline handles the actual completion matching.

## Edge Cases

- **No single quotes**: `'` is literal. Don't classify it as a quote.
- **Caret `^` as escape**: both inside and outside quotes.
- **`&` as separator**: `&` splits commands (like `;` in POSIX), not background. `&&`/`||` are conditional.
- **`REM` keyword comment**: not a rune-based comment — needs word-level detection.
- **`::` comment**: two-rune opener at word boundary.
- **`%` not a wordbreak**: variable sigil, part of words.
- **`\r\n` command terminator**: within a single command, only space/tab delimit words; newlines terminate the command.
- **No `\` escaping**: backslash is literal (Windows paths use `\` freely).

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **cmd-clink** skill — clink argmatcher, completion, line editing, cmd integration
- **carapace-dev** skill → `references/shell.md` — carapace's cmd-clink output formatting
