# Tcsh Lexical Format

Lexical rules a command-line lexer needs for tcsh. Tcsh is POSIX-family with csh heritage: single/double quotes, backslash escape, `$'...'` ANSI-C quoting, backtick command substitution, `!` history expansion (processed before quoting), and the `backslash_quote` option.

> **Source of truth**: tcsh man page ([Quoting](https://man.openbsd.org/tcsh#Quoting), [Filename substitution](https://man.openbsd.org/tcsh#Filename_substitution)). For broader tcsh internals, use the **tcsh** skill. For cross-shell comparison, see [comparison.md](comparison.md). For bash-shared rules, see [format-bash.md](format-bash.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` |
| escape | `\` | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |
| command subst | `` ` `` (backtick) | (word char, but tracked) |

## Quotes

### Single quotes `'...'`

Prevent **all** substitutions except history expansion (`!`). No escape mechanism — a single quote cannot appear inside single quotes.

```tcsh
echo '$HOME'         # $HOME (literal)
echo '`cmd`'         # `cmd` (literal backticks)
echo 'hello world'   # hello world
```

**Important tcsh quirk**: single quotes do **not** prevent history expansion (`!`). The `!` is processed *before* quoting in tcsh. To prevent history expansion, use `\!`. For the lexer, this is a pre-processing concern — the `!` inside single quotes is still a word character (history expansion happens at a different layer).

### Double quotes `"..."`

Allow variable and command substitution, prevent globbing. History expansion still occurs (use `\!`).

```tcsh
echo "$HOME"        # /home/user
echo "`date`"       # current date
echo "hello world"  # hello world (one word)
```

### ANSI-C quoting `$'...'`

tcsh supports `$'...'` for C-style escape sequences:

```tcsh
echo $'hello\nworld'  # two lines
echo $'tab\there'     # tab	here
```

Supported: `\a \b \e \f \n \r \t \v \\ \' \nnn` (octal). Like bash's `$'...'`.

## Escape Character

Backslash `\`:
- **Outside quotes**: escapes the next character (literal). `\` + newline = line continuation (treated as a blank).
- **Inside double quotes**: `\` followed by newline = literal newline; otherwise `\` escapes `\`, `'`, `"` (when `backslash_quote` is set).
- **Inside single quotes**: no escape (literal) — **except** `\!` is processed before quoting to stop history expansion.
- **`\!`**: prevents history expansion regardless of quoting context.

```tcsh
echo \$HOME         # $HOME
echo \*             # *
echo \"hello\"      # "hello"
```

### The `backslash_quote` variable

When set (`set backslash_quote`), backslashes always quote `\`, `'`, and `"` — even inside quotes. This simplifies complex quoting but may break csh compatibility. For the lexer, this changes the escape behavior inside single quotes (making `\` an escape there too).

## Word Breaks and Operators

Tcsh has its own operator grammar, distinct from bash. The wordbreak runes are derived from tcsh's `_META` character class in `sh.char.c`: `><;|&()`. Unlike bash, `=` and `@` are **not** wordbreak characters in tcsh. `!` is also **not** a wordbreak — it is `_PUN` (punctuation) in tcsh's lexer, not `_META`. The `>!` and `>>!` noclobber override operators are recognized at the **parser** level in `syn3()`, not the lexer level: the lexer produces `>` and `!` as separate tokens, and the parser checks if the word after `>` is `!`.

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `\|&` | pipe with stderr | pipeline delimiter |
| `<` `>` `>>` | redirects | redirect |
| `<<` | here document | redirect |
| `>&` | redirect stdout and stderr | redirect |
| `<&` | duplicate input fd | redirect |
| `<>` | input/output redirect | redirect |
| `;` | command separator | pipeline delimiter |
| `&` | background | list operator |
| `&&` `\|\|` | logical and/or | list operator |
| `(` `)` | command grouping | wordbreak (not pipeline delimiter) |

### Tcsh-specific operators (differ from bash)

| Operator | Tcsh | Bash equivalent |
|----------|------|----------------|
| `>&` | Combined stdout+stderr redirect | `&>` |

### Parser-level operators (not lexer tokens)

| Operator | How tcsh handles it |
|----------|---------------------|
| `>!` | Lexer produces `>` and `!` as separate tokens; parser in `syn3()` checks if word after `>` is `!` |
| `>>!` | Same: lexer produces `>>` and `!`; parser recognizes the `!` as noclobber override |

### Operators NOT in tcsh

| Bash operator | Why tcsh doesn't use it |
|---------------|------------------------|
| `>\|` | tcsh uses `>!` instead |
| `>>\|` | tcsh uses `>>!` instead |
| `&>` | tcsh uses `>&` instead |
| `&>>` | tcsh has no `&>>` equivalent |
| `<<<` | tcsh has no here-string |
| `;;` `;;&` `;&` | tcsh uses `breaksw`/`endsw` for case statements |

## Comments

`#` to end of line. `#` must be at a word boundary (start of a word).

```tcsh
echo hello # a comment
echo hello#world     # hello#world (no comment — # is mid-word)
```

## History Expansion (`!`)

tcsh processes `!` for history expansion **before** quoting. This means `!` is special even inside single quotes. For the lexer:
- `!` is a regular word character (history expansion is a separate pre-processing layer).
- The completion input (`COMMAND_LINE`) has already had history expansion applied (or not, depending on context).

## Backtick Command Substitution

tcsh uses backticks for command substitution (no `$()` in classic csh). Inside double quotes, backticks start command substitution:

```tcsh
echo "`date`"       # current date
set d = `date`      # split output on newlines
```

For the lexer: backtick inside double quotes is a word character (the tokenizer doesn't evaluate command substitution). The state machine tracks that we're inside `"..."` and backtick doesn't terminate the quote.

## Carapace Integration

Carapace's tcsh integration uses the `COMMAND_LINE` variable which contains the full command line. The lexer splits it to determine completion context. tcsh passes the command line with quoting intact.

## Edge Cases

- **`!` is a regular word character**: tcsh's lexer classifies `!` as `_PUN` (punctuation), not `_META`. The lexer treats `!` as a word char. History expansion (`!`) is a separate pre-processing layer. The `>!` and `>>!` noclobber operators are parser-level constructs (the parser checks if the word after `>` is `!`).
- **`backslash_quote`**: changes escape behavior inside quotes — a format configuration option (not currently implemented).
- **`<<` here-doc**: tcsh uses `<<` (not `<<<`). `<<<` is not a valid tcsh operator.
- **`>&`**: tcsh combined stdout+stderr redirect (bash uses `&>` instead).
- **`|&`**: tcsh pipes both stdout and stderr to the next command (same as bash).
- **`=` not a wordbreak**: tcsh treats `=` as a regular word character (bash has `=` in COMP_WORDBREAKS for `--foo=bar`). `set foo=bar` lexes as two words.
- **`@` not a wordbreak**: tcsh treats `@` as a regular word character.
- **`$'...'`**: same as bash, lexes acceptably with `$` as word char + `'` as quote.
- **Newline as command separator**: tcsh treats `\n` like `;` (command separator). The lexer treats `\n` as a space (word delimiter) — multiline COMMAND_LINE is rare in completion context.

## References

- [format-bash.md](format-bash.md) — POSIX baseline (shared rules)
- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **tcsh** skill — tcsh completion, editor, quoting/expansion, execution, startup
- **carapace-dev** skill → `references/shell.md` — carapace's tcsh output formatting
