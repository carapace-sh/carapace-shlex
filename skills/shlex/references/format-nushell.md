# Nushell Lexical Format

Lexical rules a command-line lexer needs for nushell. Nushell has six string types (single-quoted, double-quoted, raw strings `r#'...'#`, backtick, interpolated `$'...'` and `$"..."`), a broad metacharacter set, `|`/`;` operators, and C-style escapes in double quotes.

> **Source of truth**: nushell docs ([Working with strings](https://www.nushell.sh/book/working_with_strings.html), [Syntax](https://www.nushell.sh/book/commands.html)). For broader nushell internals, use the **nushell** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` (C-style escapes) |
| non-escaping quote | `'` | `QUOTING_STATE` (literal) |
| raw string | `r#'...'#` | special raw-string state |
| backtick | `` ` `` | backtick-quote state |
| interpolated single | `$'...'` | quoting state (no escapes) |
| interpolated double | `$"..."` | escaping-quoting state (C-style) |
| comment | `#` | `COMMENT_STATE` |

## String Types

Nushell has the richest string type system of any supported shell. Six types a lexer must recognize:

| Type | Syntax | Escapes | Lexer state |
|------|--------|---------|-------------|
| Single-quoted | `'text'` | none | `QUOTING_STATE` |
| Double-quoted | `"text"` | C-style `\" \' \\ \n \t \u{X}` | `QUOTING_ESCAPING_STATE` |
| Raw string | `r#'text'#` | none | raw-string state |
| Bare word | `word` | none | `IN_WORD_STATE` |
| Backtick | `` `text` `` | none | backtick-quote state |
| Interpolated single | `$'...'` | none (but `()` evaluated) | quoting state |
| Interpolated double | `$"..."` | C-style (but `()` evaluated) | escaping-quoting state |

### Single quotes `'...'`

Literal, no escapes, can span multiple lines. Cannot contain single quotes.

```nu
'hello world'       # hello world
'$HOME \n \t'       # $HOME \n \t  (all literal)
```

### Double quotes `"..."`

C-style backslash escapes:

| Sequence | Meaning |
|----------|---------|
| `\"` | double quote |
| `\'` | single quote |
| `\\` | backslash |
| `\/` | forward slash |
| `\b` `\f` `\r` `\n` `\t` | control chars |
| `\u{X...}` | Unicode (1-6 hex digits) |

```nu
"hello\nworld"      # two lines
"say \"hello\""     # say "hello"
"cost: \$5"         # cost: \$5  (\$ is NOT special in nushell)
```

**Important**: `\$` is not a special escape in nushell double-quoted strings. `$` is only special in interpolated strings (`$"..."`).

### Raw strings `r#'...'#`

Behaves like single-quoted (no escapes) but can contain single quotes. The `#` symbols are part of the delimiter — additional `#` symbols allow nesting raw strings containing `'#`.

```nu
r#'Raw strings can contain 'quoted' text.'#
r###'r##'nested raw string.'##'###
```

For the lexer: `r#'` is a three-rune opener (or more with additional `#`s). The closing delimiter is `'#` (matching the number of `#`s). **Critical**: the `#` in the opener is not a comment — the classifier must recognize `r#` as a raw-string opener before classifying `#` as a comment rune. This requires a multi-rune lookahead or a pre-scan.

### Backtick strings `` `...` ``

No escapes, can include whitespace. Cannot contain unmatched backticks. In command position, still interpreted as command or path.

```nu
`./my dir`          # path with spaces
`ls`                # run external ls
```

For the lexer: backtick is a quote character that enters a non-escaping quote state closed by another backtick. Distinct from PowerShell where backtick is an *escape* character.

### Interpolated strings `$'...'` and `$"..."`

Like single/double quotes but with `()` expression interpolation. `$'...'` has no escapes; `$"..."` has C-style escapes.

```nu
let name = "Alice"
$'Hello, ($name)'           # Hello, Alice
$"greetings, ($name)"       # greetings, Alice
```

For the lexer: the `$'` and `$"` openers are two-rune sequences. The `$` is a word character followed by the quote. The interpolation `()` means `(` and `)` have special meaning inside these strings, but for state-tracking purposes the quote char is the terminator (matching how v1 treats `$` as a word char before `'`).

## Escape Character

Backslash `\`:
- **Outside quotes**: no escape meaning (bare words only allow word characters; special chars require quoting).
- **Inside `"..."` and `$"..."`**: C-style escapes (full set below).
- **Inside `'...'`, `r#'...'#`, `` `...` ``, `$'...'`**: none.

### Full Escape Set in Double Quotes

Nushell's `unescape_string` recognizes these escapes inside `"..."`  and `$"..."`:

| Sequence | Result |
|----------|--------|
| `\"` | double quote |
| `\'` | single quote |
| `\\` | backslash |
| `\/` | forward slash |
| `\b` | backspace (0x08) |
| `\f` | form feed (0x0C) |
| `\r` | carriage return (0x0D) |
| `\n` | newline (0x0A) |
| `\t` | tab (0x09) |
| `\0` | NUL (0x00) |
| `\a` | bell (0x07) |
| `\e` | escape (0x1B) |
| `\(` `\)` | literal parens |
| `\{` `\}` | literal braces |
| `\$` | literal dollar |
| `\^` | literal caret |
| `\#` | literal hash |
| `\|` | literal pipe |
| `\~` | literal tilde |
| `\xHH` | hex byte (2 hex digits) |
| `\u{X...}` | Unicode (1-6 hex digits, max 0x10FFFF) |

Unrecognized escapes are a parse error in nushell. shlex is lenient: it keeps both `\` and the rune.

The `EscapingQuoteUnescaper` interface provides the full escape set to the tokenizer. `\xHH` and `\u{X}` are not yet handled by shlex (deferred — they require multi-rune lookahead in the escape state).

## Metacharacters

Characters requiring quoting in nushell:

```
(space) { } ( ) [ ] $ " ' ` < > & | ; # \
```

When a completion value contains any of these, carapace's nushell formatter quotes it. For the lexer, these are the characters that, when unquoted, have syntactic meaning — but only `|`, `;`, `<`, `>`, `(`, `)`, `[`, `]`, `{`, `}` are word breaks or syntax.

## Word Breaks and Operators

| Operator | Meaning | Type |
|----------|---------|------|
| `\|` | pipe | pipeline delimiter |
| `>` `<` `>>` | redirects | redirect |
| `out>` `err>` `out+err>` | stream redirects | redirect |
| `o>` `e>` `o+e>` | short stream redirects | redirect |
| `err>\|` `e>\|` | stderr pipe | pipeline delimiter (PIPE_WITH_STDERR) |
| `out+err>\|` `o+e>\|` | combined pipe | pipeline delimiter (PIPE_WITH_STDERR) |
| `;` | command separator | pipeline delimiter |

Nushell has **no** POSIX list operators (`&&`, `||`, `&`). `(`, `)`, `[`, `]`, `{`, `}` are syntax for expressions, lists, and blocks — they break words but are not pipeline operators.

### Stream Redirect PostProcessing

The tokenizer produces `out` as a WORD_TOKEN and `>` as a separate WORDBREAK_TOKEN because the rune classifier only handles single-rune word breaks. The `PostProcess` step merges adjacent word+wordbreak sequences where the word matches a known stream prefix (`out`, `err`, `o`, `e`, `out+err`, `o+e`) and the wordbreak starts with `>`. If the wordbreak includes a `|` suffix (e.g. `>|`), the merged token is classified as `WORDBREAK_PIPE_WITH_STDERR` instead of a redirect.

## Comments

`#` to end of line. Nushell also supports `#` inline after code and block comments via consecutive `#` lines. For the lexer, `#` at a word boundary starts a `COMMENT_STATE` terminated by `\n`.

**Raw string conflict**: `r#'...#...'` contains `#` that is not a comment. The classifier must check for `r#` before treating `#` as a comment.

## Carapace Integration

Carapace's `nushell.Patch()` uses `shlex.Split` to strip quotes from args passed by nushell's completer:

```go
// carapace internal/shell/nushell/patch.go
switch arg[0] {
case '"', "'"[0]:
    if tokens, err := shlex.Split(arg); err == nil {
        args[index] = tokens[0].Value
    }
case '`':
    args[index] = strings.Trim(arg, "`")
}
```

Nushell passes args **with quoting intact** (unlike bash which strips quotes). The Patch phase uses the POSIX lexer to dequote single/double-quoted args and trims backticks directly.

## Edge Cases

- **`r#'...'#` raw strings**: the `#`-not-a-comment issue. Needs multi-rune opener recognition. Deferred — the `r#` prefix is currently treated as word characters, and `#` inside `r#'...'` is a word character (not a comment) by coincidence because `#` only starts a comment from `START_STATE`.
- **Backtick as quote** (not escape): opposite of PowerShell. Don't confuse the two formats.
- **`$` prefix on quotes**: `$'...'` and `$"..."` — the `$` adjoins the quoted segment; `Words()` merges them.
- **No `COMP_WORDBREAKS`**: nushell has no equivalent env var.
- **C-style escapes produce real characters**: `\n` in `"..."` produces a newline character (0x0A), not the literal text `n`. Implemented via `EscapingQuoteUnescaper`.
- **Stream redirect operators**: `out>`, `err>`, `o+e>`, `e>|`, etc. are multi-rune operators. Implemented via `PostProcess`.
- **`\xHH` and `\u{X}` escapes**: deferred — require multi-rune lookahead in the escape state.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **nushell** skill — nushell completion, Reedline, quoting, types, externs, config
- **carapace-dev** skill → `references/shell-nushell.md` — carapace's nushell output formatting
