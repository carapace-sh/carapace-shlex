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

- **Backtick as quote** (not escape): opposite of PowerShell. Don't confuse the two formats.
- **`$` prefix on quotes**: `$'...'` and `$"..."` — the `$` adjoins the quoted segment; `Words()` merges them.
- **No `COMP_WORDBREAKS`**: nushell has no equivalent env var.
- **C-style escapes produce real characters**: `\n` in `"..."` produces a newline character (0x0A), not the literal text `n`. Implemented via `EscapingQuoteUnescaper`.
- **Stream redirect operators**: `out>`, `err>`, `o+e>`, `e>|`, etc. are multi-rune operators. Implemented via `PostProcess`. Only bare words are merged — quoted strings like `'out'` are not treated as stream operators.
- **Quoted words before `>`**: `'out'>bar` is a string literal `out` followed by a plain redirect `>`, not a stream redirect operator. The `PostProcess` check `t.Value == t.RawValue` prevents merging quoted words.

## Deferred Features

Two features require multi-rune lookahead in the tokenizer's core state machine and are documented here for future implementation.

### 1. Raw Strings `r#'...'#`

#### What they are

Nushell raw strings are delimited by `r#`...`'#` (or `r##`...`'##`, `r###`...`'###`, etc.). They behave like single-quoted strings (no escapes) but can contain single quotes. The number of `#` symbols in the opener and closer must match.

```nu
r#'Raw strings can contain 'quoted' text.'#    # → Raw strings can contain 'quoted' text.
r##'I can use '#' in a raw string'##            # → I can use '#' in a raw string
r#''#                                          # → (empty string)
```

#### Current (wrong) behavior

The tokenizer treats `r` as a regular word character and `#` as either a word character (inside `IN_WORD_STATE`) or a comment (from `START_STATE`). So `r#'hello'#` is tokenized as:

- `r` → word char (IN_WORD_STATE)
- `#` → word char (not START_STATE, so not a comment — this is the coincidence)
- `'` → enters QUOTING_STATE
- `hello` → literal content
- `'` → exits QUOTING_STATE → IN_WORD_STATE
- `#` → word char

Result: one WORD_TOKEN with `Value="r#hello#"` and `RawValue="r#'hello'#"`. The `r#` prefix and `'#` suffix are included in the value.

Expected: `Value="hello"`, `RawValue="r#'hello'#"`.

#### How nushell's lexer handles it

In nushell's `lex_item` (`crates/nu-parser/src/lex.rs`), the raw-string check happens **before** the comment check:

```rust
} else if c == b'r' && input.get(*curr_offset + 1) == Some(b'#').as_ref() {
    let lex_result = lex_raw_string(input, curr_offset, span_offset);
    ...
}
```

`lex_raw_string` counts consecutive `#` chars after `r` to determine `prefix_sharp_cnt`, expects a `'` after them, then scans forward looking for a closing `'` followed by the same number of `#` chars. The entire `r#'...'#` sequence is consumed as a single `TokenContents::Item`.

#### Implementation plan

The tokenizer's `scanStream` state machine processes one rune at a time via `ReadRune`/`UnreadRune`. To support raw strings, we need multi-rune lookahead in the `START_STATE` and `IN_WORD_STATE` handlers:

1. **Detect the `r#` pattern**: When the current rune is `r` and the state is `START_STATE` or `IN_WORD_STATE`, peek ahead to check if the next rune is `#`. If so, count consecutive `#` runes to determine `prefix_sharp_cnt`.

2. **Validate the opener**: After the `#` sequence, expect a `'`. If not found, treat `r` as a regular word char and `#` as a comment/word char as appropriate.

3. **Scan to the closer**: Read forward until finding a `'` followed by exactly `prefix_sharp_cnt` `#` runes. Everything between the opener `'` and the closer `'#` is the raw string content.

4. **Emit the token**: Set `Token.Value` to the raw content (between opener and closer), `Token.RawValue` to the full `r#'...'#` source text, and `Token.State` to `IN_WORD_STATE` (the state after closing).

5. **Handle EOF in raw string**: If the closer is not found, emit the token with whatever content was consumed and set state to a new `RAW_STRING_STATE` (or reuse `QUOTING_STATE` for completion purposes — an unclosed raw string behaves like an unclosed quote).

**Affected files**:
- `shlex.go` — `scanStream()` `START_STATE` and `IN_WORD_STATE` handlers: add `r#` detection before the default/wordbreak cases
- `format.go` — possibly a new `Format` interface method like `RawStringOpeners() bool` or add raw-string support to the classifier
- `format_nushell.go` — enable raw string support

**Key challenge**: The state machine is shared across all formats. Raw-string detection must be opt-in per format (only nushell has raw strings). This could be:
- A new `Format` interface method (e.g. `SupportsRawStrings() bool`)
- A new rune class (e.g. `rawStringOpenerRuneClass`) triggered by a multi-rune classifier
- A `PostProcessor` approach — but this won't work because by the time PostProcess runs, the tokens are already split incorrectly (the `#` may have been classified as a comment)

The cleanest approach is likely a multi-rune lookahead in `scanStream`, gated by a format check, similar to how `NonEscapingQuoteEscapes()` gates the single-quote peek logic.

**The `#`-not-a-comment issue**: Currently `#` only starts a comment from `START_STATE` (word boundary). Inside `IN_WORD_STATE`, `#` falls through to the `default` case and is added as a word char. This means `r#'hello'#` does not accidentally trigger a comment. However, `r#` at a word boundary (e.g. after a space) would have `#` classified as `commentRuneClass` from `START_STATE`, which would start a comment instead of a raw string. The fix must check for `r#` **before** the `commentRuneClass` check in `START_STATE`.

**Test cases to add**:
- `r#'hello'#` → value `hello`
- `r#''#` → value `` (empty)
- `r##'contains '#'# here'##` → value `contains '#'# here`
- `r#'unclosed` → state indicates open raw string (for completion)
- `echo r#'hello'#` → words `[echo hello]`
- `r#'it's a test'#` → value `it's a test` (single quotes allowed inside)

### 2. Hex and Unicode Escapes `\xHH` and `\u{X...}`

#### What they are

Inside double-quoted strings (`"..."` and `$"..."`), nushell supports:

| Escape | Format | Result |
|--------|--------|--------|
| `\xHH` | exactly 2 hex digits | single byte (0x00–0xFF) |
| `\u{X...}` | 1-6 hex digits in braces, max 0x10FFFF | Unicode codepoint (UTF-8 encoded) |

```nu
"\x41\x42\x43"           # → ABC
"\u{1F600}"              # → 😀
"\u{0041}"               # → A
$"hello\n($name)"        # → hello\nworld (interpolated)
```

#### Current (wrong) behavior

The `EscapingQuoteUnescaper` interface handles single-rune escapes (`\n`, `\t`, etc.) but cannot handle multi-rune escapes like `\x41` or `\u{1F600}` because the `ESCAPING_QUOTED_STATE` handler only receives one rune after the backslash. When it sees `x` or `u`, the unescaper returns `handled=false`, so both `\` and `x` (or `u`) are kept literally.

Result: `"\x41"` produces value `\x41` instead of `A`.

#### How nushell's parser handles it

In `unescape_string` (`crates/nu-parser/src/parse_literals.rs`):

```rust
Some(b'x') => {
    match parse_hex_escape(bytes, idx, span) { ... }  // reads exactly 2 hex digits
}
Some(b'u') => {
    match parse_unicode_escape(bytes, idx, span) { ... }  // reads {X...}
}
```

`parse_hex_escape` reads exactly 2 hex digits after `\x`. `parse_unicode_escape` reads `{`, then 1-6 hex digits, then `}`, validating the codepoint ≤ 0x10FFFF.

#### Implementation plan

The `EscapingQuoteUnescaper` interface is called from `ESCAPING_QUOTED_STATE` with a single rune. To support multi-rune escapes, we need the unescaper to be able to consume additional runes from the tokenizer. Two approaches:

**Option A: Multi-rune unescaper (preferred)**

Extend `EscapingQuoteUnescaper` with a method that takes a `runeReader` or similar interface, allowing it to consume additional runes:

```go
type EscapingQuoteUnescaper interface {
    EscapingQuoteUnescape(r rune) (replacement string, handled bool)
    // EscapingQuoteUnescapeMulti is called when EscapingQuoteUnescape returns
    // handled=false. It receives the first rune and a peekable reader, allowing
    // the format to consume additional runes for multi-rune escapes like \xHH.
    // Returns the replacement string and the number of additional runes consumed
    // (beyond the first rune). If not handled, returns (0, false).
    EscapingQuoteUnescapeMulti(r rune, reader *tokenizer) (replacement string, extraConsumed int, handled bool)
}
```

In the `ESCAPING_QUOTED_STATE` handler:
1. Call `EscapingQuoteUnescape(nextRune)` first — handles single-rune escapes.
2. If not handled, call `EscapingQuoteUnescapeMulti(nextRune, t)` — handles `\xHH` by reading 2 more runes, `\u{...}` by reading until `}`.
3. If still not handled, keep both `\` and the rune literally (current behavior).

The `tokenizer` already has `ReadRune`/`UnreadRune` methods. The multi-rune unescaper would read additional runes directly from the tokenizer, updating `t.index` and `token.RawValue`.

**Option B: Peek-based approach**

Give the unescaper access to a `PeekRune(n int) (rune, bool)` method that peeks ahead without consuming. The unescaper returns how many additional runes to consume. This is cleaner but requires adding a peek buffer to the tokenizer (currently it only has read/unread).

**Option C: State-machine extension**

Add new states like `ESCAPING_HEX_STATE` and `ESCAPING_UNICODE_STATE` to the state machine, with format-gated transitions. This is more invasive but keeps the single-rune-at-a-time model.

**Affected files**:
- `format.go` — extend `EscapingQuoteUnescaper` interface
- `shlex.go` — `ESCAPING_QUOTED_STATE` handler: add multi-rune escape logic
- `format_nushell.go` — implement `\xHH` and `\u{X...}` in the unescaper

**Key challenges**:
- The tokenizer's `RawValue` must include all consumed runes (the full `\x41` or `\u{1F600}`), while `Value` gets only the replacement character.
- Invalid escapes (`\x4`, `\x4z`, `\u{110000}`, `\u{6e`) are parse errors in nushell. shlex should be lenient: if the hex digits are missing or invalid, keep the backslash and the escape letter literally.
- `EscapingQuoteEscapeChars` (used by fish) and `EscapingQuoteUnescaper` are mutually exclusive — the unescaper takes priority. The multi-rune extension only applies to formats implementing `EscapingQuoteUnescaper`.

**Test cases to add**:
- `"\x41\x42\x43"` → value `ABC`
- `"\x00"` → value `\x00` (NUL)
- `"\xFF"` → value `\xFF` (byte 255)
- `"\u{1F600}"` → value `😀`
- `"\u{0041}"` → value `A`
- `"\u{0}"` → value `\x00` (NUL via unicode)
- `"\x4"` → value `\x4` (incomplete — lenient, keep literal)
- `"\x4z"` → value `\x4z` (invalid hex — lenient, keep literal)
- `"\u{110000}"` → value `\u{110000}` (out of range — lenient, keep literal)
- `"\u{6e"` → value `\u{6e"` (missing `}` — lenient, keep literal, quote stays open)
- `$"\x41"` → value `$A` (interpolated double-quote with hex escape)

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model

## Related Skills

- **nushell** skill — nushell completion, Reedline, quoting, types, externs, config
- **carapace-dev** skill → `references/shell-nushell.md` — carapace's nushell output formatting
