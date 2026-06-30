# Ion Lexical Format

Lexical rules a command-line lexer needs for ion. Ion uses `\` escape, `'`/`"` quotes, `#` comments, and unique pipe/redirect operators: `|>`, `^>`, `^|`, `&|`, `&>`, `&>>`, plus standard `|`, `>`, `>>`, `<`, `<<`, `<<<`, `&&`, `||`, `&`, `;`.

> **Source of truth**: ion docs ([language](https://doc.redox-os.org/ion-manual/), [redirection](https://doc.redox-os.org/ion-manual/redirection.html), [pipelines](https://doc.redox-os.org/ion-manual/pipelines.html)). For broader ion internals, use the **carapace-dev** skill's ion references. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` |
| escape | `\` | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |

## Quotes

### Single quotes `'...'`

Literal — no expansion, no escape processing.

```sh
echo 'Hello $world ${variable}'   # Hello $world ${variable}
```

### Double quotes `"..."`

Allow variable expansion (`$`, `@`) and escape sequences. Brace expansion does **not** work inside double quotes.

```sh
let name = "world"
echo "Hello $name"                # Hello world
echo "@items"                     # one two three (array expanded)
```

Escape sequences (interpreted by `echo -e`, but `\` is the escape char in the lexer for `"..."`):

| Sequence | Meaning |
|----------|---------|
| `\\` | backslash |
| `\n` `\t` `\r` `\a` `\b` `\e` `\f` `\v` | control chars |
| `\c` | no further output |

## Escape Character

Backslash `\`:
- **Outside quotes**: escapes special characters.

```sh
echo hello\ world     # hello world (escaped space → one word)
echo \$HOME           # $HOME (literal)
```

- **Inside double quotes**: `\` is the escape character.
- **Inside single quotes**: none (literal).

## Word Breaks and Operators

Ion's operator set is a superset of POSIX, adding stderr-specific and combined-stream operators:

### Pipes

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | stdout pipe | pipeline delimiter |
| `^|` | stderr pipe (ion unique) | pipeline delimiter |
| `&|` | stdout + stderr pipe (ion unique) | pipeline delimiter |
| `&&` | logical and | list operator |
| `\|\|` | logical or | list operator |
| `&` | background | list operator |
| `;` | command separator | pipeline delimiter |

### Redirects

| Operator | Meaning | Type |
|----------|---------|------|
| `>` | stdout to file | redirect |
| `>>` | append stdout | redirect |
| `<` | stdin from file | redirect |
| `<<` | here-document | redirect |
| `<<<` | here-string | redirect |
| `^>` | stderr to file (ion unique) | redirect |
| `^>>` | append stderr (ion unique) | redirect |
| `&>` | stdout + stderr to file | redirect |
| `&>>` | append stdout + stderr | redirect |

**Key lexer implication**: ion's `^>` and `^|` operators use `^` as a wordbreak rune — but `^` is also the cmd escape character. The ion format must classify `^` as a wordbreak (operator prefix), not an escape. The greedy multi-char operator matching in `WORDBREAK_STATE` handles `^>`, `^>>`, `^|`.

Note: ion does **not** use `=>` — that was a mischaracterization. The redirect operators are `>`, `>>`, `<`, `^>`, `^>>`, `&>`, `&>>`.

## Comments

`#` to end of line, same as POSIX. `#` at a word boundary.

```sh
echo hello # a comment
```

## Variable and Array Expansion

Ion uses `$` for scalar variables and `@` for arrays. Both are expansion sigils:

```sh
echo $name          # scalar
echo @items         # array expanded as separate words
echo "@items"       # one two three (expanded in double quotes)
echo '@items'       # @items (literal in single quotes)
```

For the lexer: `$` and `@` are word characters (expansion happens post-lexing). `@` is **not** a wordbreak in ion (unlike bash where `@` is in `COMP_WORDBREAKS`).

## Edge Cases

- **`^` as operator prefix**: `^>`, `^>>`, `^|` — `^` must be a wordbreak rune in the ion format, classified as an operator prefix, not an escape.
- **`&|` and `&>`**: `&` followed by `|` or `>` is a combined-stream operator, distinct from `&` (background) alone. Greedy multi-char matching handles this.
- **`@` not a wordbreak**: unlike bash, `@` is an array sigil, not a wordbreak.
- **No `COMP_WORDBREAKS`**: ion has no equivalent env var.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model
- [format-bash.md](format-bash.md) — POSIX baseline (shared quote/escape rules)

## Related Skills

- **carapace-dev** skill → `references/shell.md` — carapace's ion output formatting (ion row in the secondary shells table)
