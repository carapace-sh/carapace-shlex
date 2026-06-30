# carapace-shlex v2 — Implementation Plan

## Goal

Generalize the v1 POSIX-only lexer into a v2 that supports multiple shell formats (including non-POSIX), while preserving the proven tokenizer state machine and `TokenSlice` operations that carapace depends on.

## Design Principles

1. **Preserve the state machine** — the `scanStream` state machine and `Token`/`TokenSlice` model are battle-tested in carapace. V2 keeps them and makes the *inputs* configurable.
2. **Formats are configuration, not forks** — each shell format is a small struct/classifier that declares rune classes and operator grammar. No per-shell copy of the state machine.
3. **Backward compatible** — `Split(s)` with no format argument defaults to the bash (POSIX) format, matching v1 behavior. Existing carapace code keeps working.
4. **Incremental** — ship the format abstraction with bash first, then add other formats one at a time. Each format is independently testable.
5. **Learn from sibling repos** — `carapace-jjlex` and `carapace-ffmpeg` already establish patterns we should adopt (see below).

---

## Lessons from carapace-jjlex and carapace-ffmpeg

Both sibling repos (`carapace-jjlex`, `carapace-ffmpeg`) are carapace-sh ecosystem lexers/parsers for non-shell grammars (jj revsets, ffmpeg arg streams). They share patterns directly applicable to shlex v2.

### Adopt: `Span{Start, End}` on tokens

Both repos tag every AST node with a `Span{Start, End}` (byte offsets):

```go
// carapace-jjlex/pkg/fileset/span.go
type Span struct {
    Start int
    End   int
}
```

v1 shlex has `Token.Index` (start) and computes the end as `Index + len(RawValue)`. Replacing `Index` with `Span{Start, End}` is a low-risk improvement: it makes adjacency explicit (`t.Span.End == other.Span.Start`), matches the ecosystem convention, and is clearer in the `adjoins`/`Words` code. **Note**: v1 `Index` is rune-based (the tokenizer increments per-rune), so `Span` should be rune offsets to preserve behavior — or byte offsets if we switch to `utf8.DecodeRuneInString` (see below).

### Adopt: `CompletionContext` (dual-parser pattern)

Both repos have a strict parser and a tolerant completion parser:

```go
// carapace-jjlex/pkg/fileset
func Parse(input string) (*Expression, error)              // strict
func ParseForCompletion(input string) *CompletionContext   // tolerant, cursor at end

// carapace-ffmpeg/pkg/argstream
func Parse(args []string) (*Program, error)
func ParseForCompletion(args []string, trailingSpace bool) *CompletionContext
```

The completion parser stops at the cursor and returns a structured `CompletionContext` describing what's expected. v1 shlex is already error-tolerant (handles open quotes, EOF-after-escape), but carapace extracts completion context *manually* via chained calls:

```go
// carapace internal/shell/zsh/action.go (current)
splitted, _ := shlex.Split(env.Compline())
rawValue := splitted.CurrentToken().RawValue
// regex on rawValue to detect quoting state...
splitted.CurrentPipeline().FilterRedirects().Words()...
```

A v2 `SplitForCompletion` returning a `CompletionContext` directly would be cleaner:

```go
type CompletionContext struct {
    CurrentWord   string     // the word at the cursor (dequoted)
    RawCurrentWord string    // raw source of the current word
    Prefix        string     // wordbreak prefix up to cursor
    QuotingState  LexerState // IN_WORD / QUOTING / QUOTING_ESCAPING / ESCAPING
    Pipeline      TokenSlice // current pipeline (redirects filtered)
    Words         TokenSlice // all words in current pipeline
}
```

This is **additive** — `Split` stays for backward compat. `SplitForCompletion` is the ergonomic API for completion callers. Phase 1 can include it or defer to a later phase (see phasing discussion).

### Adopt: `Profile`/`Format` config pattern (confirmed)

carapace-ffmpeg already uses exactly this pattern:

```go
// carapace-ffmpeg/pkg/argstream/profile.go
type ToolProfile struct {
    Name            string
    HasOutputSection bool
    OptionIndex     *OptionIndex
}

func ParseWithProfile(args []string, profile *ToolProfile) (*Program, error)
func ParseForCompletionWithProfile(args []string, trailingSpace bool, profile *ToolProfile) *CompletionContext

var DefaultFFmpegProfile = &ToolProfile{...}
var DefaultFFplayProfile  = &ToolProfile{...}
var DefaultFFprobeProfile = &ToolProfile{...}
```

Our `Format` + `SplitWith(s, format)` is the same pattern. This validates the approach. **Naming**: ffmpeg calls it "Profile"; we call it "Format". Either name works — "Format" is clearer for a lexer (it's a lexical format, not a tool profile).

### Do NOT adopt: recursive-descent grammar parsing

The sibling repos use recursive-descent parsers for *structured grammars* (revset operator precedence, filtergraph chains, ffmpeg option scopes). shlex's job is different: it lexes a raw command-line string into words while tracking quote state. The state machine is the right tool — it's not parsing a grammar, just tracking quote/escape/operator state. The state machine stays.

### Do NOT adopt: byte-based `peek`/`advance` (yet)

carapace-ffmpeg's `filtergraph` parser uses byte-level `peek()`/`advance()` (`byte`, not `rune`), while `carapace-jjlex`'s fileset parser uses `utf8.DecodeRuneInString` for rune-aware peeking. v1 shlex uses `bufio.Reader.ReadRune` (rune-aware). Since shell command lines can contain non-ASCII (filenames, etc.), rune-aware is correct. **Keep rune-based**; don't downgrade to bytes.

---

## Current State (v1)

The lexer is hardcoded to POSIX/bash:

- **Rune classes** are global constants (`spaceRunes`, `escapingQuoteRunes`, `escapeRunes`, etc.) baked into `newDefaultClassifier`.
- **Operator grammar** is hardcoded in `wordbreakType()` matching literal strings (`"<"`, `">>"`, `"&&"`, etc.).
- **`COMP_WORDBREAKS`** is read from the environment inside the classifier — a bash-specific concern leaking into the core.
- The state machine handles two quote types: escaping (`"..."`) and non-escaping (`'...'`), one escape char (`\`), and `#` comments to newline.

This covers bash, zsh (mostly), oil OSH, and tcsh (mostly). It cannot express: fish `\'` in single quotes, elvish `''` doubled-quote, nushell `r#'...'#`/backtick strings, PowerShell backtick escape and here-strings, xonsh Python string literals, cmd caret escape and `"`-only quotes, ion `^>`/`^|` operators, fish keyword operators (`and`/`or`).

---

## Architecture

### The Format interface

A `Format` declares everything shell-specific that the tokenizer needs:

```go
package shlex

// Format describes a shell's lexical rules.
type Format interface {
    // Classifier maps runes to rune classes.
    Classifier() tokenClassifier

    // ClassifyOperator maps a wordbreak token's RawValue to a WordbreakType.
    // Called for WORDBREAK_TOKENs to determine redirect vs pipeline vs other.
    ClassifyOperator(raw string) WordbreakType

    // KeywordOperators returns bare-word operators (e.g. fish "and"/"or")
    // that should be treated as WORDBREAK_TOKEN despite being word characters.
    // Returns nil for shells without keyword operators.
    KeywordOperators() map[string]WordbreakType
}
```

### Rune classes (unchanged)

The `runeTokenClass` enum stays as-is — it's general enough:

```go
const (
    unknownRuneClass runeTokenClass = iota
    spaceRuneClass
    escapingQuoteRuneClass
    nonEscapingQuoteRuneClass
    escapeRuneClass
    commentRuneClass
    wordbreakRuneClass
    eofRuneClass
)
```

Every shell's quoting fits into "escaping quote" / "non-escaping quote" / "escape" with one caveat: shells where single quotes have *partial* escapes (fish `\'`, elvish `''`) need a small state-machine extension (below).

### State machine extensions

The v1 state machine handles two quote modes. V2 adds:

1. **`nonEscapingQuoteEscapes` flag** — for formats where the non-escaping quote has limited escapes (fish `\'`/`\\`, elvish `''`, zsh `RC_QUOTES`, PowerShell `''`). When enabled, the `QUOTING_STATE` peeks at the next rune on seeing the quote char or escape char:

   - **elvish/zsh/PowerShell `''`**: on `'` in `QUOTING_STATE`, peek — if next is also `'`, consume both, emit one `'`, stay in state; else close.
   - **fish `\'`/`\\`**: on `\` in `QUOTING_STATE`, peek — if next is `'` or `\`, consume both, emit the escaped char, stay; else `\` is literal.

   This is a format-configurable behavior, not a new state — the `QUOTING_STATE` handler checks a format flag.

2. **Multi-rune openers** (future) — `r#'...'#` (nushell), `@"..."@` (PowerShell here-strings), `'''...'''` (oil/xonsh). These need a format-specific "on enter quote" hook or a pre-classify step. **Defer to phase 3** — the common case (completion of a partial word) works without full multi-rune string type support, since carapace's Patch phase mostly needs basic quote stripping.

### Operator grammar

`wordbreakType()` becomes a format method. The v1 hardcoded switch moves into the bash format. Formats declare their operator→type mapping. Multi-char greedy matching in `WORDBREAK_STATE` stays in the state machine (it already accumulates consecutive wordbreak runes).

Keyword operators (fish `and`/`or`/`not`) need special handling: the tokenizer emits them as `WORD_TOKEN` normally, so a post-pass or a keyword-lookup in `scanStream`'s word-termination path is needed. **Approach**: after emitting a `WORD_TOKEN`, if the format has `KeywordOperators()` and `RawValue` matches a keyword, reclassify the token as `WORDBREAK_TOKEN` with the mapped type. This keeps the state machine simple.

### COMP_WORDBREAKS

Move `COMP_WORDBREAKS` env-var reading into the bash format only. The core `tokenClassifier` no longer reads environment variables. The bash format's `Classifier()` builds the classifier with `COMP_WORDBREAKS` merged in (preserving v1 behavior for bash).

---

## API

### New

```go
// SplitWith partitions s into tokens using the given format.
func SplitWith(s string, format Format) (TokenSlice, error)

// SplitForCompletion parses s (up to cursor) and returns a structured
// completion context. Cursor defaults to len(s) (end of input).
// This is the ergonomic API for completion callers, replacing the manual
// tokens.Words().CurrentPipeline().CurrentToken() chains carapace does today.
func SplitForCompletion(s string, format Format) *CompletionContext

// CompletionContext describes the completion state at a cursor position.
// Modeled after the dual-parser pattern in carapace-jjlex/carapace-ffmpeg.
// This is the primary API for completion callers, replacing the manual
// tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken() chains.
type CompletionContext struct {
    Words          []string    // CurrentPipeline().FilterRedirects().Words().Strings()
    CurrentWord    string      // the word at the cursor (dequoted Value)
    RawCurrentWord string      // raw source of the current word (with quotes)
    Prefix         string      // wordbreak prefix up to cursor
    QuotingState   LexerState  // IN_WORD / QUOTING / QUOTING_ESCAPING / ESCAPING
    IsRedirect     bool        // true when completing a redirect target (e.g. after >)
    Pipeline       TokenSlice  // raw pipeline tokens (escape hatch for edge cases)
}

// Formats registry
func BashFormat() Format       // POSIX/bash (reads COMP_WORDBREAKS)
func ZshFormat() Format        // bash + RC_QUOTES, WORDCHARS
func FishFormat() Format       // fish quoting, keyword operators
func ElvishFormat() Format     // elvish quoting, bareword \
func NushellFormat() Format
func PowershellFormat() Format
func XonshFormat() Format
func TcshFormat() Format
func OilFormat() Format
func CmdFormat() Format
```

### Token: add Span (additive, backward compatible)

Adopt the `Span{Start, End}` pattern from carapace-jjlex/carapace-ffmpeg:

```go
type Span struct {
    Start int  // rune offset in input (matches v1 Token.Index semantics)
    End   int  // rune offset after the token's RawValue
}

type Token struct {
    Type           TokenType
    Value          string
    RawValue       string
    Span           Span         // replaces Index; Index kept as alias = Span.Start
    State          LexerState
    WordbreakType  WordbreakType `json:",omitempty"`
    WordbreakIndex int
}
```

**Backward compat**: `Token.Index` becomes `Token.Span.Start` (or a method/alias if struct field rename is too disruptive). The `adjoins` check becomes `t.Span.End == other.Span.Start`. Existing tests using `Index` need the field rename — this is the one breaking change, but it's mechanical and improves clarity.

### Unchanged (backward compatible)

```go
func Split(s string) (TokenSlice, error)   // defaults to BashFormat()
func Join(s []string) string               // unchanged (POSIX join)
type TokenSlice []Token                    // unchanged operations
type WordbreakType int                     // unchanged
```

`TokenSlice` and `WordbreakType` remain. The `TokenSlice` helper methods (`CurrentPipeline`, `FilterRedirects`, `Words`, `WordbreakPrefix`, `CurrentToken`, `Strings`, `Pipelines`) remain exported — the CLI and carapace both use them directly. `Split` stays public for low-level token access (nushell `Patch()` uses `tokens[0].Value` directly, no helpers needed).

---

## Phased Delivery

### Phase 1 — Format abstraction + bash + Span + CompletionContext (preserve v1 behavior)

**Goal**: refactor the core to be format-driven without changing any behavior. Bash format reproduces v1 exactly. Add `Span` and `CompletionContext` as additive improvements.

1. Extract `Format` interface and `tokenClassifier` construction into a format-configurable path.
2. Move `wordbreakType()` hardcoded switch into a `BashFormat.ClassifyOperator()`.
3. Move `COMP_WORDBREAKS` env reading into `BashFormat.Classifier()`.
4. Add `SplitWith(s, format)`.
5. Make `Split(s)` delegate to `SplitWith(s, BashFormat())`.
6. Add `nonEscapingQuoteEscapes` format flag (off for bash).
7. Add `Span{Start, End}` to `Token` (replaces `Index`; update `adjoins` and existing tests).
8. Add `SplitForCompletion(s, format)` returning `CompletionContext`.
9. **All existing tests pass** (with mechanical `Index`→`Span.Start` updates) — this is the acceptance criterion.

**Files**: `shlex.go`, `tokenslice.go`, `wordbreak.go`, new `format.go`, new `format_bash.go`, new `completion.go`.

### Phase 2 — POSIX-family formats

**Goal**: add the formats that are close to bash.

1. **Zsh** — bash format + `RC_QUOTES` flag (`''`→`'` in single quotes), `WORDCHARS`/`FIGNORE` wordbreak adjustment. No new states.
2. **Oil (OSH)** — bash format (alias). YSH string types deferred.
3. **Tcsh** — bash format + `backslash_quote` option flag. Backtick/command-subst is already a word char.

These reuse the state machine with at most the `nonEscapingQuoteEscapes` flag flipped. Each gets a `format_<shell>.go` and a `format_<shell>_test.go`.

### Phase 3 — Non-POSIX formats with quote extensions

**Goal**: formats needing the `nonEscapingQuoteEscapes` flag or keyword operators.

1. **Fish** — `nonEscapingQuoteEscapes` with `\'`/`\\`; keyword operators (`and`/`or`/`not`); narrower escape set in `"..."`.
2. **Elvish** — `nonEscapingQuoteEscapes` with `''`; `\` is *not* an escape outside quotes (bareword char); no POSIX list operators.
3. **PowerShell** — backtick as escape char (not `\`); `''` and `""` doubled-quote; `#` comments. Here-strings and `--%` deferred.

### Phase 4 — Complex string types (optional / as needed)

**Goal**: multi-rune string openers that don't fit the single-rune state machine.

1. **Nushell** — `r#'...'#` raw strings, backtick strings, `$'...'`/`$"..."`. Needs a pre-classify or multi-rune-opener hook.
2. **PowerShell here-strings** — `@'...'@` / `@"..."@`, multi-line, line-start closer.
3. **Xonsh** — Python string prefixes (`r`/`f`/`p`/`b`), triple-quotes.
4. **Oil YSH** — `r'...'`, `'''...'''`.

**Approach**: add an optional `OnWordStart(runes []rune) (skip int, class runeTokenClass)` hook to `Format` that lets a format recognize a multi-rune opener and return how many runes to consume + which quote class to enter. This keeps the common case (single-rune quotes) fast.

**Note**: Phase 4 may not be needed for carapace's use case — carapace mostly needs quote stripping for the Patch phase, and the common single/double quote case covers the vast majority of completion input. Phase 4 is for completeness.

### Phase 5 — Cmd.exe format

**Goal**: the most divergent format.

1. **Cmd** — caret `^` escape; `"`-only quotes (no single quote); `&`/`&&`/`||` as command separators; `REM`/`::` comments; `%` as word char.

Cmd needs the most format-specific logic: `REM` keyword-comment detection, `::` two-rune comment, `&` as separator (not background). This is last because it's the furthest from POSIX and has the least carapace usage.

---

## Migration & Compatibility

- **v1 `Split(s)` signature unchanged** — defaults to bash format, same behavior.
- **`Token`, `TokenSlice`, `WordbreakType` unchanged** — carapace code using these needs zero changes.
- **Carapace integration** — carapace's `bash.Patch()`, `nushell.Patch()`, `cmd_clink.Patch()`, and `zsh.ActionRawValues()` call `shlex.Split()` today. They can opt into `SplitWith(s, ZshFormat())` etc. incrementally — no forced migration.
- **Module path** — stays `github.com/carapace-sh/carapace-shlex`. V2 is a superset.

## Testing Strategy

- **Phase 1**: all existing tests pass unchanged (behavior preservation).
- **Each format**: a `format_<shell>_test.go` with table-driven tests covering:
  - basic word splitting
  - each quote type (open and closed)
  - escape at EOF
  - open quote at EOF (cursor inside quotes)
  - operator splitting (all operators for that format)
  - comment handling
  - edge cases from the `format-*.md` skill references
- **Cross-format**: a shared test suite that runs the same "universal" cases (plain words, spaces, basic quotes) against every format to ensure the common path works.
- **Carapace regression**: run carapace's test suite against the updated shlex to verify no behavior change in the bash path.

## Risks & Open Questions

1. **`nonEscapingQuoteEscapes` peek-ahead** — adds a `UnreadRune`/peek in `QUOTING_STATE`. Need to verify this doesn't break the `Span`/`RawValue` accounting. The v1 code already uses `UnreadRune` in other states, so the mechanism exists.

2. **Keyword operators (fish)** — reclassifying a `WORD_TOKEN` to `WORDBREAK_TOKEN` post-emission is slightly awkward. Alternative: check `KeywordOperators()` in `IN_WORD_STATE` at word termination. Need to decide where the check lives.

3. **`COMP_WORDBREAKS` env timing** — v1 reads it at classifier construction. The bash format must read it at `Classifier()` call time (same). Verify no test sets `COMP_WORDBREAKS` mid-run expecting a change.

4. **Multi-rune openers (phase 4)** — the `OnWordStart` hook adds complexity. Is it needed for carapace's actual usage, or is basic quote stripping enough? **Recommend**: defer until a concrete need arises.

5. **Join()** — v1 `Join` is POSIX-specific. Should v2 add `JoinWith(s, format)`? Probably not needed — carapace uses `Join` for bash only. Leave as-is unless a need arises.

6. **`Span` rename of `Index`** — replacing `Token.Index` with `Token.Span.Start` is the one breaking change in Phase 1. It's mechanical (search/replace in tests) but touches the public API. **Alternative**: keep `Index` as a field and add `Span` alongside it, deprecating `Index` later. **Recommend**: do the clean rename in v2 since we're already breaking the internal `wordbreakType` API; carapace's usage of `Token.Index` is limited.

7. **`CompletionContext` scope** — `SplitForCompletion` is the primary API for completion callers. The `TokenSlice` helper methods (`CurrentPipeline`, `FilterRedirects`, `Words`, `WordbreakPrefix`, `CurrentToken`) remain exported for the CLI and low-level use. `Split` stays public for low-level use (nushell `Patch()`, raw token access). Carapace's `action.go`, `bash/patch.go`, `cmd_clink/patch.go`, and `zsh/action.go` can migrate to `SplitForCompletion` — the zsh regex hack (4 regexes on `RawValue`) is replaced by `ctx.QuotingState`. `IsRedirect` is a field in `CompletionContext` (not caller logic), with `Pipeline TokenSlice` as the escape hatch for edge cases.

---

## Deliverables Checklist

- [x] Phase 1: `Format` interface, `BashFormat`, `SplitWith`, `Split` delegates, `Span`, `SplitForCompletion`/`CompletionContext`, all tests pass
- [x] Phase 2: `ZshFormat`, `OilFormat`, `TcshFormat` + tests
- [x] Phase 3: `FishFormat`, `ElvishFormat`, `PowershellFormat` + tests
- [x] Phase 4: `NushellFormat`, `XonshFormat` + tests (raw strings `r#'...'#` and triple-quotes deferred)
- [x] Phase 5: `CmdFormat` + tests (REM/:: keyword comments deferred)
- [x] Carapace regression: carapace builds and all tests pass with 2-line `Index`→`Span.Start` change in `action.go`
- [x] Carapace migration: all call sites use SplitForCompletion (zsh regex hack eliminated)
- [x] Join rewrite: format-aware JoinWith with per-shell QuoteWord

---

## Deferred Format Features

These require multi-rune opener support or keyword-based comment detection
not yet implemented. The basic quote types (single, double, backtick) cover
the vast majority of completion input. Implement when a concrete need arises.

| Feature | Shell | What's needed | Impact |
|---------|-------|---------------|--------|
| Raw strings `r#'...'#` | nushell | Multi-rune opener: `r#` prefix before `'`, matching `#` count in closer. `#` in opener must not trigger comment state. | Low — nushell completers mostly use basic quotes |
| Triple-quotes `'''...'''` / `"""..."""` | xonsh, oil (YSH) | 3-rune lookahead when entering quote state: if next 2 runes match, consume all 3 as opener. | Low — triple-quotes are rare in completion input |
| `REM` / `::` keyword comments | cmd | `REM` is a word that starts a comment at command position (keyword detection, not rune-based). `::` is a 2-rune comment opener. | Low — comments don't affect completion in practice |
| Here-strings `@'...'@` / `@"..."@` | powershell | Multi-line string with line-start closer. Needs format-specific scan routine. | Low — here-strings are rare in completion input |
| `--%` stop-parsing | powershell | Special word that switches remainder of line to raw lexing mode. | Low — rarely appears in completion input |
| `WORDCHARS` env var | zsh | Characters that are NOT word breaks (inverse of COMP_WORDBREAKS). Read env var in ZshFormat.Classifier(). | Medium — affects word boundary detection |
