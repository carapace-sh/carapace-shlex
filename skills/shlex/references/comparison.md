# Cross-Shell Lexical Comparison

Side-by-side comparison of the lexical rules a command-line lexer needs for each supported shell: quote characters, escape semantics, word delimiters, operators, and comment syntax. Use this to pick the right format and to understand where shells diverge from POSIX.

> For the common token model and how formats plug into the tokenizer, see [architecture.md](architecture.md). For a single shell's full lexical rules, see its `format-*.md` reference.

## Shell Families

The shells fall into lexical families. A v2 format can often reuse a family's configuration with small overrides.

| Family | Shells | Lexical character |
|--------|--------|-------------------|
| **POSIX** | bash, zsh, oil (OSH), tcsh | backslash escape, `'`/`"` quotes, `#` comment, `|`/`<`/`>`/`&`/`;` operators |
| **Non-POSIX Unix** | fish, elvish, nushell | backslash escape, `'`/`"` quotes, but different operators / no word splitting |
| **Python-hybrid** | xonsh | Python string literals + shell operators |
| **Windows** | PowerShell, cmd (clink) | backtick / caret escape, here-strings, `&` separator |

POSIX-family shells share the v1 operator grammar almost verbatim. The non-POSIX and Windows shells need format-specific operator sets and, in some cases, extra string types.

## Quote Characters

| Shell | Single quote `'` | Double quote `"` | Escaping quote? | Extra string types |
|-------|:-:|:-:|------------------|--------------------|
| **bash** | literal (no escapes) | `$ \`` `` ` `` `\` escapes | `"` | `$'...'` ANSI-C, `$"..."` locale |
| **zsh** | literal (RC_QUOTES: `''`→`'`) | `$ \`` `` ` `` `\` escapes | `"` | `$'...'` ANSI-C |
| **oil (OSH)** | literal | `$ \`` `` ` `` `\` escapes | `"` | `$'...'` ANSI-C |
| **oil (YSH)** | literal | `$ \`` `` ` `` `\` escapes | `"` | `r'...'`, `u'...'`, `b'...'`, `'''...'''` |
| **tcsh** | literal (`!` still expands) | `$`, backtick `\` escapes | `"` | `$'...'` ANSI-C, backtick cmd subst |
| **fish** | `\'` and `\\` only escapes | `\"`, `\$`, `\\`, `\`+newline | `"` | — |
| **elvish** | `''`→`'` (doubled) | `\n \t \xHH \uHHHH \UHHHHHHHH \cX` | `"` | bareword |
| **nushell** | literal (no escapes) | C-style `\" \' \\ \n \t \u{X}` | `"` | `r#'...'#`, `` `...` ``, `$'...'`, `$"..."` |
| **powershell** | literal (`` ` `` is literal) | `` `$ `" `` `` `` `` `n `t `` escapes | `"` | `@'...'@`, `@"..."@` here-strings |
| **xonsh** | Python single-quote | Python double-quote | `"` | `r'...'`, `f'...'`, `p'...'`, `b'...'` |
| **cmd** | — (not a quote) | `"..."` (no escaping inside) | n/a | `%VAR%` |

**Key lexer implication**: "escaping quote" (double-quote-like, `\` is special inside) vs "non-escaping quote" (single-quote-like, literal) maps to `QUOTING_ESCAPING_STATE` vs `QUOTING_STATE` in the state machine. Shells where single quotes have *some* escapes (fish `\'`, elvish `''`) need format-specific handling — the two-state model isn't enough.

## Escape Character

| Shell | Escape char | Outside quotes | Inside escaping quote | Inside literal quote |
|-------|-------------|----------------|------------------------|----------------------|
| **bash** | `\` | escapes next rune | `$ \`` `` ` `` `"` `\` newline only | none (literal) |
| **zsh** | `\` | escapes next rune | `$ \`` `` ` `` `"` `\` newline only | none (RC_QUOTES `''`) |
| **oil** | `\` | escapes next rune | same as bash | none |
| **tcsh** | `\` | escapes next rune (`\!` stops history) | `$`, backtick, `\` newline | none (`!` still expands) |
| **fish** | `\` | escapes next rune | `\"`, `\$`, `\\`, `\`+newline | `\'`, `\\` only |
| **elvish** | `\` | — (no bareword escapes) | `\n \t \xHH`... | `''`→`'` |
| **nushell** | `\` | — | C-style in `"..."` and `$"..."` | none in `'...'` |
| **powershell** | `` ` `` | `` `$ `" `` `` `` `` `n `t `` | same (in `"..."`) | none (literal in `'...'`) |
| **xonsh** | `\` | Python rules | Python rules | none |
| **cmd** | `^` | escapes next char | `^` escapes inside `"..."` | n/a |

## Word Delimiters (Spaces)

All shells treat ` \t\r\n` as word delimiters. The tokenizer's `spaceRuneClass` is the same across formats. Differences:

- **fish** does **not** word-split on unquoted variable expansion (a `$var` with spaces is one word), but the *lexer* still splits on literal whitespace in the source — so lexing is the same as POSIX for literal input.
- **elvish** and **nushell** similarly don't word-split; the lexer splits on literal source whitespace only.
- **cmd** treats `\r\n` and `&` as command separators; within a single command, space/tab delimit words.

## Operators and Word Breaks

These are the characters that break a word and are classified as `WORDBREAK_TOKEN`. The v1 set is `BASH_WORDBREAKS = " \t\r\n" + "'\"@><=;|&(:"`.

| Shell | Pipe | Redirect | Command sep | List operators | Other wordbreaks |
|-------|------|----------|-------------|----------------|------------------|
| **bash** | `\|` | `< > >> <<< <> <& &> &>>` | `;` `&` `&&` `\|\|` `\|&` | `&&` `\|\|` `&` `;` | `@ = ( :` + `COMP_WORDBREAKS` |
| **zsh** | `\|` | `< > >> <<< <> <& &> &>>` `\|&` `=(...)` | `;` `&` `&&` `\|\|` | `&&` `\|\|` `&` `;` | `@ = ( :` + `WORDCHARS`/`FIGNORE` |
| **oil** | `\|` | `< > >> <<< <> <& &> &>>` | `;` `&` `&&` `\|\|` `\|&` | same | `@ = ( :` |
| **tcsh** | `\|` | `< > >> << >& <&` | `;` `&` | `&&` `\|\|` `&` `;` | `( )` |
| **fish** | `\|` | `< > >> >>? >? <>&` | `;` | `and` `or` `not` (keywords) | `( )` |
| **elvish** | `\|` | `>` `<` `>>` `>>?` `<>>` | `;` | — (no list operators) | — |
| **nushell** | `\|` | `>` `<` `>>` `out>` `err>` `out+err>` | `;` | — | `( ) [ ] { }` |
| **powershell** | `\|` | `>` `>>` `2>` `2>&1` etc. | `;` | `&&` `\|\|` (PS7+) | `( ) { } ,` |
| **xonsh** | `\|` | `>` `>>` `<` `2>` `2>&1` `e>` `e<` etc. | `;` | `&` `\|` `&&` `\|\|` | `@()` `$()` `![]` |
| **cmd** | `\|` | `>` `>>` `<` | `&` `&&` `\|\|` | `&` `&&` `\|\|` | — |

**Key lexer implication**: POSIX shells share the bash operator grammar (multi-char greedy matching of `>>`, `<<`, `&&`, `||`, `|&`, `&>>`). Fish needs *keyword* operator recognition (`and`/`or`/`not` are bare words that act as operators — the tokenizer must match them at word boundaries, not as operator runes). Cmd uses `&` as a *command separator* (like `;` in POSIX), not just a background operator.

## Comment Syntax

| Shell | Starts | Terminates | Notes |
|-------|--------|------------|-------|
| **bash** | `#` | end of line (`\n`) | only when `#` is at word start (unquoted/unescaped) |
| **zsh** | `#` | end of line | same as bash |
| **oil** | `#` | end of line | same as bash |
| **tcsh** | `#` | end of line | `#` must be at word start |
| **fish** | `#` | end of line | `#` at word start |
| **elvish** | `#` | end of line | `#` at word start |
| **nushell** | `#` | end of line | also `#` inline after code; block `#` lines |
| **powershell** | `#` | end of line | also `<# ... #>` block comments (multi-line) |
| **xonsh** | `#` | end of line | Python `#` comments |
| **cmd** | `REM` / `::` | end of line | `REM` is a command; `::` is a label-style comment |

The v1 tokenizer handles `#`-to-newline comments in `COMMENT_STATE`. Cmd's `REM`/`::` and PowerShell's `<# #>` block comments need format-specific comment handling.

## String Interpolation (affects escaping quote state)

Whether the escaping-quote state has additional special characters beyond the escape char:

| Shell | Interpolation in `"..."` | Special chars in `"..."` |
|-------|:-:|--------------------------|
| **bash** | yes | `$`, `` ` `` |
| **zsh** | yes | `$`, `` ` `` |
| **oil** | yes | `$`, `` ` `` |
| **tcsh** | yes | `$`, backtick |
| **fish** | yes | `$`, `$(...)`, `(...)` |
| **elvish** | no | (concatenation instead) |
| **nushell** | only in `$"..."` | `$`, `(...)` |
| **powershell** | yes | `$variable`, `$(...)` |
| **xonsh** | yes (f-strings) | `{...}` |

For a *lexer* (not an expander), interpolation matters only insofar as the interpolating characters must not be confused with quote terminators. The tokenizer generally treats the quote char as the only terminator in the escaping-quote state and lets `$`/`` ` `` pass through as regular word characters — matching v1 behavior.

## POSIX vs Non-POSIX Summary

| Property | POSIX (bash, zsh, oil, tcsh) | Non-POSIX (fish, elvish, nushell) | Windows (powershell, cmd) |
|----------|------------------------------|-----------------------------------|---------------------------|
| Escape char | `\` | `\` | `` ` `` / `^` |
| Word splitting on `$var` | yes | **no** | n/a (PS) / no (cmd) |
| `;` command separator | yes | yes (fish, elvish, nushell) | yes (PS) / `&` (cmd) |
| `&&` / `\|\|` list operators | yes | keywords (fish) / none (elvish, nushell) | yes (PS7+, cmd) |
| Comment | `#` EOL | `#` EOL | `#` EOL (PS) / `REM`/`::` (cmd) |
| Backtick meaning | command subst (bash/zsh/oil/tcsh) | — | escape (PS) / n/a (cmd) |

## Choosing a Format

- **bash** → use the POSIX format (this is v1's default, `BASH_WORDBREAKS`).
- **zsh** → POSIX format + `RC_QUOTES` (`''`→`'` inside single quotes) + `WORDCHARS`/`FIGNORE` for word breaks.
- **oil (OSH)** → POSIX format (bash-compatible). YSH adds `r'...'`/triple-quoted string types.
- **tcsh** → POSIX format + backtick command substitution tracking + `!` history-expansion awareness + `$'...'`.
- **fish** → POSIX-like quotes but `\'` works inside single quotes; keyword operators `and`/`or`/`not`; no word splitting.
- **elvish** → `''` doubled-quote in single quotes; bareword charset; no POSIX list operators.
- **nushell** → extra string types (`r#'...'#`, backtick, `$'...'`, `$"..."`); metacharacter set for quoting.
- **powershell** → backtick escape; `''`/`""` doubled-quote; here-strings; `--%` stop-parsing.
- **xonsh** → Python string literals (raw, f-, p-, b-strings) + shell operators.
- **cmd** → `^` escape; `"`-only quotes; `&` separator; `REM`/`::` comments.

## References

- [architecture.md](architecture.md) — common token model, state machine, adding a format
- `format-*.md` — per-shell lexical details
