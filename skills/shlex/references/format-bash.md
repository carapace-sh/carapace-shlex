# Bash Lexical Format

Lexical rules a command-line lexer needs for bash: quote characters, escape semantics, word breaks, operators, and comments. This is the v1 default format and the baseline for the POSIX family.

> **Source of truth**: bash manual ([Quoting](https://www.gnu.org/software/bash/manual/html_node/Quoting.html), [Redirections](https://www.gnu.org/software/bash/manual/html_node/Redirections.html), [Pipelines](https://www.gnu.org/software/bash/manual/html_node/Pipelines.html), [Lists](https://www.gnu.org/software/bash/manual/html_node/Lists.html)). For broader bash internals, use the **bash** skill. For cross-shell comparison, see [comparison.md](comparison.md).

## Classification

Bash is the **POSIX baseline**. V1's hardcoded rune sets are exactly the bash format:

```go
// v1 shlex.go
spaceRunes            = " \t\r\n"
escapingQuoteRunes    = `"`      // QUOTING_ESCAPING_STATE
nonEscapingQuoteRunes = "'"      // QUOTING_STATE
escapeRunes           = `\`
commentRunes          = "#"
BASH_WORDBREAKS       = " \t\r\n" + `"'@><=;|&(:`
```

| Rune class | Runes | Tokenizer state |
|------------|-------|-----------------|
| space | ` \t\r\n` | word delimiter |
| escaping quote | `"` | `QUOTING_ESCAPING_STATE` |
| non-escaping quote | `'` | `QUOTING_STATE` |
| escape | `\` | `ESCAPING_STATE` / `ESCAPING_QUOTED_STATE` |
| comment | `#` | `COMMENT_STATE` |
| wordbreak | `"'@><=;|&(:` + `COMP_WORDBREAKS` | `WORDBREAK_STATE` |

## Quotes

### Single quotes `'...'`

Non-escaping (literal). No escape mechanism at all — a single quote cannot appear inside single quotes. The tokenizer enters `QUOTING_STATE` and only the closing `'` exits.

```bash
echo 'hello world'      # hello world
echo '$HOME \n \t'      # $HOME \n \t  (all literal)
echo 'it'\''s'          # it's (close, escaped quote, reopen)
```

### Double quotes `"..."`

Escaping quote — the tokenizer enters `QUOTING_ESCAPING_STATE`. Backslash retains special meaning **only** before `$`, `` ` ``, `"`, `\`, and `newline`. Those pairs are removed; backslash before other chars is kept literal.

```bash
echo "$HOME"            # /home/user
echo "say \"hello\""    # say "hello"  (\" removed)
echo "cost: \$5"        # cost: $5     (\$ removed)
echo "hello\nworld"     # hello\nworld (\n literal — not an escape here)
```

Special characters inside double quotes: `$` (parameter expansion) and `` ` `` (command substitution). These are **not** quote terminators — the tokenizer treats them as regular word characters, matching v1.

### ANSI-C quoting `$'...'`

Bash-specific. Behaves like single quotes but processes backslash escape sequences (`\n`, `\t`, `\xHH`, `\uHHHH`, etc.). For the lexer, the `$'` opener is a two-rune sequence that enters an escaping-quote-like state with the closing `'`. V1 does **not** specially handle `$'...'` — it treats `$` as a word character and `'` as a non-escaping quote, so `$'...'` lexes as a bareword `$` followed by a single-quoted string. This is usually adequate for completion (the `$` adjoins the quoted segment and `Words()` merges them).

### Locale translation `$"..."`

Like double quotes but with `gettext` translation. Lexically identical to `"..."` for the tokenizer — the leading `$` is a word character.

## Escape Character

Backslash `\`:

- **Outside quotes** (`ESCAPING_STATE`): the next rune is literal (added to `Value`, the backslash itself is not). At EOF after `\`, the token is returned with whatever was accumulated.
- **Inside double quotes** (`ESCAPING_QUOTED_STATE` → back to `QUOTING_ESCAPING_STATE`): only special before `$`, `` ` ``, `"`, `\`, `newline`. The tokenizer consumes the backslash and adds the next rune literally.
- **Inside single quotes**: not an escape (literal).

```bash
echo \$HOME             # $HOME
echo a\ b               # a b (escaped space → one word)
echo "a\"b"             # a"b
```

## Word Breaks and Operators

`BASH_WORDBREAKS = " \t\r\n" + "'\"@><=;|&(:"`. The classifier filters out runes already classified (quotes, escape, comment, space) so the effective wordbreak set is `@><=;|&(:` plus any custom `COMP_WORDBREAKS` runes.

### `COMP_WORDBREAKS`

The environment variable `COMP_WORDBREAKS` overrides the default wordbreak set. V1 reads it in `newDefaultClassifier`:

```go
wordbreakRunes := BASH_WORDBREAKS
if wordbreaks := os.Getenv("COMP_WORDBREAKS"); wordbreaks != "" {
    wordbreakRunes = wordbreaks
}
// filter out runes already classified as space/quote/escape/comment
```

This is critical for completion: bash's `COMP_WORDBREAKS` determines where the word being completed starts. Carapace's `bash.Patch()` uses `WordbreakPrefix()` to extract the prefix up to the last wordbreak.

### Operators (WordbreakType)

| Operator | RawValue | WordbreakType | Category |
|----------|----------|---------------|----------|
| `<` | `<` | `WORDBREAK_REDIRECT_INPUT` | redirect |
| `>` | `>` | `WORDBREAK_REDIRECT_OUTPUT` | redirect |
| `>>` | `>>` | `WORDBREAK_REDIRECT_OUTPUT_APPEND` | redirect |
| `&>` / `>&` | `&>` / `>&` | `WORDBREAK_REDIRECT_OUTPUT_BOTH` | redirect |
| `&>>` | `&>>` | `WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND` | redirect |
| `<<<` | `<<<` | `WORDBREAK_REDIRECT_INPUT_STRING` | redirect |
| `<&` | `<&` | `WORDBREAK_REDIRECT_INPUT_DUPLICATE` | redirect |
| `<>` | `<>` | `WORDBREAK_REDIRECT_INPUT_OUTPUT` | redirect |
| `\|` | `\|` | `WORDBREAK_PIPE` | pipeline |
| `\|&` | `\|&` | `WORDBREAK_PIPE_WITH_STDERR` | pipeline |
| `&` | `&` | `WORDBREAK_LIST_ASYNC` | list |
| `;` | `;` | `WORDBREAK_LIST_SEQUENTIAL` | list |
| `&&` | `&&` | `WORDBREAK_LIST_AND` | list |
| `\|\|` | `\|\|` | `WORDBREAK_LIST_OR` | list |

Multi-char operators are matched greedily in the `WORDBREAK_STATE` (consecutive wordbreak runes accumulate into one `WORDBREAK_TOKEN`).

`IsPipelineDelimiter()` → `|`, `|&`, `&`, `;`, `&&`, `||` — these split pipelines in `CurrentPipeline()`.
`IsRedirect()` → the eight redirect operators — these are stripped in `FilterRedirects()`.

### File-descriptor prefixes

`FilterRedirects()` also strips the numeric fd prefix of a redirect (e.g., the `2` in `2>`):

```go
// tokenslice.go FilterRedirects
if _, err := strconv.Atoi(token.RawValue); err == nil {
    if wordbreakType(t[index+1]).IsRedirect() {
        continue
    }
}
```

## Comments

`#` starts a comment when at a word boundary (unquoted/unescaped). The tokenizer enters `COMMENT_STATE` and consumes until `\n` (which terminates the comment and returns to `START_STATE`). The `lexer.Next` skips `COMMENT_TOKEN`.

```bash
echo hello # a comment
echo hello#world       # hello#world (no comment — # is mid-word)
```

Note: v1's tokenizer treats `#` as a comment starter *whenever* it appears at `START_STATE` (word boundary), which matches bash's rule that `#` must be at the start of a word.

## Quotation State for Completion

The token's `State` field tells completion code whether the cursor is inside an open quote:

| Input | Last token State | Meaning |
|-------|------------------|---------|
| `echo foo` | `IN_WORD_STATE` | normal word |
| `echo "foo` | `QUOTING_ESCAPING_STATE` | inside open `"` |
| `echo 'foo` | `QUOTING_STATE` | inside open `'` |
| `echo foo\` | `ESCAPING_STATE` | escape at EOF |

Carapace's bash snippet uses 3-stage retry (`''`, `'"`, `"`) to handle open-quote scenarios.

## Edge Cases

- **`@` wordbreak quirk**: `@` is a wordbreak but `WordbreakPrefix()` skips it — bash does not include `@` in the completion prefix.
- **Empty trailing token**: at EOF after spaces or a wordbreak, the tokenizer emits an empty `WORD_TOKEN` at the cursor position so completion has a word to complete.
- **Adjacent quoted segments**: `a"b"'c'` produces three tokens that `Words()` merges into one word `abc`.
- **Escape at EOF**: `\` at end of input returns the token without the trailing backslash in `RawValue`.

## References

- `shlex.go` — v1 tokenizer (the bash format)
- `wordbreak.go` — `BASH_WORDBREAKS`, operator classification
- [comparison.md](comparison.md) — how bash compares to other shells
- [architecture.md](architecture.md) — common token model

## Related Skills

- **bash** skill — bash completion system, Readline, expansion, execution, startup
- **carapace-dev** skill → `references/shell-bash.md` — carapace's bash output formatting
