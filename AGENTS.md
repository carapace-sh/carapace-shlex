# AGENTS.md

Guide for agents working in the `carapace-shlex` repository — a multi-shell command-line lexer (fork of go-shlex) that splits and re-joins command lines while tracking quotation state for shell completion.

## Commands

```bash
# Build everything (library + CLI)
go build -v ./...

# Run tests with coverage (matches CI)
go test -v -coverprofile=profile.cov ./...

# Formatting check enforced by CI — note the -s (simplify) flag
gofmt -d -s .

# Static analysis enforced by CI
go install honnef.co/go/tools/cmd/staticcheck@latest && staticcheck ./...

# Run the CLI directly (no build step needed)
go run ./cmd/carapace-shlex --format bash --completion-context "echo foo | grep hel"
go run ./cmd/carapace-shlex --format elvish --current-pipeline --words "bat | {|"

# Test a single format
go test -run TestElvish -v ./...
```

Go 1.24.0. The CI image is `ghcr.io/carapace-sh/go:1.25.4`. Tags trigger GoReleaser builds (see `.goreleaser.yml`).

## Repository Layout

- **Root package `shlex`** — the library: tokenizer state machine (`shlex.go`), `Format` interface (`format.go`), per-shell formats (`format_<shell>.go`), token slice operations (`tokenslice.go`), wordbreak types (`wordbreak.go`), quoting helpers (`quote.go`), completion context (`completion.go`).
- **`cmd/carapace-shlex/`** — a **separate Go module** (`cmd/go.mod`) that imports the library and wraps it as a cobra CLI. It depends on `carapace` and `carapace-bridge` for its own completion.
- **`go.work`** — workspace including both the root module and `./cmd`. Contains a `replace` directive: `github.com/carapace-sh/carapace v1.11.0 => ../carapace`. This means local development expects a sibling `../carapace` checkout. The `cmd/go.mod` has its own `replace ... => ../` for the shlex library itself.
- **`skills/shlex/`** — in-depth reference docs (architecture, cross-shell comparison, per-format references). Load these via the `shlex` skill when doing substantial format work.
- **`plan.md`** — design doc for elvish lambda-pipe handling; useful context for the `PostProcessor` interface and `WORDBREAK_LAMBDA_PIPE`.

## Architecture

### Data flow

```
command line string
  → SplitWith(s, format) or SplitForCompletion(s, format)
    → format.Classifier()           (rune → rune class)
      → tokenizer.scanStream()       (flat state machine, shared across all formats)
        → TokenSlice                 (typed tokens with Span + quotation State)
          → [optional] format.PostProcess(tokens)   (post-pass reclassification)
            → CompletionContext       (current word, prefix, quoting state, pipeline)
```

### The tokenizer is a flat state machine

The core state machine in `shlex.go` (`scanStream`) has **no nesting awareness**. It classifies runes one at a time into `TokenType`s (WORD/SPACE/COMMENT/WORDBREAK) and tracks quotation state via `LexerState`. Every shell format plugs into the **same** machine via the `Format` interface — there is no per-shell parser.

**Consequence**: format-specific behavior that requires context the flat machine can't track (e.g. elvish `|` inside `{|params|}` being a parameter delimiter, not a pipeline pipe) is handled via the optional `PostProcessor` interface, which runs a post-pass over the `TokenSlice` after tokenization. Do **not** add nesting/brace tracking into `scanStream` — add a `PostProcess` method on the format instead.

### The `Format` interface and its optional companions

`format.go` defines:
- **`Format`** (required): `Classifier`, `ClassifyOperator`, `KeywordOperators`, quote-behavior flags (`NonEscapingQuoteEscapes`, `NonEscapingQuoteBackslashEscapes`, `EscapeNotBareword`, `EscapeNotInEscapingQuote`, `EscapingQuoteEscapeChars`), `TripleQuoteSupport`, `RawPrefixSupport`, `QuoteWord`.
- **Optional interfaces** (asserted via type assertion in `tokenizer.Next` / `SplitWith`):
  - `PostProcessor` — post-pass token reclassification (elvish, nushell)
  - `BlockCommenter` — multi-line block comments (PowerShell `<# #>`)
  - `StopParsingToken` — raw lexing mode after a token (PowerShell `--%`)
  - `LineContinuationEscaper` — escape+newline as line continuation (PowerShell backtick)
  - `EscapingQuoteUnescaper` — custom unescape inside double quotes beyond simple backslash-dropping

When adding a new format, implement `Format` plus whichever optional interfaces apply. No-op returns (e.g. `KeywordOperators() nil`) are the norm for formats that don't need a feature.

### Token model

```go
type Token struct {
    Type           TokenType     // WORD_TOKEN, WORDBREAK_TOKEN, etc.
    Value          string        // dequoted value
    RawValue       string        // raw source text including quotes/escapes
    Span           Span          // rune offsets {Start, End} — NOT byte offsets
    State          LexerState    // quotation state after this token
    WordbreakType  WordbreakType // operator type for WORDBREAK_TOKENs
    WordbreakIndex int           // index of last opening quote in Value (prefix boundary)
}
```

`Span` offsets are **rune offsets**, not byte offsets — relevant when inspecting multi-byte input. `TokenSlice.Words()` merges tokens by `Span` adjacency (End==Start), so quote-openers, wordbreaks, and word fragments that touch get merged into one word.

### WordbreakType drives TokenSlice operations

`WordbreakType.IsPipelineDelimiter()` and `IsRedirect()` determine how `Pipelines()`, `CurrentPipeline()`, `FilterRedirects()`, and `WordbreakPrefix()` behave. When adding a new operator type, decide deliberately whether it should split pipelines or be filtered as a redirect — `WORDBREAK_LAMBDA_PIPE` intentionally returns false for both so elvish lambda parameter lists don't break pipeline splitting.

`FilterRedirects()` has a special case: a numeric token (e.g. `2`) immediately adjoining a redirect operator (e.g. `>`) is filtered out as the fd prefix. Don't break this when touching redirect logic.

### Public API surface

- `Split(s)` / `SplitWith(s, format)` → `TokenSlice, error`
- `SplitForCompletion(s, format)` → `*CompletionContext` (never errors; returns empty context with `START_STATE` on failure)
- `Join(s)` / `JoinWith(s, format)` → quoted string
- `CompletionContext` — the completion-oriented API: `Words`, `CurrentWord`, `RawCurrentWord`, `Prefix`, `QuotingState`, `IsRedirect`, `InLambdaParams`, and `Pipeline` (raw token escape hatch)

`SplitForCompletion` is the primary entry point for completion callers (carapace). It internally calls `SplitWith` then derives the context fields. `InLambdaParams` is detected via an odd count of `WORDBREAK_LAMBDA_PIPE` in the current pipeline (toggle heuristic — see limitations in `plan.md`).

## Adding a New Shell Format

1. Create `format_<shell>.go` implementing `Format` (+ optional interfaces as needed).
2. Add a `<shell>QuoteWord` function in `quote.go` and reference it from the format's `QuoteWord` method. Quoting helpers are kept in `quote.go`, not in the format file.
3. If the shell has operators that differ from the bash grammar, add a `<shell>WordbreakType` function in `wordbreak.go` (see `bashWordbreakType`, `tcshWordbreakType` as templates).
4. Add the format to `formatFromFlag` in `cmd/carapace-shlex/cmd/root.go` and to the `--format` flag's completion values.
5. Add the format to the table in `README.md`.
6. Create `format_<shell>_test.go` (see existing test files for the pattern).
7. If the shell needs behavior the flat state machine can't express, implement `PostProcessor` rather than modifying `scanStream`.

## Testing Patterns

Tests live alongside their format: `format_<shell>_test.go`. The established pattern:

```go
tokens, err := SplitWith(input, SomeFormat())
if err != nil { t.Fatal(err) }
words := tokens.Words().Strings()
// assert on words, and on token State / WordbreakType for quoting/operator cases
```

Tests assert on **dequoted `Value`** via `Words().Strings()`, and on `State` (e.g. `IN_WORD_STATE`, `QUOTING_STATE`) and `WordbreakType` for quotation/operator behavior. The `Equal` method on `Token` compares all fields — useful for golden-style tests.

`completion_test.go` tests `SplitForCompletion` and the `CompletionContext` fields. `join_test.go` tests `JoinWith` roundtrips. `shlex_test.go` tests the core tokenizer/state machine.

## Gotchas

- **`go.work` replace expects `../carapace`** as a sibling checkout. The root `go.mod` also has the replace, so even non-workspace `go build` pulls the local carapace. CI uses the image `ghcr.io/carapace-sh/go` which has the dependency available.
- **`cmd/carapace-shlex` is a separate module** with its own `go.mod`. Changes to the library's public API must be reflected in `cmd/go.mod`'s `require` (often via a replace to `../`).
- **gofmt `-s` (simplify) is enforced**, not just plain gofmt. Run `gofmt -d -s .` before committing.
- **`staticcheck` is enforced** in CI. Install and run it locally — it's not in the standard toolchain.
- **`bufio.Reader` only supports one `UnreadRune`**. The triple-quote peek helpers (`checkTripleQuote`, `checkTripleClose`) handle this constraint by returning a `consumedRune` when the second peek fails to match — the first peeked rune can't be unread, so callers must add it to `RawValue`. Preserve this pattern when extending peek-based logic.
- **`BashFormat().Classifier()` reads `COMP_WORDBREAKS`** from the environment at call time, not at init. Tests that assert on bash wordbreaks should set/unset `COMP_WORDBREAKS` explicitly or they inherit the ambient value.
- **`plan.md` is a design doc, not implemented spec**. The elvish lambda-pipe `PostProcess` is implemented, but `InLambdaParams` uses a toggle heuristic that doesn't handle nested lambdas — documented as a known limitation in `plan.md`.
- **Don't add comments to code** unless explaining *why* (and only when non-obvious). The codebase uses minimal comments; match that style.
