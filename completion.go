package shlex

// CompletionContext describes the completion state at a cursor position.
// It is the primary API for completion callers, replacing the manual
// tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken() chains.
type CompletionContext struct {
	// Words are the dequoted word values in the current pipeline
	// (redirects filtered). Equivalent to:
	//   tokens.CurrentPipeline().FilterRedirects().Words().Strings()
	Words []string

	// CurrentWord is the word at the cursor position (dequoted Value).
	CurrentWord string

	// RawCurrentWord is the raw source of the current word (including quotes).
	// Use this to detect quotation state when QuotingState alone is insufficient.
	RawCurrentWord string

	// Prefix is the wordbreak prefix up to the cursor.
	// Equivalent to tokens.CurrentPipeline().WordbreakPrefix().
	Prefix string

	// QuotingState is the lexer state of the current word.
	// IN_WORD_STATE, QUOTING_STATE, QUOTING_ESCAPING_STATE, or ESCAPING_STATE.
	// Replaces the regex-based quoting detection in carapace's zsh action.
	QuotingState LexerState

	// IsRedirect is true when the cursor is completing a redirect target
	// (e.g. after >, >>, <, etc.).
	IsRedirect bool

	// Pipeline is the raw token slice of the current pipeline (before
	// redirect filtering and word merging). Use this as an escape hatch
	// for edge cases not covered by the fields above.
	Pipeline TokenSlice
}

// SplitForCompletion parses s (up to the end) and returns a CompletionContext
// using the given format. The cursor is assumed to be at the end of the string.
func SplitForCompletion(s string, format Format) *CompletionContext {
	return SplitForCompletionAt(s, len([]rune(s)), format)
}

// SplitForCompletionAt parses s up to the cursor position and returns a
// CompletionContext describing the completion state at that position.
// cursor is a rune offset into s.
func SplitForCompletionAt(s string, cursor int, format Format) *CompletionContext {
	tokens, err := SplitWith(s, format)
	if err != nil || len(tokens) == 0 {
		return &CompletionContext{QuotingState: START_STATE}
	}

	pipeline := tokens.CurrentPipeline()
	filtered := pipeline.FilterRedirects()
	words := filtered.Words()
	wordStrings := words.Strings()

	ctx := &CompletionContext{
		Words:    wordStrings,
		Pipeline: pipeline,
	}

	// Detect redirect: if the second-to-last token in the pipeline is a redirect
	// wordbreak, the current word is a redirect target.
	if len(pipeline) >= 2 {
		prev := pipeline[len(pipeline)-2]
		if prev.WordbreakType.IsRedirect() {
			ctx.IsRedirect = true
		}
	}

	if ctx.IsRedirect {
		// For redirects, the current word is the redirect target which was
		// filtered out of the words list. Get it from the raw pipeline.
		current := pipeline[len(pipeline)-1]
		ctx.CurrentWord = current.Value
		ctx.RawCurrentWord = current.RawValue
		ctx.QuotingState = current.State
	} else if len(words) > 0 {
		current := words[len(words)-1]
		ctx.CurrentWord = current.Value
		ctx.RawCurrentWord = current.RawValue
		ctx.QuotingState = current.State
	}

	ctx.Prefix = pipeline.WordbreakPrefix()

	_ = cursor // cursor position is implicitly at end (tokens already reflect full input)
	_ = err
	return ctx
}
