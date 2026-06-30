# Zsh Lexical Format

Lexical rules a command-line lexer needs for zsh. Zsh is POSIX-family and largely shares the bash format, with `RC_QUOTES` (`''`→`'` inside single quotes), `WORDCHARS`/`FIGNORE` for word breaks, and `$'...'` ANSI-C quoting.

> **Source of truth**: zsh manual ([Quoting](https://zsh.sourceforge.io/Doc/Release/Shell-Grammar.html#Quoting), [Expansion](https://zsh.sourceforge.io/Doc/Release/Expansion.html)). For broader zsh internals, use the **zsh** skill. For cross-shell comparison, see [comparison.md](comparison.md). For bash-shared rules, see [format-bash.md](format-bash.md).

## Classification

Zsh uses the same rune classes as bash, with two configuration differences:

| Rune class | Runes | Notes |
|------------|-------|-------|
| space | ` \t\r\n` | same as bash |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` — but `RC_QUOTES` changes behavior |
| escape | `\` | `ESCAPING_STATE` |
| comment | `#` | `COMMENT_STATE` |
| wordbreak | `COMP_WORDBREAKS`-style, modified by `WORDCHARS`/`FIGNORE` | see below |

## Quotes

### Single quotes `'...'`

Like bash, single quotes are literal. **Difference**: zsh has `RC_QUOTES` (off by default) — when set, `''` inside single quotes produces a literal `'` instead of closing-then-opening. This is the same as elvish's default behavior.

```zsh
echo 'hello world'         # hello world
echo '$HOME \n \t'         # $HOME \n \t  (literal)
# with RC_QUOTES set:
echo 'it''s a test'        # it's a test
```

For the lexer: with `RC_QUOTES`, the `QUOTING_STATE` must treat `''` as an escaped quote (stay in state) rather than close-then-open. This is a format-specific extension to the single-quote state machine.

### Double quotes `"..."`

Same as bash — escaping quote, backslash special before `$`, `` ` ``, `"`, `\`, `newline`.

```zsh
echo "$HOME"               # /home/user
echo "say \"hello\""       # say "hello"
```

### ANSI-C quoting `$'...'`

Zsh supports `$'...'` like bash. The `$'` opener enters an ANSI-C escaping state closed by `'`. As with bash, v1 treats `$` as a word character and `'` as a non-escaping quote, so `$'...'` lexes acceptably for completion.

## Word Breaks: WORDCHARS and FIGNORE

Zsh's word-breaking is configurable via two mechanisms that differ from bash's single `COMP_WORDBREAKS`:

### `WORDCHARS`

A set of characters that are **part of a word** (not word breaks). This is the inverse of `COMP_WORDBREAKS` — characters in `WORDCHARS` are kept in the word being completed. Default is empty-ish; users often set `WORDCHARS='*?_-.[]~=/&;!#$%^<>{}'` to make path completion smoother.

For the lexer: zsh completion passes words with `WORDCHARS` characters intact. The wordbreak set should exclude `WORDCHARS` runes.

### `FIGNORE`

A set of filename suffixes (patterns) to ignore during completion. Not a lexer concern — it filters candidates, not word boundaries.

### Carapace integration

Carapace's zsh action uses `shlex.Split(env.Compline())` and reads `RawValue` of the current token to detect quotation state:

```go
// carapace internal/shell/zsh/action.go
rawValue := splitted.CurrentToken().RawValue
switch {
case regexp.MustCompile(`^'$|^'.*[^']$`).MatchString(rawValue):
    state = QUOTING_STATE
case regexp.MustCompile(`^"$|^".*[^"]$`).MatchString(rawValue):
    state = QUOTING_ESCAPING_STATE
case regexp.MustCompile(`^".*"$`).MatchString(rawValue):
    state = FULL_QUOTING_ESCAPING_STATE
case regexp.MustCompile(`^'.*'$`).MatchString(rawValue):
    state = FULL_QUOTING_STATE
}
```

Zsh has a 5-state model (vs bash's 2) because of the "full quoting" states where the space suffix ends up *inside* the closing quote — a zsh-specific quirk that forces nospace.

## Operators

Same operator grammar as bash (`|`, `||`, `|&`, `&`, `;`, `&&`, `<`, `>`, `>>`, `<<<`, `<>`, `<&`, `&>`, `&>>`). See [format-bash.md](format-bash.md#operators-wordbreaktype) for the full table.

Zsh additions (not in bash):
- **`=(...)` process substitution** — `=` as a wordbreak followed by `(...)`.
- **Glob qualifiers** `(...)` after a path — these are word breaks but not operators in the pipeline sense.

## Comments

`#` to end of line, same as bash. `#` must be at a word boundary.

## Edge Cases

- **`RC_QUOTES` off (default)**: `''` closes then reopens single quotes (same as bash).
- **Named directories**: zsh's `hash -d` creates `~name` expansions. Carapace handles this via `NamedDirectories.Matches` in `quoteValue`, not in the lexer.
- **`FULL_QUOTING_*_STATE` quirk**: when a word both starts and ends with the same quote, zsh places the trailing space *inside* the quote. Carapace forces nospace in these states.

## References

- [format-bash.md](format-bash.md) — shared POSIX rules (operators, escapes, comments)
- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **zsh** skill — compsys, ZLE, expansion/quoting, startup
- **carapace-dev** skill → `references/shell-zsh.md` — carapace's zsh output formatting
