# Fish Lexical Format

Lexical rules a command-line lexer needs for fish. Fish uses `'`/`"` quotes and `\` escape like POSIX shells, but differs in: `\'` and `\\` are escapes *inside single quotes*, no word splitting on variable expansion, keyword list operators (`and`/`or`/`not`), and a simpler expansion model.

> **Source of truth**: fish documentation ([Quotes](https://fishshell.com/docs/current/language.html#quotes), [Escaping](https://fishshell.com/docs/current/language.html#escaping)). For broader fish internals, use the **fish** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` — **limited escapes: `\"`, `\$`, `\\`, `\`+newline only** |
| non-escaping quote | `'` | `QUOTING_STATE` — **but `\'` and `\\` are escapes** |
| escape | `\` | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |
| wordbreak | `\| ; < > & ?` | `WORDBREAK_STATE` |

## Quotes

### Single quotes `'...'`

No expansions are performed. **Difference from bash**: `\'` and `\\` are meaningful escapes inside fish single quotes. All other backslash sequences are literal.

```fish
echo 'The value is $HOME'     # The value is $HOME (literal)
echo 'it'\''s a test'         # it's a test (close, escaped quote, reopen)
echo 'cost: \$5'              # cost: \$5  (\$ is NOT an escape in single quotes — literal)
echo 'say \'hello\''          # say 'hello'  (\' IS an escape in fish single quotes)
```

For the lexer: fish single quotes need a format-specific `QUOTING_STATE` that recognizes `\'` and `\\` as escapes. This is the key deviation from the v1 two-state model (where single quotes have zero escapes). The state machine needs an `ESCAPING_QUOTED_STATE`-like transition inside the non-escaping quote state when `\` precedes `'` or `\`.

### Double quotes `"..."`

Escaping quote. Only variable expansion (`$VAR`) and command substitution (`$(cmd)`) occur. Escape sequences like `\n` are **not** interpreted in double quotes. Meaningful escapes: `\"`, `\$`, `\\`, and `\` + newline (line continuation).

```fish
echo "The value is $HOME"     # The value is /home/user (expanded)
echo "say \"hello\""          # say "hello"
echo "cost: \$5"              # cost: $5
echo "hello\nworld"           # hello\nworld (literal \n, not newline)
```

For the lexer: inside `"..."`, backslash is special only before `"`, `$`, `\`, and newline — narrower than bash. Other `\X` sequences are kept literal.

### No word splitting

Fish does **not** word-split on unquoted `$var` expansion. `$var` and `"$var"` behave the same — both produce one argument even if the value contains spaces. This doesn't change the lexer (which splits on literal source whitespace), but it means completion values with spaces don't need defensive quoting the way bash does.

## Escape Character

Backslash `\`:

- **Outside quotes**: escapes the next character (literal). Full set of special escapes: `\$ \\ \* \? \~ \# \( \) \{ \} \[ \] \< \> \& \| \; \" \' \ ` plus unicode escapes (`\xHH`, `\uXXXX`, `\UXXXXXXXX`, `\cX`).
- **Inside double quotes**: only `\" \$ \\` and `\`+newline.
- **Inside single quotes**: only `\'` and `\\`.

```fish
echo \$HOME                   # $HOME
echo a\ b                    # a b (escaped space → one word)
```

## Word Breaks and Operators

Fish's operators differ from POSIX:

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `\|&` `&\|` | pipe with stderr merge | pipeline delimiter |
| `>\|` | pipe with explicit fd | pipeline delimiter |
| `<` `>` `>>` `>>?` `>?` `<?` `<>` `<>&` `>&` | redirects | redirect (for `FilterRedirects`) — `>&` is fd duplication |
| `&>` `&>>` `&>>?` `&>?` | redirects with stderr merge | redirect |
| `;` | command separator | pipeline delimiter |
| `&&` | logical and | list operator (pipeline delimiter) |
| `\|\|` | logical or | list operator (pipeline delimiter) |
| `&` | background | list operator (pipeline delimiter) |
| `and` | logical and (keyword) | list operator |
| `or` | logical or (keyword) | list operator |
| `not` | logical not (keyword) | (prefix, not a delimiter) |

**Keyword operators**: fish supports both rune-based operators (`&&`, `||`, `&`) and keyword-based operators (`and`, `or`, `not`). The keyword operators are bare words that act as pipeline delimiters. The tokenizer recognizes these via `KeywordOperators()` at word boundaries. `not` is not included since it's a prefix keyword, not a pipeline delimiter.

The wordbreak rune set for fish is `|;<>&?` — includes `&` for `&&`/`||`/`&`/`|&`/`&|`/`&>`/`>&`/`<>&` and `?` for `>?`/`>>?`/`<?`. Parentheses are not wordbreaks (command substitution syntax).

## Comments

`#` to end of line, same as POSIX. `#` must be at a word boundary.

```fish
echo hello # a comment
echo hello#world       # hello#world (no comment — # is mid-word)
```

## Expansion Differences (lexer-irrelevant but context)

Fish performs expansions in order: command substitution → variable expansion → bracket expansion → wildcard expansion. There is no brace expansion, tilde expansion (as a separate phase), parameter expansion, arithmetic expansion, or process substitution. For the lexer, this means fewer special characters to track — `$` and `()`/`$()` are the main interpolating constructs, and they pass through as word characters.

## Edge Cases

- **`\'` and `\\` in single quotes**: the main lexer deviation from bash. Only `\'` and `\\` are escapes; other `\X` sequences (e.g. `\$`) are literal `\X` (both characters emitted). The `NonEscapingQuoteBackslashEscapes` flag enables this behavior in the state machine.
- **Limited double-quote escapes**: fish only recognizes `\"`, `\$`, `\\`, and `\`+newline as escapes inside double quotes. All other `\X` sequences are literal (both `\` and `X` emitted). The `EscapingQuoteEscapeChars` method returns the allowed set; the state machine checks it in `ESCAPING_QUOTED_STATE`.
- **`(...)` command substitution**: parentheses are not word breaks in fish (they're part of command substitution syntax). The lexer treats `(` and `)` as regular word characters.
- **No `COMP_WORDBREAKS`**: fish has no equivalent of bash's `COMP_WORDBREAKS` env var. The wordbreak set is fixed by the format.
- **QuoteWord**: fish uses double-quote wrapping with `\"`, `\$`, `\\`, and `\`+newline escapes for `JoinWith`. Backtick is not special in fish (not command substitution), so it doesn't trigger quoting.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model
- [format-bash.md](format-bash.md) — POSIX baseline (shared quote/escape rules)

## Related Skills

- **fish** skill — fish completion, editor, language, startup
- **carapace-dev** skill → `references/shell-fish.md` — carapace's fish output formatting
