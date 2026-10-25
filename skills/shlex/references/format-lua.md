# Lua Lexical Format

Lexical rules a command-line lexer needs for Lua. The format exists for
**cmd-clink snippets**, which are Lua scripts (e.g. `argmatcher git
:addarg(x, 'y')`); it is not a general Lua front-end. Both `'...'` and
`"..."` support backslash escapes, and identifiers are the only barewords.

> **Source of truth**: Lua 5.4 reference manual, §3.1 (Lexical Conventions)
> and §3.1.1 (String literals). For broader clink internals, use the
> **cmd-clink** skill. For cross-shell comparison, see
> [comparison.md](comparison.md).

## Classification

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` — fish/zsh-style `''` and `\'` rules |
| escape | `\` | — |
| comment | `#` (inherited from the POSIX base; **not** Lua syntax) | `COMMENT_STATE` |
| wordbreak | `;` `(` `)` `{` `}` `,` `.` `=` `:` | `WORDBREAK_STATE` |

Lua has **no shell operators**: no pipes, no redirects, no list operators.
`;` is only a statement separator and is classified
`WORDBREAK_LIST_SEQUENTIAL` (the only pipeline delimiter). `(` and `)` are
`WORDBREAK_BRACKET`; all remaining punctuation is `WORDBREAK_UNKNOWN`.

## Quotes

### Both quote styles are equivalent

Lua treats `'...'` and `"..."` identically (§3.1.1), and \'
is an escape in **both**.

| Sequence | Meaning |
|----------|---------|
| `\\` | backslash |
| `\"` | double quote |
| `\'` | single quote |
| `\n` | newline |
| `\t`, `\r`, `\a`, `\b`, `\f`, `\v` | control characters |
| literal newline | **not allowed** inside short strings (must be an escape) |
| `\xHH` | hex byte |
| `\ddd` | decimal byte (up to 3 digits) |
| `\z` | skip following whitespace |

The tokenizer never evaluates escape sequences; it tracks only quotation
state. The relevant simplifications (both documented model limitations):

- **`''` in single quotes**: Lua itself only concatenates adjacent
  string literals (`'it''s'` = "it" ++ "s" = `its`). The lexer instead
  treats `''` as an escaped quote (fish/zsh `RC_QUOTES` style), keeping
  `'it''s'` as the single word `it's` so completion can resume inside
  the string. The same flags make `\'` an escape, matching real Lua
  (`'don\'t'` → `don't`).
- **`hex/decimal/\z` escapes**: not modeled. The tokenizer still consumes
  `\` as an escape inside quotes, so `"\x41"` dequotes to `x41` (the
  backslash is dropped and the rune taken literally) rather than the byte
  `A`. Round-trip through `Split`/`Join` is therefore not preserved for
  these sequences; only `\"`, `\'`, `\\`, and newline matters.
- **raw newline**: Lua forbids a literal unescaped newline inside short
  strings. The lexer keeps it as a word character and `luaQuoteWord`
  re-emits it as `\n` on join.

### Single quotes `'...'`

Fish-style: `''` and `\'` produce one literal `'` and stay in
`QUOTING_STATE` (with `\\` as well). Requires
`NonEscapingQuoteEscapes` **and** `NonEscapingQuoteBackslashEscapes`.

### Double quotes `"..."`

Backslash escapes the next rune (`EscapingQuoteEscapeChars` returns nil).

## Barewords

Lua identifiers are the only barewords, and they are ASCII-only:

```
[(A-Za-z_)][(A-Za-z0-9_)]*
```

A leading digit is invalid (`123abc` lexes as the number `1` followed by
the identifier `abc`), and unicode letters are invalid — both are handled
by `isLuaBareword` (any non-matching word gets quoted via `luaQuoteWord`).

## Operators and Wordbreak Punctuation

| Punctuation | Meaning | WordbreakType | Pipeline delimiter? |
|-------------|---------|---------------|---------------------|
| `;` | statement separator | `WORDBREAK_LIST_SEQUENTIAL` | yes |
| `(` `)` | function call / grouping | `WORDBREAK_BRACKET` | no |
| `,` | list element separator | `WORDBREAK_UNKNOWN` | no |
| `.` `:` | method access | `WORDBREAK_UNKNOWN` | no |
| `=` | table / default value | `WORDBREAK_UNKNOWN` | no |

Note that `Words()` merges tokens adjacent to `(`/`.`/`,`, so
`:addarg(x,` is one word — this matches how POSIX shells treat
`(` with default `COMP_WORDBREAKS`. The per-token `WordbreakType` still
identifies brackets for JSON consumers.

## Unsupported Syntax (Model Limitations)

The flat state machine does not lex these; leave them alone and do not
feed them to the lexer:

- **line comments** (`--`) and **block comments** (`--[[ ... ]]`)
- **long strings** (`[[ ... ]]`, `[= ... =]`)
- **hex / decimal / `\z` escapes** inside strings (the `\` still escapes
  the following rune when tokenizing, so round-tripping \x41 etc. yields
  the decoded value only if the consumer dequotes it)
- `#` is a POSIX-style comment in the classifier, not Lua syntax

## QuoteWord / Escape

- `luaQuoteWord`: identifiers stay bare; everything else is wrapped in
  double quotes with `\"`, `\\`, and `\n` escapes.
- `shlex.Escape(..., shlex.Lua)` escapes string *interior* content (no
  surrounding quotes supplied by the lexer): `\'`, `\\`, and `\n`.

## Edge Cases

- **Both quote styles escape**: unlike POSIX (backslash inside `'...'` is
  literal), Lua's `'` and `"` behave the same; the lexer approximates this
  with the fish flags for single quotes.
- **No `COMP_WORDBREAKS`**: Lua is not driven by an env-var wordbreak set.
- **Lexer design note**: Lua is not a shell in the pipeline sense, so
  `IsRedirect()` is always false and `FilterRedirects()` is a no-op (the
  numeric fd adjacency rule is inert).

## References

- [comparison.md](comparison.md) — cross-shell comparison
- [architecture.md](architecture.md) — common token model
- [format-fish.md](format-fish.md) — same single-quote escape flags
- [format-elvish.md](format-elvish.md) — adjacent-brackets comparison

## Related Skills

- **cmd-clink** skill — clink argmatcher, completion, cmd integration
  (the consumer of Lua snippets)
- **hilbish** skill — a shell *embedded in Lua* (different format with
  bash-language lexing)
