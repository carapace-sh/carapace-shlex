# Elvish Brace/Lambda Context — Implementation Plan

## Problem

The tokenizer is a flat state machine with no nesting awareness. It classifies every `|` as `WORDBREAK_PIPE` regardless of brace context. In elvish, `|` inside `{...}` is a lambda parameter delimiter, not a pipeline operator.

Example: `bat | {|` currently produces:

```
WORD(bat)  WORDBREAK_PIPE(|)  WORD({)  WORDBREAK_PIPE(|)  WORD("")
```

The second `|` is wrong — it's a lambda parameter delimiter. `CurrentPipeline()` splits at the wrong point, yielding an empty pipeline. The cursor is seen as "start of a new command" instead of "inside a lambda parameter list."

## Elvish Grammar (from source analysis)

From `pkg/parse/parse.go` in the elvish source:

### Lambda disambiguation (`lbrace()`, line 814)

After consuming `{`, peek at the next rune:
- `|`, whitespace, `;`, `\r`, `\n` → **lambda**
- anything else → **braced list** (e.g. `{a,b}`)

### Lambda syntax (`lambda()`, line 783)

```
Lambda = '{' [ '|' { (Compound | MapPair) { Space } } '|' ] Chunk '}'
```

- First `|` opens the parameter list (optional — `{ body }` has no params)
- Between the `|...|`, parameters are `Compound` expressions or `&name=default` MapPairs
- Second `|` closes the parameter list
- Then a `Chunk` (full command body) follows
- `}` closes the lambda

### How elvish disambiguates `|`

Elvish's parser is recursive-descent with no separate lexer. The **call stack** is the context: when inside `lambda()`, `|` means parameter delimiter; when inside `Pipeline.parse()`, `|` means pipe. There is no ambiguity because they're at different grammar levels.

### Elvish's own completion limitation

Elvish's `completeCommand` completer (`pkg/edit/complete/completers.go`, line 68-73) has a TODO acknowledging it incorrectly triggers after `{|` — it offers command completions in parameter position. There is no dedicated lambda parameter completer.

## Design Constraints

1. **The tokenizer is a flat state machine** — it doesn't track nesting. Adding full recursive-descent parsing would be a fundamental architecture change, not justified by this one case.
2. **The `Format` interface is the extension point** — format-specific behavior should be expressed there, not hardcoded in the core state machine.
3. **Other shells don't have this problem** — bash/zsh/fish/etc. don't use `|` inside braces as a delimiter. This is elvish-specific.
4. **Completion is the primary use case** — the lexer exists to support shell completion. The key question is: what does the completion caller need when the cursor is inside `{|...|`?
5. **Backward compatibility** — existing tests and other formats must not be affected.

## Approach: Post-Pass Brace Context Tracking

Rather than adding nesting state to the tokenizer's core state machine, add a **post-pass** that reclassifies `WORDBREAK_PIPE` tokens when they appear inside brace context. This keeps the state machine untouched and isolates elvish-specific logic.

### Why post-pass, not state machine change

- The tokenizer's `WORDBREAK_STATE` greedily accumulates consecutive wordbreak runes (`|><;`). Adding brace tracking here would complicate every format's path.
- A post-pass is format-specific: only elvish needs it, and only elvish's `ClassifyOperator` or a new `Format` method would invoke it.
- The post-pass operates on the `TokenSlice` — it's already the right abstraction level for reclassification.

### Implementation

#### Step 1: New `WordbreakType` for lambda parameter delimiter

In `wordbreak.go`, add:

```go
WORDBREAK_LAMBDA_PIPE  // | inside {|params| ...} — not a pipeline delimiter
```

This type should **not** be a pipeline delimiter (`IsPipelineDelimiter()` returns false) and **not** be a redirect (`IsRedirect()` returns false). It's a parameter delimiter — `CurrentPipeline()` will not split on it.

#### Step 2: Brace context tracking post-pass

A new function in `format_elvish.go` (or `tokenslice.go` if we make it a `TokenSlice` method):

```go
// resolveLambdaPipes reclassifies WORDBREAK_PIPE tokens that are inside
// brace/lambda context as WORDBREAK_LAMBDA_PIPE.
//
// The rule (from elvish's lbrace() disambiguation):
// - When '{' is followed by '|', whitespace, ';', or newline, it's a lambda.
// - The first '|' after '{' in lambda context opens the parameter list.
// - The second '|' closes it.
// - Any '|' between the opening and closing '|' is a parameter delimiter,
//   not a pipeline pipe.
// - After the closing '|', the lambda body is a Chunk which may contain
//   real pipeline pipes — but those are inside the braces too, so we need
//   to track brace depth.
//
// For the flat tokenizer, we use a heuristic:
// - Track brace depth: '{' increments, '}' decrements.
// - When inside braces (depth > 0), '|' is a lambda parameter delimiter,
//   not a pipeline pipe.
// - This is an approximation: it can't distinguish lambda body pipes from
//   parameter pipes. But for completion purposes, the key case is:
//   cursor inside {|...| (parameter position), where we must NOT split
//   the pipeline.
//
// Limitation: a pipe in a lambda body like {|a| echo $a | grep foo} would
// also be reclassified. This is acceptable — the lambda body's pipeline
// structure is less important for completion than the parameter position.
```

Actually, let me reconsider. The heuristic "all `|` inside braces are lambda pipes" is too coarse — it would break `{|a| cmd1 | cmd2}` where the second `|` is a real pipeline inside the lambda body. Let me refine.

#### Refined approach: Track the parameter-list state machine

The post-pass can track a simple state machine for each brace scope:

```
BRACE_OUTSIDE     — not inside braces
BRACE_LAMBDA_OPEN — saw '{' followed by |/whitespace/newline (it's a lambda)
BRACE_PARAMS      — inside {|...| parameter list (between first | and second |)
BRACE_LAMBDA_BODY — after the closing | of params, inside the lambda body
BRACE_BRACED      — inside a braced list ({a,b})
```

Transitions:
- `{` when BRACE_OUTSIDE → peek next non-space token:
  - if `|` or the `{` is followed by space/`;`/newline → BRACE_LAMBDA_OPEN
  - else → BRACE_BRACED
- `|` when BRACE_LAMBDA_OPEN → BRACE_PARAMS (opening pipe)
- `|` when BRACE_PARAMS → BRACE_LAMBDA_BODY (closing pipe)
- `|` when BRACE_PARAMS → stays BRACE_PARAMS (shouldn't happen — params are words between the pipes)
- `|` when BRACE_LAMBDA_BODY → stays BRACE_LAMBDA_BODY, but this `|` is a **real pipeline pipe** (not reclassified)
- `}` when any BRACE_* → BRACE_OUTSIDE (or pop to previous brace scope if nested)
- `{` when any BRACE_* → push new brace scope (nesting)

For nesting, use a stack of brace states.

The key reclassification: `|` in BRACE_PARAMS (or BRACE_LAMBDA_OPEN transitioning to BRACE_PARAMS) → `WORDBREAK_LAMBDA_PIPE`. `|` in BRACE_LAMBDA_BODY stays `WORDBREAK_PIPE`.

Wait — but the tokenizer already emitted `{` as a `WORD_TOKEN` (it's not a wordbreak). So in the TokenSlice, `{` and `}` appear as words with empty or `{`/`}` values. The post-pass can scan the token slice and use these word tokens as brace markers.

Actually, looking at the current output for `bat | {|`:

```json
{"Type": "WORD_TOKEN", "Value": "bat", ...}
{"Type": "WORDBREAK_TOKEN", "Value": "|", "WordbreakType": "WORDBREAK_PIPE", ...}
{"Type": "WORD_TOKEN", "Value": "{", ...}
{"Type": "WORDBREAK_TOKEN", "Value": "|", "WordbreakType": "WORDBREAK_PIPE", ...}
{"Type": "WORD_TOKEN", "Value": "", ...}
```

The `{` is a `WORD_TOKEN` with `Value == "{"`. The `}` would also be a `WORD_TOKEN` with `Value == "}"`. The post-pass can scan for these.

But there's a subtlety: `{` could be part of a larger word, like `{a,b}` which might be a single `WORD_TOKEN` with `Value == "{a,b}"`. In that case the `{` is not a standalone token. However, for the completion use case, we mostly care about `{|` where `{` is a standalone token (it's followed by `|` which is a wordbreak, so `{` gets its own token).

Hmm, but `{` followed by a space like `{ body }` — the space after `{` terminates the word, so `{` is a standalone `WORD_TOKEN`. And `{` followed by `|` — the `|` is a wordbreak, so `{` is a standalone `WORD_TOKEN`. So for lambda detection, `{` will always be a standalone token. Good.

For `}` — it could be part of a word or standalone. In `{|a| echo $a }`, the space before `}` terminates the previous word, making `}` standalone. In `{|a| echo $a}` (no space), `}` follows a word — but `}` is not a wordbreak, so it would be part of the word `a}`. This is a complication. However, for the completion use case (cursor at end of input), if `}` is present, the lambda is closed and we're outside it. The main case we need to handle is when the cursor is inside an **unclosed** lambda.

#### Simplification: focus on the completion use case

For `SplitForCompletion`, the cursor is at the end of the input. The primary case is:

```
bat | {|
bat | {|a
bat | {|a b
bat | {|a b|
bat | {|a b| echo
```

In all these cases, the lambda is unclosed (no `}`). The post-pass only needs to:

1. Detect that we're inside a lambda parameter list
2. Reclassify `|` tokens inside it as `WORDBREAK_LAMBDA_PIPE` instead of `WORDBREAK_PIPE`

For the general `Split` case (not completion), we should still handle it correctly, but the bar is lower — `Split` is used for parsing, not completion context.

#### Step 3: TokenSlice post-pass method

Add a method to `TokenSlice` or as a format-specific function:

```go
// resolveBraceContext reclassifies WORDBREAK_PIPE tokens inside lambda
// parameter lists as WORDBREAK_LAMBDA_PIPE.
// This is an elvish-specific post-pass.
func resolveBraceContext(tokens TokenSlice) TokenSlice {
    // Scan tokens, tracking brace context.
    // When inside {|...| (parameter list), reclassify | as WORDBREAK_LAMBDA_PIPE.
    ...
}
```

This would be called in `SplitWith` when the format is elvish, or via a new `Format` interface method like `PostProcess(tokens TokenSlice) TokenSlice`.

#### Step 4: Format interface extension

Add an optional method to the `Format` interface:

```go
type Format interface {
    // ... existing methods ...

    // PostProcess reclassifies tokens after the main tokenization pass.
    // Used by formats that need context not available in the flat state machine
    // (e.g. elvish brace/lambda context for | disambiguation).
    // Returns the input slice unchanged if not needed.
    PostProcess(tokens TokenSlice) TokenSlice
}
```

Or, to avoid breaking the interface (all existing formats would need a no-op implementation), use an optional interface:

```go
type PostProcessor interface {
    PostProcess(tokens TokenSlice) TokenSlice
}
```

And in `SplitWith`:

```go
func SplitWith(s string, format Format) (TokenSlice, error) {
    // ... existing tokenization ...
    if pp, ok := format.(PostProcessor); ok {
        tokens = pp.PostProcess(tokens)
    }
    return tokens, nil
}
```

The optional interface approach is cleaner — no changes to existing formats.

#### Step 5: ElvishFormat implements PostProcessor

```go
func (elvishFormat) PostProcess(tokens TokenSlice) TokenSlice {
    return resolveBraceContext(tokens)
}
```

#### Step 6: resolveBraceContext implementation

```go
type braceScope struct {
    state  braceState
}

type braceState int
const (
    braceOutside     braceState = iota  // not inside braces
    braceLambdaOpen                     // saw '{', next is |/space/newline → lambda
    braceParams                         // inside {|...| parameter list
    braceLambdaBody                     // after closing |, in lambda body
    braceBraced                         // inside braced list {a,b}
)

func resolveBraceContext(tokens TokenSlice) TokenSlice {
    var stack []braceScope

    for i := range tokens {
        t := &tokens[i]

        if t.Type == WORD_TOKEN && t.Value == "{" {
            // Determine if this is a lambda or braced list.
            // Peek at the next token: if it's a WORDBREAK "|" or a space-ended word,
            // it's a lambda.
            isLambda := false
            if i+1 < len(tokens) {
                next := tokens[i+1]
                if next.Type == WORDBREAK_TOKEN && next.Value == "|" {
                    isLambda = true
                }
                // Also: '{' followed by space then anything → lambda.
                // But space tokens are not in the TokenSlice (lexer skips them).
                // The '{' token's span end vs next token's span start tells us
                // if there's a gap (space) between them.
                if !t.adjoins(next) {
                    isLambda = true
                }
            }
            if isLambda {
                stack = append(stack, braceScope{state: braceLambdaOpen})
            } else {
                stack = append(stack, braceScope{state: braceBraced})
            }
            continue
        }

        if t.Type == WORD_TOKEN && t.Value == "}" {
            if len(stack) > 0 {
                stack = stack[:len(stack)-1]
            }
            continue
        }

        if t.Type == WORDBREAK_TOKEN && t.Value == "|" {
            if len(stack) == 0 {
                continue // top-level pipe, leave as WORDBREAK_PIPE
            }

            scope := &stack[len(stack)-1]
            switch scope.state {
            case braceLambdaOpen:
                // First | after {| → opens parameter list
                scope.state = braceParams
                t.WordbreakType = WORDBREAK_LAMBDA_PIPE
            case braceParams:
                // Second | → closes parameter list, enters body
                scope.state = braceLambdaBody
                t.WordbreakType = WORDBREAK_LAMBDA_PIPE
            case braceLambdaBody:
                // Pipe inside lambda body → real pipeline pipe
                // Leave as WORDBREAK_PIPE
            case braceBraced:
                // | inside braced list — unusual but treat as non-pipe
                t.WordbreakType = WORDBREAK_LAMBDA_PIPE
            }
        }
    }

    return tokens
}
```

Wait, there's an issue with the `braceParams` case. Between the opening `|` and closing `|`, there shouldn't be any `|` tokens — the parameters are words. The closing `|` is the next `|` token we see. So the transition `braceParams → braceLambdaBody` on `|` is correct.

But what about `{|a| echo $a | grep foo}`? Let's trace:

1. `{` → push braceLambdaOpen
2. `|` (first) → braceLambdaOpen → braceParams, reclassify as LAMBDA_PIPE
3. `a` → word, no change
4. `|` (second) → braceParams → braceLambdaBody, reclassify as LAMBDA_PIPE
5. `echo` → word
6. `|` (third) → braceLambdaBody, leave as WORDBREAK_PIPE ✓
7. `grep` → word
8. `}` → pop

The third `|` (inside the lambda body) correctly stays as `WORDBREAK_PIPE`. The first two `|` (parameter delimiters) are reclassified. This is correct!

And for `bat | {|`:

1. `bat` → word, stack empty
2. `|` (first, top-level) → stack empty, leave as WORDBREAK_PIPE ✓
3. `{` → push braceLambdaOpen
4. `|` (second, after `{`) → braceLambdaOpen → braceParams, reclassify as LAMBDA_PIPE ✓
5. `` (empty word at cursor) → word

After post-pass, the pipeline splitting sees:

```
WORD(bat)  PIPE(|)  WORD({)  LAMBDA_PIPE(|)  WORD("")
```

`CurrentPipeline()` splits on `WORDBREAK_PIPE` but not `WORDBREAK_LAMBDA_PIPE`, so the current pipeline is:

```
WORD({)  LAMBDA_PIPE(|)  WORD("")
```

Hmm, this is still not ideal. The `{` and LAMBDA_PIPE are in the current pipeline, which means the completion caller sees `["{", ""]` as the words. That's better than splitting at the wrong `|`, but it still includes the `{` token.

Actually, looking at how `CurrentPipeline` works (line 17-31 of tokenslice.go): it splits on `IsPipelineDelimiter()`. If `WORDBREAK_LAMBDA_PIPE` is not a pipeline delimiter, the current pipeline would be everything after the last real pipeline delimiter. So:

```
Full slice: WORD(bat)  PIPE(|)  WORD({)  LAMBDA_PIPE(|)  WORD("")
```

`Pipelines()` splits at index 1 (the real PIPE), giving:
- Pipeline 1: `[WORD(bat)]`
- Pipeline 2: `[WORD({), LAMBDA_PIPE(|), WORD("")]`

`CurrentPipeline()` returns pipeline 2: `[WORD({), LAMBDA_PIPE(|), WORD("")]`.

`Words()` merges adjoining tokens. `{` (span 6-7) and LAMBDA_PIPE `|` (span 7-8) adjoin, so they'd merge into `{|`. Then the empty word (span 8-8) adjoins the pipe, so it merges too. Result: `["{|"]`.

Hmm, that's not right either. The completion caller would see `["{|"]` as the words, with `{|` as the current word. That's not useful for completion.

Actually, let me re-examine. The `Words()` method (line 38-53 of tokenslice.go) merges tokens that adjoin. But WORDBREAK_TOKEN and WORD_TOKEN are both included — it merges based on adjacency, not type. So:

- `{` (WORD, span 6-7) and `|` (WORDBREAK, span 7-8) → adjoin → merge to `{|`
- `{|` and `` (WORD, span 8-8) → adjoin (8 == 8) → merge to `{|`

Result: one word `{|` with state START_STATE.

For completion, the caller needs to know:
1. We're inside a lambda parameter list (not at a command position)
2. The current word being typed (could be a parameter name)

The merged word `{|` doesn't convey this well. But the `Pipeline` escape hatch in `CompletionContext` gives access to the raw tokens, so a caller that needs to detect lambda context can inspect the token types.

Actually, wait. Let me reconsider whether `WORDBREAK_LAMBDA_PIPE` should be filtered or handled differently in `Words()`. Currently `Words()` merges all adjoining tokens. The `{` word and the `|` wordbreak adjoin, so they merge. If we want the completion caller to see `[""]` (the empty word as the current word being completed), we'd need to either:

- Not merge across `WORDBREAK_LAMBDA_PIPE` tokens, or
- Handle the lambda context in `CompletionContext` / `SplitForCompletion`

Option 2 is better — `SplitForCompletion` can detect the lambda context from the raw pipeline tokens and set appropriate fields in `CompletionContext`.

#### Step 7: CompletionContext extension

Add fields to `CompletionContext`:

```go
type CompletionContext struct {
    // ... existing fields ...

    // InLambdaParams is true when the cursor is inside a lambda parameter
    // list (e.g. after "{|" in elvish). When true, the completion caller
    // should complete parameter names, not commands or file arguments.
    InLambdaParams bool
}
```

`SplitForCompletion` detects this by checking if the current pipeline's last wordbreak is a `WORDBREAK_LAMBDA_PIPE` and the cursor is after it but before a closing `|`.

Actually, let me think about this more carefully. The completion caller (carapace) needs to know:
- Am I completing a **command/argument** or a **lambda parameter**?
- If I'm completing a lambda parameter, what context am I in?

For the `bat | {|` case:
- The current pipeline is `bat` (after the real pipe)
- The current "form" within that pipeline starts with `{|...`
- The cursor is in the parameter list of the lambda

But carapace-shlex doesn't have a concept of "forms" within a pipeline — it just has words. The lambda context is a higher-level concept.

For now, the minimal useful change is:
1. Don't split the pipeline at lambda `|` (so `CurrentPipeline` returns the right thing)
2. Expose `InLambdaParams` so the caller knows

Actually, I realize there's a more fundamental issue. In `bat | {|`, the real pipe `|` separates `bat` from the next form. The next form starts with `{|...}` which is a lambda. But the lambda is the **command** of the second form, not an argument. So the completion caller needs to know: "the current form's command is a lambda, and the cursor is in its parameter list."

This is getting into parser territory. The flat tokenizer can't easily express "the current form starts with a lambda." But for completion purposes, the key insight is:

**When the cursor is inside `{|...|`, completion should offer parameter name completions (or nothing), not command completions.**

The simplest way to signal this: set `InLambdaParams = true` in `CompletionContext`. The caller checks this flag and adjusts behavior.

#### Step 8: How `SplitForCompletion` detects `InLambdaParams`

After `SplitWith` with the post-pass, scan the current pipeline's tokens:

```go
// Check if cursor is inside a lambda parameter list.
// This is true when the last WORDBREAK_LAMBDA_PIPE in the pipeline
// is the last wordbreak (no closing | seen after it).
func isInLambdaParams(pipeline TokenSlice) bool {
    foundOpen := false
    for _, t := range pipeline {
        if t.Type == WORDBREAK_TOKEN && t.WordbreakType == WORDBREAK_LAMBDA_PIPE {
            foundOpen = !foundOpen // toggle: first | opens, second | closes
        }
    }
    return foundOpen // true if we saw an odd number of lambda pipes
}
```

Wait, this toggle approach is too simplistic. The first `|` opens, the second closes. If we've seen one `|` and no closing `|`, we're in params. If we've seen two `|`, we're in the body. If we've seen three `|` (shouldn't happen in params), it's an error.

Actually, the toggle works for the simple case:
- 1 lambda pipe → in params (toggle: true)
- 2 lambda pipes → in body (toggle: false)
- 0 lambda pipes → not in lambda (toggle: false)

But if there are nested lambdas, the toggle breaks. For now, nesting is an edge case — let's handle the common case and note nesting as a limitation.

Hmm, actually the toggle doesn't work either. Consider `{|a| echo}`:
- `|` (first) → toggle true (in params)
- `|` (second) → toggle false (in body)
- Result: false → not in lambda params. Correct!

And `{|`:
- `|` → toggle true
- Result: true → in lambda params. Correct!

And `{|a`:
- `|` → toggle true
- Result: true → in lambda params. Correct!

And `{|a|`:
- `|` (first) → toggle true
- `|` (second) → toggle false
- Result: false → not in lambda params. Correct! (cursor is after the closing |, in the body)

The toggle approach works for the simple non-nested case. Let's use it.

But wait — `{|a| echo $a | grep foo}` has 2 lambda pipes and 1 regular pipe. The toggle on lambda pipes only: true, false → false. Correct (we're in the body, which has a real pipe).

What about `{ |` (lambda with space after `{`, no param pipe yet)? The `{` is followed by space, so it's a lambda (braceLambdaOpen). But there's no `|` yet — the parameter list hasn't started. In this case:
- No lambda pipes in the pipeline
- Toggle: false
- `InLambdaParams`: false

But the cursor is at `{ ` which is the start of a lambda body (no params). This is actually command position — elvish would complete commands here. So `InLambdaParams: false` is correct.

What about `{ |` (lambda with space then pipe)? This is `{` followed by space (lambda) then `|` (start of param list). The `{` and `|` don't adjoin (space between them). So:
- `{` is a WORD token (span 0-1)
- `|` is a WORDBREAK token (span 2-3), reclassified as LAMBDA_PIPE
- Toggle: true
- `InLambdaParams`: true. Correct!

Good, the toggle approach works.

## Implementation Steps

### 1. Add `WORDBREAK_LAMBDA_PIPE` to `wordbreak.go`

```go
// In the WordbreakType const block:
WORDBREAK_LAMBDA_PIPE  // | inside {|params| ...} — lambda parameter delimiter (elvish)

// In wordbreakTypes map:
WORDBREAK_LAMBDA_PIPE: "WORDBREAK_LAMBDA_PIPE",

// IsPipelineDelimiter: do NOT include WORDBREAK_LAMBDA_PIPE
// IsRedirect: do NOT include WORDBREAK_LAMBDA_PIPE
```

### 2. Add `PostProcessor` optional interface

In `format.go` or `shlex.go`:

```go
// PostProcessor is an optional interface for formats that need to
// reclassify tokens after the main tokenization pass.
type PostProcessor interface {
    PostProcess(tokens TokenSlice) TokenSlice
}
```

### 3. Call `PostProcess` in `SplitWith`

In `shlex.go` `SplitWith`:

```go
func SplitWith(s string, format Format) (TokenSlice, error) {
    // ... existing tokenization ...
    if pp, ok := format.(PostProcessor); ok {
        tokens = pp.PostProcess(tokens)
    }
    return tokens, nil
}
```

### 4. Implement `PostProcess` for elvish in `format_elvish.go`

```go
func (elvishFormat) PostProcess(tokens TokenSlice) TokenSlice {
    return resolveLambdaPipes(tokens)
}
```

### 5. Implement `resolveLambdaPipes` in `format_elvish.go`

The brace context state machine as described above, tracking `{`/`}` word tokens and reclassifying `|` wordbreaks inside lambda parameter lists.

### 6. Add `InLambdaParams` to `CompletionContext`

In `completion.go`:

```go
type CompletionContext struct {
    // ... existing fields ...

    // InLambdaParams is true when the cursor is inside a lambda parameter
    // list (e.g. after "{|" in elvish). The completion caller should
    // complete parameter names, not commands or arguments.
    InLambdaParams bool
}
```

### 7. Detect `InLambdaParams` in `SplitForCompletion`

In `completion.go`, after building the pipeline, scan for unclosed lambda pipes:

```go
// After existing logic:
lambdaPipeCount := 0
for _, t := range pipeline {
    if t.Type == WORDBREAK_TOKEN && t.WordbreakType == WORDBREAK_LAMBDA_PIPE {
        lambdaPipeCount++
    }
}
ctx.InLambdaParams = lambdaPipeCount % 2 == 1
```

### 8. Tests in `format_elvish_test.go`

```go
func TestElvishFormat_LambdaPipe(t *testing.T) {
    // {| should not split pipeline — | is lambda param delimiter
    tokens, err := SplitWith("bat | {|", ElvishFormat())
    // ...
    // Second | should be WORDBREAK_LAMBDA_PIPE, not WORDBREAK_PIPE
    // CurrentPipeline should include the lambda tokens
}

func TestElvishFormat_LambdaPipeParams(t *testing.T) {
    // {|a b| should have both | as LAMBDA_PIPE
    tokens, err := SplitWith("bat | {|a b|", ElvishFormat())
    // ...
}

func TestElvishFormat_LambdaBodyPipe(t *testing.T) {
    // {|a| cmd1 | cmd2} — third | is a real pipe in lambda body
    tokens, err := SplitWith("{|a| cmd1 | cmd2}", ElvishFormat())
    // ...
}

func TestElvishFormat_BracedListNotLambda(t *testing.T) {
    // {a,b} should not trigger lambda detection
    tokens, err := SplitWith("echo {a,b}", ElvishFormat())
    // ...
}

func TestElvishFormat_NestedLambda(t *testing.T) {
    // {|a| {|b| echo $a $b }} — nested lambdas
    tokens, err := SplitWith("{|a| {|b| echo $a $b }}", ElvishFormat())
    // ...
}

func TestElvishFormat_CompletionInLambdaParams(t *testing.T) {
    ctx := SplitForCompletion("bat | {|", ElvishFormat())
    // InLambdaParams should be true
}

func TestElvishFormat_CompletionInLambdaBody(t *testing.T) {
    ctx := SplitForCompletion("{|a| ", ElvishFormat())
    // InLambdaParams should be false (after closing |)
}
```

## Limitations and Future Work

1. **Nested lambdas**: The toggle-based `InLambdaParams` detection doesn't handle nested lambdas correctly. A proper stack-based approach would be needed for `{|a| {|b| ...` where the cursor is in the inner lambda's params. The `resolveLambdaPipes` post-pass uses a stack, so the token reclassification is correct for nesting, but the `InLambdaParams` toggle in `SplitForCompletion` would need to be stack-based too.

2. **Lambda body context**: After the closing `|` in `{|params| body}`, the cursor is in the lambda body which is a `Chunk` — full command completion should work there. The current design handles this (WORDBREAK_PIPE in the body stays as a real pipe), but the `Words()` merging of `{` with adjacent tokens may produce unexpected word boundaries. A future improvement could filter out `{` and `}` tokens from the word list when they're lambda/braced delimiters.

3. **Braced lists**: `{a,b}` is a braced list, not a lambda. The post-pass detects this (no `|` or whitespace after `{`), so `|` inside a braced list would be unusual. The current code treats `|` in `braceBraced` state as `WORDBREAK_LAMBDA_PIPE` (non-pipe), which is safe — it just means the pipeline won't split there.

4. **`{` embedded in words**: If `{` appears as part of a larger word (not at the start), it won't be a standalone `WORD_TOKEN` with `Value == "{"`. The post-pass only checks `t.Value == "{"`, so embedded braces are ignored. This is correct — we only care about `{` at the start of a token where it's a lambda/braced-list opener.

5. **Other shells**: This post-pass is elvish-specific. Other shells don't use `|` as a parameter delimiter inside braces. The `PostProcessor` optional interface ensures no other format is affected.

6. **Elvish's old lambda syntax**: `[args]{...}` uses `[...]` for parameters. This is a separate syntax from `{|...|...}`. The `[` and `]` are not wordbreaks in the current elvish format, so `[args]` would be a single word. This syntax is not handled by the post-pass. It's less common and could be added later if needed.

---

# Subshell / Subexpression Support — Implementation Plan

## Problem

The flat state machine has no nesting awareness for `$(...)`, `` `...` ``, `$((...))`, or `(...)` command substitutions. This breaks two things:

1. **Word splitting**: `echo $(echo test)` produces `["echo", "$(echo", "test)"]` — three words instead of two. The `(` and `)` are in `BASH_WORDBREAKS` but classified as `WORDBREAK_UNKNOWN`. They break the word but carry no nesting context. After `Words()` merges by `Span` adjacency, the space inside `$(echo test)` prevents merging — the substitution is split across two words.

2. **Pipeline splitting**: `echo foo $(bar | grep x) baz` splits at the inner `|` because `Pipelines()` sees `WORDBREAK_PIPE` and doesn't know it's inside a substitution. The outer command is broken into two pipelines, both wrong.

3. **No inner completion**: When the cursor is inside `echo $(git ch` there is no way for a completion caller to get the inner command's words (`["git", "ch"]`). The context always describes the outer level.

### Current behavior (confirmed by testing)

```
echo $(echo test)        → Words: ["echo", "$(echo", "test)"]     ← broken: 3 words, should be 2
echo `echo test`         → Words: ["echo", "`echo test`"]        ← accidental: 2 words, backtick not tracked
echo $((1+2))            → Words: ["echo", "$((1+2))"]           ← accidental: works when no spaces inside
echo foo $(bar | grep x) baz → pipeline splits at inner |       ← broken
echo $(echo $(echo test)) → Words: ["echo", "$(echo", "$(echo", "test))"]  ← broken
```

### Consumer context

The carapace `origin/shlex-v2` branch is the WIP migration that uses `SplitForCompletion`. It calls `SplitForCompletion` at four sites:

| File | Fields consumed |
|------|-----------------|
| `internal/shell/bash/patch.go` | `ctx.Words`, `ctx.Prefix`, `ctx.IsRedirect`, `ctx.CurrentWord` |
| `internal/shell/cmd_clink/patch.go` | `ctx.Words` |
| `internal/shell/zsh/action.go` | `ctx.QuotingState`, `ctx.RawCurrentWord` |
| `action.go` (`Action.Split`/`SplitP`) | `ctx.Words`, `ctx.IsRedirect`, `ctx.CurrentWord`, `ctx.QuotingState`, `ctx.Prefix` |

All use `shlex.SplitForCompletion(compline, format)` and consume `CompletionContext` fields directly. The `origin/shlex-v2` branch's `action.go` also uses `Span.Start` from pipeline tokens for prefix calculation:

```go
prefix := originalValue[:pipelineTokens.Words().CurrentToken().Span.Start]
```

This means inner tokens must retain their original `Span` offsets relative to the full input string, even when extracted as the inner context.

## Scope — per-shell substitution forms

| Shell | Form | Opener | Closer | Notes |
|-------|------|--------|--------|-------|
| bash/zsh/oil | `$(...)` | `$` + `(` | `)` | `$` is a WORD_TOKEN, `(` is a WORDBREAK_TOKEN — adjacency-detected |
| bash/zsh/oil/tcsh | `` `...` `` | `` ` `` | `` ` `` | backtick is NOT a wordbreak — passes as word char, detect-only |
| bash/zsh/oil | `$((...))` | `$` + `((` | `))` | arithmetic; not a command context |
| bash/zsh/oil | `<(...)` / `>(...)` | `<(`/`>(` | `)` | process substitution; same paren tracking |
| fish | `(...)` | `(` | `)` | command substitution (bare parens, no `$` prefix) |
| elvish | `(...)` | `(` | `)` | output capture (already `WORDBREAK_OUTPUT_CAPTURE`) |
| nushell | `(...)` | `(` | `)` | subexpression |
| powershell | `$(...)` | `$` + `(` | `)` | subexpression |
| xonsh | `$(...)` / `@(...)` | `$`+`(` or `@`+`(` | `)` | subprocess / Python eval |
| cmd | — | — | — | no subshell |

**Common thread**: `(` and `)` delimit nesting in every shell except cmd. POSIX adds `$(` and backtick as distinct openers. The tokens are already in the stream — we need a post-pass to track nesting.

## Design

### Principle: inner-first context

When the cursor is inside an unclosed substitution scope, `SplitForCompletion` returns a `CompletionContext` where **all fields describe the inner command**, not the outer. The caller gets the inner words, inner current word, inner quoting state — everything needed to complete the inner command as if it were standalone. `SubstitutionDepth` tells the caller how deep it is.

This is the simplest API for carapace: no code changes needed beyond checking `SubstitutionDepth > 0` (and even that is optional — the fields are already correct for completion).

### Step 1: `WordbreakType` for substitution delimiters

`wordbreak.go`:

```go
WORDBREAK_SUBSTITUTION_OPEN   // ( that opens a command/arithmetic/subexpression scope
WORDBREAK_SUBSTITUTION_CLOSE  // ) that closes such a scope
```

Both: `IsPipelineDelimiter() == false`, `IsRedirect() == false`. This ensures `Pipelines()` doesn't treat the delimiters themselves as pipeline breaks.

### Step 2: `SubstitutionScope` and `SubstitutionScopes()`

New file `substitution.go`:

```go
type SubstitutionKind int

const (
    SubstitutionCommand    SubstitutionKind = iota // $(...) or bare (...) 
    SubstitutionArithmetic                          // $((...))
    SubstitutionBacktick                            // `...`
)

type SubstitutionScope struct {
    OpenIndex  int              // TokenSlice index of opener (-1 for backtick-in-word)
    CloseIndex int              // TokenSlice index of closer (-1 if unclosed)
    Kind       SubstitutionKind
    Depth      int              // nesting depth at this scope
}

// SubstitutionScopes returns all substitution scopes in the token slice,
// ordered by open position. Unclosed scopes have CloseIndex == -1.
func (t TokenSlice) SubstitutionScopes() []SubstitutionScope
```

The scanner tracks a stack of open scopes. On `WORDBREAK_SUBSTITUTION_OPEN`, push. On `WORDBREAK_SUBSTITUTION_CLOSE`, pop. Backtick scopes are detected by scanning WORD_TOKEN `RawValue` for unescaped backticks — these get `OpenIndex == -1` (no single opener token) and are detect-only (can't split inner words since the backtick content is already merged into one word token).

### Step 3: Shared POSIX `PostProcess`

New shared function used by bash/zsh/oil/tcsh formats:

```go
// posixSubstitutionPostProcess reclassifies ( and ) as substitution
// delimiters, merging $ + ( adjacency into a single opener token.
// Does NOT reclassify inner pipeline operators — that's handled by
// Pipelines() respecting substitution depth.
func posixSubstitutionPostProcess(tokens TokenSlice) TokenSlice {
    // 1. Merge $ + ( adjacency into WORDBREAK_SUBSTITUTION_OPEN
    //    Detect $(( for arithmetic (second ( after $()
    // 2. Reclassify standalone ( as WORDBREAK_SUBSTITUTION_OPEN
    //    (for process substitution <(, >( — the < or > is already
    //    classified as redirect, the ( is the substitution opener)
    // 3. Reclassify ) as WORDBREAK_SUBSTITUTION_CLOSE
    //    (matching depth: arithmetic closes on ))
    // 4. Scan WORD_TOKEN RawValue for unescaped backticks,
    //    flag unclosed backtick scopes
}
```

bash/zsh/oil call this directly. tcsh calls it with backtick awareness (tcsh backtick command substitution is common).

### Step 4: Non-POSIX `PostProcess` extensions

- **fish**: Add `(` `)` to classifier as wordbreaks, classify as `WORDBREAK_SUBSTITUTION_OPEN/CLOSE`. Fish uses bare `()` for command substitution (no `$` prefix).
- **elvish**: Already classifies `(` `)` as `WORDBREAK_OUTPUT_CAPTURE`. Extend `PostProcess` to track paren nesting depth so `Pipelines()` can respect it. Reclassify `WORDBREAK_OUTPUT_CAPTURE` as `WORDBREAK_SUBSTITUTION_OPEN/CLOSE` (or teach `Pipelines()` about `WORDBREAK_OUTPUT_CAPTURE`).
- **nushell**: Add `(` `)` to classifier, classify as substitution delimiters.
- **PowerShell**: `$` + `(` → `$(`. Add `(` `)` to wordbreaks, detect `$(` adjacency in `PostProcess`.
- **xonsh**: `$` + `(` or `@` + `(` → `$(` / `@(`. Same pattern as PowerShell.

### Step 5: `Pipelines()` — respect substitution depth

The critical pipeline fix. Modify `Pipelines()` to track substitution scope depth: when inside a substitution scope (depth > 0), don't split on pipeline delimiters.

```go
func (t TokenSlice) Pipelines() []TokenSlice {
    pipelines := make([]TokenSlice, 0)
    pipeline := make(TokenSlice, 0)
    depth := 0

    for _, token := range t {
        switch {
        case token.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN:
            depth++
            pipeline = append(pipeline, token)
        case token.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE:
            depth--
            pipeline = append(pipeline, token)
        case depth == 0 && token.Type == WORDBREAK_TOKEN &&
            token.WordbreakType.IsPipelineDelimiter():
            pipelines = append(pipelines, pipeline)
            pipeline = make(TokenSlice, 0)
        default:
            pipeline = append(pipeline, token)
        }
    }
    return append(pipelines, pipeline)
}
```

This way, inner `|` retains its `WORDBREAK_PIPE` classification. When `Pipelines()` runs on the full token slice, it skips the inner pipe because depth > 0. When we extract the inner tokens and call `Pipelines()` on them, the inner pipe correctly splits the inner pipeline.

`CurrentPipeline()` on the full slice returns the outer pipeline. When we extract inner tokens and call `CurrentPipeline()` on them, it returns the inner pipeline (the one after the inner `|`). No reclassification needed — the `PostProcess` only merges `$` + `(` and reclassifies `)`.

### Step 6: `WordsWithSubstitutions()`

New method on `TokenSlice`, used by `SplitForCompletion` for the **outer** context (when cursor is NOT inside a substitution). Merges substitution content into single words:

```go
// WordsWithSubstitutions merges tokens, treating substitution scopes
// as single words. When a WORDBREAK_SUBSTITUTION_OPEN is encountered,
// all tokens until the matching WORDBREAK_SUBSTITUTION_CLOSE (or end
// of slice if unclosed) are merged into one word.
//
// For unclosed scopes (cursor inside), the inner tokens are NOT merged
// into the outer word — they're left as separate tokens so the inner
// context can be built separately.
func (t TokenSlice) WordsWithSubstitutions() TokenSlice {
    words := make(TokenSlice, 0)
    depth := 0
    current := &Token{}

    for index, token := range t {
        switch {
        case token.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN:
            if depth == 0 {
                // Start new word that will absorb the substitution
                current = &Token{
                    Type: WORD_TOKEN,
                    Span: token.Span,
                }
            }
            depth++
            current.RawValue += token.RawValue
            current.Value += token.Value
            current.Span.End = token.Span.End
            current.State = token.State
        case token.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE:
            current.RawValue += token.RawValue
            current.Value += token.Value
            current.Span.End = token.Span.End
            current.State = token.State
            depth--
            if depth == 0 {
                words = append(words, *current)
            }
        case depth > 0:
            // Inside substitution — absorb everything
            current.RawValue += token.RawValue
            current.Value += token.Value
            current.Span.End = token.Span.End
            current.State = token.State
        default:
            // Normal Words() logic
            if index == 0 {
                words = append(words, token)
            } else if t[index-1].adjoins(token) {
                words[len(words)-1].Value += token.Value
                words[len(words)-1].RawValue += token.RawValue
                words[len(words)-1].Span.End = token.Span.End
                words[len(words)-1].State = token.State
            } else {
                words = append(words, token)
            }
        }
    }
    // Unclosed substitution at end — the current word includes
    // everything up to cursor. Don't append it; the inner context
    // will be built separately.
    return words
}
```

When cursor is inside a substitution (unclosed), `SplitForCompletion` uses the inner tokens with regular `Words()` — the inner pipeline/word logic works as-is.

### Step 7: `CompletionContext` changes

```go
type CompletionContext struct {
    // ... existing fields unchanged ...

    // SubstitutionDepth is the number of unclosed substitution scopes
    // at the cursor position. 0 = cursor at top level.
    // When > 0, all other fields (Words, CurrentWord, etc.) describe
    // the innermost substitution's command, not the outer command.
    SubstitutionDepth int

    // SubstitutionKind indicates what type of substitution the cursor
    // is inside (command, arithmetic, backtick). Only meaningful when
    // SubstitutionDepth > 0.
    SubstitutionKind SubstitutionKind
}
```

No `InnerCompletion` pointer — the top-level fields ARE the inner context. Simpler API, less allocation. When `SubstitutionDepth > 0`:
- `Words` = inner command words (e.g. `["git", "ch"]` for `echo $(git ch`)
- `CurrentWord` = inner current word (e.g. `"ch"`)
- `QuotingState` = inner quoting state
- `Pipeline` = inner pipeline tokens (with original spans)
- `Prefix` = inner wordbreak prefix
- `IsRedirect` = inner redirect detection

### Step 8: `SplitForCompletion` changes

Extract the existing pipeline/redirect/word logic into `buildCompletionContext` helper that operates on any token slice. Then add substitution detection:

```go
func SplitForCompletion(s string, format Format) *CompletionContext {
    tokens, err := SplitWith(s, format)
    if err != nil || len(tokens) == 0 {
        return &CompletionContext{QuotingState: START_STATE}
    }

    // Check for unclosed substitution scopes at cursor
    scopes := tokens.SubstitutionScopes()
    innermost := innermostUnclosedScope(scopes)

    if innermost != nil && innermost.Kind != SubstitutionArithmetic {
        // Cursor inside a command substitution — build context from inner tokens
        innerTokens := tokens[innermost.OpenIndex+1:]
        ctx := buildCompletionContext(innerTokens, format)
        ctx.SubstitutionDepth = countUnclosedScopes(scopes)
        ctx.SubstitutionKind = innermost.Kind
        return ctx
    }

    // Cursor at top level (or inside arithmetic — no inner command)
    return buildCompletionContext(tokens, format)
}

func buildCompletionContext(tokens TokenSlice, format Format) *CompletionContext {
    pipeline := tokens.CurrentPipeline()
    filtered := pipeline.FilterRedirects()
    words := filtered.WordsWithSubstitutions()
    wordStrings := words.Strings()

    ctx := &CompletionContext{
        Words:    wordStrings,
        Pipeline: pipeline,
    }

    // ... existing redirect / current word / prefix / lambda detection logic ...
    return ctx
}
```

For backtick substitution: `innermost.Kind == SubstitutionBacktick` with `OpenIndex == -1`. The inner tokens can't be extracted (backtick content is merged into one word token). Set `SubstitutionDepth` and `SubstitutionKind` but leave `Words`/`CurrentWord` as the outer context. Document as a limitation.

### Step 9: Backtick handling

Backticks are the hardest case because the tokenizer doesn't emit them as wordbreaks — they're embedded in `RawValue` of word tokens. Two approaches:

**A. Post-processor scan (implemented)**: The POSIX `PostProcess` scans WORD_TOKEN `RawValue` for unescaped backticks. When it finds an odd count, the word is inside an unclosed backtick substitution. This is enough for completion (detect "cursor inside backtick") but can't split the inner command into words (the backtick content is already merged into one word token).

**B. State machine change (deferred)**: Add backtick as a wordbreak for POSIX formats and track it in the state machine. This would let `Words()` split `` `echo test` `` into inner words, but changes core lexing behavior and risks breaking existing tests.

Decision: Start with approach A. Backtick command substitution inside completion input is rare. The `$()` form is the modern equivalent and will be fully supported. Document backtick as a known limitation (detect but don't split inner words).

## Implementation Steps

### 1. `wordbreak.go` — new constants

```go
WORDBREAK_SUBSTITUTION_OPEN   // ( or $( that opens a substitution scope
WORDBREAK_SUBSTITUTION_CLOSE  // ) that closes a substitution scope
```

Add to `wordbreakTypes` map. Do NOT add to `IsPipelineDelimiter()` or `IsRedirect()`.

### 2. `substitution.go` (new file)

- `SubstitutionKind` type and constants
- `SubstitutionScope` struct
- `SubstitutionScopes()` method on `TokenSlice`
- `innermostUnclosedScope()` helper
- `countUnclosedScopes()` helper

### 3. `tokenslice.go` — modify `Pipelines()`

Add substitution depth tracking: don't split on pipeline delimiters when inside a substitution scope (depth > 0).

### 4. `tokenslice.go` — add `WordsWithSubstitutions()`

New method that merges substitution content into single words for the outer context.

### 5. POSIX shared `PostProcess`

New shared function `posixSubstitutionPostProcess(tokens TokenSlice) TokenSlice`:
- Merge `$` WORD_TOKEN + `(` WORDBREAK_TOKEN adjacency into `WORDBREAK_SUBSTITUTION_OPEN`
- Detect `$((` for arithmetic
- Reclassify `)` as `WORDBREAK_SUBSTITUTION_CLOSE`
- Scan for unclosed backticks in WORD_TOKEN `RawValue`

bash/zsh/oil: `PostProcess` calls this directly.
tcsh: wraps it with backtick awareness.

### 6. Per-format `PostProcess` extensions

- **fish**: Add `(` `)` to classifier, classify as substitution delimiters in `PostProcess`
- **elvish**: Extend existing `PostProcess` — track paren nesting alongside brace nesting, reclassify `WORDBREAK_OUTPUT_CAPTURE` as `WORDBREAK_SUBSTITUTION_OPEN/CLOSE`
- **nushell**: Add `(` `)` to classifier, classify as substitution delimiters
- **PowerShell**: Add `(` `)` to wordbreaks, detect `$(` adjacency in `PostProcess`
- **xonsh**: Add `(` `)` to wordbreaks, detect `$(` and `@(` adjacency in `PostProcess`

### 7. `completion.go` — new fields and `SplitForCompletion` changes

- Add `SubstitutionDepth` and `SubstitutionKind` to `CompletionContext`
- Extract `buildCompletionContext` helper from existing `SplitForCompletion` logic
- Add substitution scope detection: find innermost unclosed scope, extract inner tokens, build inner context
- Set `SubstitutionDepth`/`SubstitutionKind` on the returned context

### 8. Tests

```
$(echo test)              → one word at outer level: ["echo", "$(echo test)"]
echo $(git ch             → InnerCompletion: Words=["git","ch"], SubstitutionDepth=1
echo foo $(bar | grep x) baz → one pipeline at outer level, inner pipe not split
$((1+2))                  → one word, no inner command context (arithmetic)
echo $(echo $(git ch))    → SubstitutionDepth=2, Words=["git","ch"]
echo $(git ch             → unclosed: SubstitutionDepth=1, Words=["git","ch"]
echo `echo test`          → one word (limitation: no inner context)
echo `git ch              → SubstitutionDepth=1 (detect, no inner words)
```

## Known limitations

1. **Backtick**: detected (odd count in `RawValue`) but inner words not split — `$()` is the modern equivalent, fully supported
2. **Arithmetic `$((...))`**: not a command context — `SubstitutionDepth` is set but no inner `Words` (arithmetic isn't a command to complete)
3. **Process substitution `<(cmd)` / `>(cmd)` (bash/zsh)**: handled by the same `(`/`)` tracking with `<` / `>` prefix detection in the shared `PostProcess`
4. **Brace expansion `{a,b}`**: not substitution (no nested command), out of scope
5. **Heredoc `<<EOF...EOF`**: multi-line, not relevant for single-line completion input
