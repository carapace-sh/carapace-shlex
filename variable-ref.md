# Variable Reference Completion — Reasoning and Plan

## Problem

Completion callers need to detect a variable reference ending the current word
(`echo $HO`, `echo "text$HO`, `echo "test${`) to decide what to complete
(environment variables, shell variables) and how to insert the selection.
Today every consumer re-implements this detection on top of ad-hoc word
splitting, and each re-derivation reintroduces the same bug class.

The trigger case (tabdance): the completion tokenizer strips quotes, so
`echo "text$` reaches the consumer as the word `text$`; a naive
`HasPrefix(word, "$")` check fails and the variable is not completed at all.

## Why this belongs in the lexer

The detection requires exactly the facts the lexer already tracks:

1. **Is the `$` expandable?** The word token's `State` answers this: inside
   `QUOTING_STATE` (single quotes) `$` is literal. Bash's `-F` completion
   interface cannot answer it — it hands the caller the naive word `text$`
   regardless of quote type (verified: `git "text$` and `git 'text$` both
   yield cur=`text$`).
2. **Is the `$` escaped?** `\$HO` is literal while `\\$HO` expands. This
   needs the format's escape rules; counting backslashes against
   `RawValue` is only correct when the format defines backslash escapes.
3. **Where does the name start?** Last unescaped `$`, brace form (`${`),
   valid name-prefix check. Closed expansions (`${HOME}`) and command
   substitutions (`$(...)`) must not be mistaken for references.

A shared implementation gets hardened once against this corpus; today each
shell snippet (carapace-bin, tabdance) re-encodes it and fails differently.

## Design: two layers

### Layer 1 — expansion facts on the lexer word (shared, consistent)

`VariableRef` is detected on the lexer's final word — quote-aware,
escape-aware, identical across shells, because expansion semantics of the
word are identical across shells. `Prefix` deliberately does **not** carry
the literal text before the `$`: that text is only meaningful for insertion
in some shells, and carrying it would bake one shell's replacement model
into the shared struct.

```go
type VariableRef struct {
    Name   string // variable name typed so far
    Brace  bool   // `${` form
    Span   Span   // span of the `$`/`${` opener within the raw input
}

func (c *CompletionContext) VariableRef() (VariableRef, bool) // false when not expandable
```

### Layer 2 — the shell's replacement word (per-format, interface-driven)

What the shell's completion *interface* hands back as the word-to-be-replaced
is a separate, per-shell fact:

- **bash**: naive `COMP_WORDS` split on `COMP_WORDBREAKS` — quotes and
  operators break words, so `git "text$` yields `text$` but
  `git "text $HO` yields `$HO`. This is where the insertion prefix comes
  from: raw suffix of the current word's `RawValue` after its last
  wordbreak character, up to and including the unescaped `$`/`${`.
- **zsh**: `${words}` splits on whitespace only (quotes stay inside the
  word); quote stripping happens later in compsys (`IPREFIX`) at insertion
  time. Same concept, different rule set — so this layer is
  format-parameterized, not bash-only.
- **fish/elvish/nushell/xonsh/powershell**: the completion API resolves
  quoting itself; layer 2 is a no-op (fish additionally does not use `${`
  as a variable form at all, which is layer 1 format input, see below).

```go
// Raw word as the shell's naive splitter sees it (sibling of WordbreakPrefix).
func (c *CompletionContext) RawReplacementWord() string
```

The bash consumer then composes: layer 1 decides *what* to complete,
layer 2 decides *what text gets replaced* — insertion value is
`RawReplacementWord` up to the `$` + `Name`.

## Format input the detection needs

- Which sigils are expansion sigils (`$` for POSIX shells; nushell `$`,
  elvish `$`, xonsh `$`; **fish: no `${` form**).
- Whether the sigil is active in the current `State` (single-quote rules
  already differ per format: `NonEscapingQuoteEscapes` etc.).
- Valid name characters per format.

Expressed as an optional interface on `Format` (no-op = no variable
expansion for that shell), following the `PostProcessor` pattern:

```go
type VariableExpander interface {
    VariableRef(word Token) (VariableRef, bool)
}
```

with a shared `bashVariableRef` helper in a new `variable.go` that the
POSIX-like formats call with their own name-character set.

## Implementation plan

1. `variable.go` — `VariableRef` type + bash implementation:
   single-quote state check, odd-backslash escape check on `RawValue`,
   brace form, name-prefix validation (`[A-Za-z0-9_]*`), `Span` from the
   token's raw offsets.
2. `format_bash.go` (+ zsh, nushell, elvish, xonsh where trivial) —
   implement `VariableExpander`.
3. `completion.go` — `CompletionContext.VariableRef()` delegating to the
   format via `SplitForCompletion` (store the format or the resolved ref
   on the context; prefer resolving in `buildCompletionContext` so the
   context stays a plain struct — add a `VariableRef *VariableRef` field,
   nil when absent).
4. Layer 2: `CompletionContext.RawReplacementWord()` — bash: suffix of
   `RawCurrentWord` after the last wordbreak rune from the format's
   classifier (must honor `COMP_WORDBREAKS` env, same as the classifier);
   other formats: whole `RawCurrentWord`.
5. Tests in `completion_test.go` / `format_*_test.go` (corpus below).
6. README table note + AGENTS.md one-liner.

## Test corpus (from the tabdance fix)

```
echo $HO          → Name=HO
echo ${HO         → Name=HO, Brace
echo "text$HO     → Name=HO            (quote stripped by lexer, still a ref)
echo "text $HO    → Name=HO            (lexer word: `text $HO`)
echo 'text$HO     → not a ref          (single quotes)
echo "text"$HO    → Name=HO            (quote closed, State back to IN_WORD)
echo ${HOME}      → not a ref          (closed expansion)
echo "a$          → Name=""            (empty name prefix is valid)
echo \$HO         → not a ref          (escaped)
echo "\\$HO"      → Name=HO            (escaped backslash, real ref)
echo $(           → not a ref          (command substitution)
echo a$-x         → not a ref          (invalid name char)
fish: echo ${     → not a ref          (fish has no brace form)
```

## Migration

tabdance (`tabdance-bin`) currently carries the layer-1+2 logic locally in
`internal/shell/bash` (`Variable`/`VariableRef`, ~50 lines + tests). Once a
shlex release with this API exists, it replaces that code with
`shlex.SplitForCompletion(line, shlex.BashFormat{}).VariableRef()` and the
bash insertion prefix from `RawReplacementWord()`.

## Open questions

1. Field on `CompletionContext` vs method computing on demand — leaning
   field resolved during `buildCompletionContext` (single code path, no
   format plumbing in the method).
2. Should layer 2 be a method at all, or just documented math over
   `RawCurrentWord` + wordbreak runes? Keeping it a method hides the
   per-format difference from consumers.
3. tcsh: `COMMAND_LINE`-based interface — naive-split rules differ again;
   defer until a tcsh consumer needs it.

## Implementation notes

Implemented on `feat/variable-ref-completion`:

- `variable.go` — `VariableRef` (`Name`, `Brace`, `Span`), the optional
  `VariableExpander` and `NaiveWordSplitter` format interfaces, and the
  shared `posixVariableRef` / `naiveSplitWord` helpers.
- `bashFormat` implements both; `zshFormat` implements `VariableExpander`
  only. fish, nushell, elvish, xonsh, oil, tcsh, cmd, powershell do not
  implement the interfaces yet (their expansion rules need per-format
  name-character and sigil decisions).
- `CompletionContext` gains `VariableRef *VariableRef` and
  `RawReplacementWord string`, both resolved in
  `buildCompletionContext(tokens, format)`.

Decisions on the open questions above:

1. **Field, not method.** `VariableRef` and `RawReplacementWord` are
   resolved once during `buildCompletionContext`; the context stays a
   plain struct with no format plumbing.
2. **Layer 2 is a field**, filled via the `NaiveWordSplitter` optional
   interface (`bashFormat.NaiveSplitWord` splits on the classifier's
   space/wordbreak/quote/comment classes, escape character excluded,
   honoring `COMP_WORDBREAKS`); formats without a naive word interface
   keep the whole `RawCurrentWord`.
3. **tcsh deferred**, as planned.

Consumer composition for bash-style insertion: strip `VariableRef.Name`
from the end of `RawReplacementWord` and append the completed name
(`"text$HO` → `text$` + `HOME`).
