# Elvish Lexical Format

Lexical rules a command-line lexer needs for elvish. Elvish uses `'`/`"` quotes with `''` (doubled single quote) as the escape inside single quotes, a bareword charset, no word splitting, and no POSIX list operators.

> **Source of truth**: elvish docs ([String](https://elv.sh/ref/language.html#string), [Bareword](https://elv.sh/ref/language.html#bareword), [Function](https://elv.sh/ref/language.html#function), [Pipeline](https://elv.sh/ref/language.html#pipeline)) and elvish source (`pkg/parse/parse.go`). For broader elvish internals, use the **elvish** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` — **`''` → `'`** |
| escape | `\` (only inside `"..."`) | — |
| comment | `#` | `COMMENT_STATE` |
| wordbreak | `\|` `<` `>` `;` `(` `)` `[` `]` | `WORDBREAK_STATE` |

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
| `\|` | pipe / lambda parameter delimiter | pipeline delimiter (context-dependent — see [Lambdas and Brace Context](#lambdas-and-brace-context)) |
| `>` `<` `>>` `>>?` `<>>` | redirects | redirect |
| `;` | command separator | pipeline delimiter |
| `(` `)` | output capture delimiters | `WORDBREAK_OUTPUT_CAPTURE` — not a pipeline delimiter |
| `[` `]` | list literal / indexing delimiters | `WORDBREAK_BRACKET` — not a pipeline delimiter |

Elvish has **no** POSIX list operators (`&&`, `||`, `&`, etc.). Logical operations use the `and`/`or` commands (not keyword operators). The `&` character is used for map literals (`&key=value`) and option syntax, not as a background/list operator.

The wordbreak rune set is `|`, `<`, `>`, `;`, `(`, `)`, `[`, `]`. The metacharacters `{` and `}` are not wordbreak runes — they are handled by the `PostProcess` brace-context tracker (see [Lambdas and Brace Context](#lambdas-and-brace-context)).

### Output capture `(...)`

`(` and `)` are unambiguous word breaks — they always delimit words. `echo (ls)` tokenizes as `echo`, `(`, `ls`, `)`. The `Words()` merge rejoins adjacent tokens so `(ls)` becomes a single word.

### List literals `[...]` and indexing `var[0]`

`[` and `]` are word breaks at the rune level. `Words()` merges them back with adjacent tokens: `[a` and `b]` for `[a b]` (list literal with space), or `$var[0]` for indexing (all tokens adjoin, so they merge into one word). Neither `WORDBREAK_OUTPUT_CAPTURE` nor `WORDBREAK_BRACKET` is a pipeline delimiter, so they don't cause pipeline splitting.

## Comments

`#` to end of line. `#` must be at a word boundary.

```elvish
echo hello # a comment
```

## No Word Splitting

Elvish does not word-split on variable expansion. `$var` is a single value regardless of contents. The lexer splits on literal source whitespace only.

## Lambdas and Brace Context

Elvish uses `{...}` for both **lambda literals** and **braced lists**. The disambiguation rule (from `pkg/parse/parse.go` `lbrace()`):

- If the first rune after `{` is `|`, whitespace, `;`, `\r`, or `\n` → **lambda**
- Otherwise → **braced list** (e.g. `{a,b}`)

### Lambda syntax

```
Lambda = '{' [ '|' { (Compound | MapPair) { Space } } '|' ] Chunk '}'
```

- `{|a b| body}` — lambda with positional parameters `a`, `b`
- `{ |&opt=default| body }` — lambda with options
- `{ body }` — lambda with no parameters (whitespace after `{` required)
- `{a,b}` — braced list (no whitespace after `{`)

### The `|` ambiguity: pipe vs lambda parameter delimiter

`|` has two meanings in elvish:
1. **Pipeline separator** at the top level: `form1 | form2`
2. **Lambda signature delimiter** inside `{...}`: `{|params| body}`

Elvish's parser resolves this **contextually** via the recursive-descent call stack — when parsing inside `lambda()`, `|` means parameter delimiter; when parsing inside `Pipeline.parse()`, `|` means pipe. There is no lexer-level disambiguation.

### Current lexer limitation

The carapace-shlex tokenizer is a **flat** state machine with no nesting awareness. It classifies every `|` as `WORDBREAK_PIPE` regardless of brace context. This means:

- `bat | {|` produces: `WORD(bat)`, `WORDBREAK_PIPE(|)`, `WORD({)`, `WORDBREAK_PIPE(|)`, `WORD("")`
- The second `|` is incorrectly classified as a pipeline delimiter
- `CurrentPipeline()` splits at the wrong point, producing an empty pipeline
- The cursor position (after the second `|`) is seen as "start of a new command" rather than "inside a lambda parameter list"

This affects completion correctness when the cursor is inside `{|...|` (parameter declaration position).

### What elvish's own completion system does

Elvish's parser (`pkg/parse/parse.go`) is a handwritten recursive-descent parser with no separate lexer — it tracks context structurally via the call stack. The completion system (`pkg/edit/complete/`) uses `np.FindLeft()` to find the AST path from leaf to root at the cursor position, then pattern-matches upward to determine if the cursor is inside a `Lambda` primary's parameter list.

Notably, elvish's own `completeCommand` completer (line 68-73 of `completers.go`) has a TODO acknowledging it incorrectly triggers after `{|` — it offers command completions when the cursor is in parameter position. There is no dedicated "lambda parameter" completer in elvish.

## Edge Cases

- **`''` in single quotes**: the primary lexer deviation. The `NonEscapingQuoteEscapes` flag enables peek-ahead on `'` in `QUOTING_STATE` to handle `''` as a literal `'`.
- **`\` as bareword**: outside quotes, `\` is literal. The `EscapeNotBareword` flag (returns false for elvish) prevents the state machine from entering `ESCAPING_STATE` on `\` in `IN_WORD_STATE`.
- **Map/option syntax `&`**: `&` is not a list operator in elvish. Including it in the wordbreak set would break `&key=value` map literals.
- **No `COMP_WORDBREAKS`**: elvish has no equivalent env var.
- **QuoteWord**: elvish uses single-quote wrapping with `''` for literal `'` in `JoinWith`.
- **Brace/lambda context**: `{` and `}` are not wordbreak runes. `|` inside braces is a lambda parameter delimiter, not a pipe. The `PostProcess` post-pass reclassifies these — see [Lambdas and Brace Context](#lambdas-and-brace-context) above.
- **`^` line continuation**: elvish uses `^` followed by newline as line continuation (like `\` + newline in bash). The lexer currently treats `^` as a regular word character. This only affects multi-line input; single-line completion input is unaffected. Not yet handled.
- **Metacharacters not in wordbreaks**: `$`, `*`, `?`, `~`, `&`, `=`, `,` are metacharacters in elvish's grammar but are correctly treated as word characters by the lexer (they compound with adjacent text without breaking words). `$` starts a variable use, `*`/`?` are wildcards, `~` is tilde expansion, `&` introduces map pairs/options, `=` terminates map keys, `,` separates braced list elements — none of these cause word breaks in practice.

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model
- [format-zsh.md](format-zsh.md) — zsh `RC_QUOTES` (same `''` behavior)

## Related Skills

- **elvish** skill — elvish completion, editor, styling, language, startup
- **carapace-dev** skill → `references/shell-elvish.md` — carapace's elvish output formatting
