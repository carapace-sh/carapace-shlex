# Xonsh Lexical Format

Lexical rules a command-line lexer needs for xonsh. Xonsh is a Python/shell hybrid — every line is Python, extended with subprocess syntax. The lexer must handle Python string literals (single, double, raw `r'...'`, f-strings `f'...'`, p-strings `p'...'`, byte `b'...'`) alongside shell operators (`|`, `>`, `<`, `;`, `&&`, `||`).

> **Source of truth**: xonsh docs ([Strings](https://xon.sh/tutorial.html#strings), [Subprocess](https://xon.sh/tutorial.html#subprocess-commands)). For broader xonsh internals, use the **xonsh** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` (Python escapes) |
| non-escaping quote | `'` | `QUOTING_STATE` (Python single) |
| escape | `\` | `ESCAPING_STATE` (Python rules) |
| comment | `#` | `COMMENT_STATE` (Python `#`) |

## The Hybrid Model

Xonsh determines whether a line is Python or shell (subprocess) mode based on parsing:
- Lines starting with a known command → subprocess mode
- Lines with Python syntax → Python mode
- Ambiguous → Python takes precedence

For a command-line **lexer focused on completion**, the input is typically in subprocess mode (the user is typing a command). The lexer needs to handle:
- Bare command tokens (subprocess args)
- Python string literals as arguments
- Shell operators (`|`, `>`, `<`, `;`)

## Quotes

Xonsh supports Python's full string literal syntax as argument values:

| Type | Syntax | Escapes | Notes |
|------|--------|---------|-------|
| Single-quoted | `'text'` | `\` escapes (Python) | literal-ish |
| Double-quoted | `"text"` | `\` escapes (Python) | supports interpolation in f-strings |
| Raw | `r'text'` / `r"text"` | none | backslashes literal |
| F-string | `f'text'` / `f"text"` | `\` escapes + `{expr}` | interpolation |
| P-string | `p'text'` / `p"text"` | `\` escapes | pathlib `Path` |
| Byte | `b'text'` / `b"text"` | `\` escapes | bytes |
| Triple-quoted | `'''text'''` / `"""text"""` | `\` escapes | multi-line |

```python
echo 'hello world'           # hello world
echo "say \"hello\""         # say "hello"
echo r'C:\path\to\file'      # C:\path\to\file (raw — backslashes literal)
echo f"value: {x}"           # value: <x> (interpolated)
```

For the lexer: the prefix characters (`r`, `f`, `p`, `b`) are word characters that adjoin the following quoted segment — `Words()` merges them. The quote behavior inside is standard Python: `\` is an escape in non-raw strings, literal in raw strings. Triple-quotes (`'''`/`"""`) are detected via 2-rune lookahead and enter dedicated triple-quote states. Raw string prefixes (`r`/`R`) suppress escape processing in double quotes.

### Python escape sequences (in non-raw strings)

| Sequence | Meaning |
|----------|---------|
| `\\` | backslash |
| `\'` `\"` | quote chars |
| `\n` `\r` `\t` | control chars |
| `\xHH` | hex byte |
| `\uHHHH` `\UHHHHHHHH` | Unicode |
| `\ooo` | octal |

## Escape Character

Backslash `\` follows Python rules:
- **Outside quotes**: no special meaning (bareword context).
- **Inside non-raw strings**: Python escape sequences.
- **Inside raw strings** (`r'...'`, `r"..."`, `r"""..."""`): literal (no escapes). The lexer detects the `r`/`R` prefix and suppresses escape processing.

## Word Breaks and Operators

Xonsh uses shell-style operators for subprocess pipelines:

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `>` `>>` `<` | redirects | redirect |
| `2>` `2>&1` `e>` `e<` `e>>` `o>` `o>>` etc. | stream redirects | redirect |
| `;` | command separator | pipeline delimiter |
| `&` | background | list operator |
| `&&` `\|\|` | logical and/or | list operator |

Special subprocess syntax (not word breaks but syntax):
- `$(cmd)` — stdout capture
- `!(cmd)` — object capture
- `$[cmd]` — uncaptured
- `![cmd]` — hidden capture
- `@()` — Python expression evaluation in subprocess args

These `()`, `[]`, `{}` are syntax that the lexer treats as word characters (they're part of the subprocess expression), similar to how POSIX treats `$(...)`.

## Comments

`#` to end of line (Python comment style). `#` at a word boundary.

```python
echo hello # a comment
```

## Word Splitting and Globbing

Xonsh performs word splitting and glob expansion on subprocess arguments, controlled by environment variables:
- `SUBPROC_TTY` — whether to capture
- Glob: `*`, `?`, `**` (Python-style globbing)
- Tilde: `~` expansion

For the lexer, glob and tilde characters are regular word characters — expansion happens after lexing.

## Carapace Integration

Carapace's xonsh integration receives `CommandContext.args` (already split by xonsh) and `sub_proc_get_output` for the callback. The lexer is used to re-split the command line for completion context when needed.

## Edge Cases

- **String prefixes** (`r`, `f`, `p`, `b`): word characters that adjoin the quote. `Words()` handles merging.
- **Triple-quotes**: `'''` and `"""` are detected via 2-rune lookahead and enter dedicated triple-quote states (`QUOTING_TRIPLE_STATE` for non-escaping, `QUOTING_TRIPLE_ESCAPING_STATE` for escaping). Closing requires 3 consecutive matching quote runes.
- **Raw strings**: `r'...'` and `r"..."` suppress escape processing. The lexer checks if the word ends with `r`/`R` before entering the quoting state. When raw prefix is detected, backslash is treated as a literal character inside double quotes (same as single quotes). Also applies to triple-quoted raw strings (`r"""..."""`).
- **Keyword operators**: `and` and `or` bare words are treated as `&&` and `||` respectively when they appear as standalone words (surrounded by whitespace). Implemented via `KeywordOperators()`.
- **Stream redirect operators**: `e>`, `e>>`, `o>`, `o>>`, `a>`, `a>>`, `err>`, `out>`, `all>`, and pipe-channel variants (`e>p`, `o>p`, `a>p`) are merged via `PostProcess`. The word portion (`e`, `o`, `a`, `err`, `out`, `all`) is a WORD_TOKEN that adjoins the `>`/`>>` WORDBREAK_TOKEN; the PostProcess step merges them into a single WORDBREAK_TOKEN.
- **Subprocess capture syntax**: `$(...)`, `!(...)`, `$[...]`, `![...]` — `(`, `)`, `[`, `]` are word characters in this context.
- **`@()` expression**: Python expression interpolation in subprocess args.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **xonsh** skill — xonsh completion, prompt-toolkit, language/execution, startup
- **carapace-dev** skill → `references/shell-xonsh.md` — carapace's xonsh output formatting
