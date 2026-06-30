# Elvish Lexical Format

Lexical rules a command-line lexer needs for elvish. Elvish uses `'`/`"` quotes with `''` (doubled single quote) as the escape inside single quotes, a bareword charset, no word splitting, and no POSIX list operators.

> **Source of truth**: elvish docs ([String](https://elv.sh/ref/language.html#string), [Bareword](https://elv.sh/ref/language.html#bareword)). For broader elvish internals, use the **elvish** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` — **`''` → `'`** |
| escape | `\` (only inside `"..."`) | — |
| comment | `#` | `COMMENT_STATE` |

## Quotes

### Single quotes `'...'`

All characters literal **except** `''` (two single quotes) which produces one literal `'`. This is elvish's default behavior — unlike bash where `''` closes-then-opens, elvish treats `''` as an escaped quote and stays inside the string.

```elvish
echo 'hello world'        # hello world
echo '$HOME \n \t'        # $HOME \n \t  (all literal)
echo 'it''s a test'       # it's a test ('' → ')
echo 'a'\''b'             # INVALID in elvish (no backslash escaping outside quotes)
```

For the lexer: elvish single quotes need a `QUOTING_STATE` where `''` is recognized as an escaped quote (consume both, emit one `'`, stay in state). This is the same as zsh's `RC_QUOTES`. The state machine needs a peek-ahead when it sees `'` in `QUOTING_STATE`: if the next rune is also `'`, consume both and stay; otherwise, close.

### Double quotes `"..."`

Escaping quote with backslash escape sequences:

| Sequence | Meaning |
|----------|---------|
| `\n` | newline |
| `\t` | tab |
| `\r` | carriage return |
| `\\` | backslash |
| `\"` | double quote |
| `\xHH` | hex byte |
| `\uHHHH` | 16-bit Unicode |
| `\UHHHHHHHH` | 32-bit Unicode |
| `\cX` | control character |

```elvish
echo "hello\nworld"       # two lines (\n IS interpreted in elvish double quotes)
echo "say \"hello\""      # say "hello"
```

Note: elvish double quotes interpret `\n` etc. — unlike bash/fish where `\n` is literal in double quotes. For the lexer (which doesn't evaluate escapes, just tracks state), this difference doesn't matter: the tokenizer just needs to know `\` is an escape inside `"..."`.

### No string interpolation

Elvish has **no** string interpolation. Use concatenation (juxtaposition):

```elvish
var name = world
echo "hello "$name        # concatenation, not interpolation
```

For the lexer: `$` is a regular word character inside `"..."` (no special handling needed beyond not being a quote terminator).

## Barewords

Unquoted strings (barewords) consisting of these characters need no quoting:

```
ASCII letters, digits, ! % + , - . / : @ _ \  (backslash is a bareword char!)
non-ASCII printable characters
```

**Important**: backslash `\` is a valid bareword character in elvish (unlike POSIX where `\` is always an escape). This means outside quotes, `\` is **not** an escape character in elvish — it's a literal bareword character.

For the lexer: the elvish format must **not** classify `\` as `escapeRuneClass` outside quotes. `\` only has escape meaning inside `"..."`. This is a significant departure from the v1 model where `\` is always an escape.

## Escape Character

- **Outside quotes**: `\` is a literal bareword character (not an escape).
- **Inside double quotes**: `\` is an escape (the sequences above).
- **Inside single quotes**: no escapes (`''` is the only special sequence, handled as a quote-doubling rule, not a backslash escape).

## Word Breaks and Operators

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `>` `<` `>>` `>>?` `<>>` | redirects | redirect |
| `;` | command separator | pipeline delimiter |

Elvish has **no** POSIX list operators (`&&`, `||`, `&`, etc.). Logical operations use the `and`/`or` commands (not keyword operators). The `&` character is used for map literals (`&key=value`) and option syntax, not as a background/list operator.

The wordbreak rune set is minimal: `|`, `<`, `>`, `;` and their multi-char combinations. `(`, `)`, `[`, `]`, `{`, `}` are syntax for lambdas, lists, and maps — not word breaks in the pipeline sense.

## Comments

`#` to end of line. `#` must be at a word boundary.

```elvish
echo hello # a comment
```

## No Word Splitting

Elvish does not word-split on variable expansion. `$var` is a single value regardless of contents. The lexer splits on literal source whitespace only.

## Edge Cases

- **`''` in single quotes**: the primary lexer deviation. Must peek-ahead on `'` in `QUOTING_STATE`.
- **`\` as bareword**: outside quotes, `\` is literal. The classifier must not mark it as `escapeRuneClass` for the elvish format.
- **Map/option syntax `&`**: `&` is not a list operator in elvish. Including it in the wordbreak set would break `&key=value` map literals.
- **No `COMP_WORDBREAKS`**: elvish has no equivalent env var.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model
- [format-zsh.md](format-zsh.md) — zsh `RC_QUOTES` (same `''` behavior)

## Related Skills

- **elvish** skill — elvish completion, editor, styling, language, startup
- **carapace-dev** skill → `references/shell-elvish.md` — carapace's elvish output formatting
