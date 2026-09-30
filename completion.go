package shlex

// CompletionContext describes the completion state at the end of the input.
// It is the primary API for completion callers, replacing the manual
// tokens.CurrentPipeline().FilterRedirects().Words().CurrentToken() chains.
type CompletionContext struct {
	// Words are the dequoted word values in the current pipeline
	// (redirects filtered). When the cursor is inside a substitution
	// scope (e.g. $(...), these are the inner command's words.
	Words []string

	// CurrentWord is the word at the cursor position (dequoted Value).
	CurrentWord string

	// RawCurrentWord is the raw source of the current word (including quotes).
	// Use this to detect quotation state when QuotingState alone is insufficient.
	RawCurrentWord string

	// Prefix is the wordbreak prefix up to the cursor.
	Prefix string

	// QuotingState is the lexer state of the current word.
	// IN_WORD_STATE, QUOTING_STATE, QUOTING_ESCAPING_STATE, QUOTING_TRIPLE_STATE,
	// QUOTING_TRIPLE_ESCAPING_STATE, or ESCAPING_STATE.
	QuotingState LexerState

	// IsRedirect is true when the cursor is completing a redirect target
	// (e.g. after >, >>, <, etc.).
	IsRedirect bool

	// InLambdaParams is true when the cursor is inside a lambda parameter
	// list (e.g. after "{|" in elvish). The completion caller should
	// complete parameter names, not commands or arguments.
	InLambdaParams bool

	// VariableRef describes a variable reference ending the current word,
	// detected lexically on the lexer's final word. Nil when the word does
	// not end in a variable reference, or when the format does not
	// implement variableExpander.
	//
	// For insertion, shells with a naive word interface (see
	// RawReplacementWord) replace the typed name suffix of
	// RawReplacementWord (VariableRef.Name) with the completed name;
	// shells whose completion API resolves quoting handle the prefix
	// themselves.
	VariableRef *VariableRef

	// RawReplacementWord is the raw text the shell replaces on insertion.
	// It equals RawCurrentWord unless the format implements
	// naiveWordSplitter (bash: the naive COMP_WORDS suffix, so
	// `"text $HO` yields `$HO` while the lexer word is `text $HO`).
	RawReplacementWord string

	// Pipeline is the raw token slice of the current pipeline (before
	// redirect filtering and word merging). Use this as an escape hatch
	// for edge cases not covered by the fields above.
	Pipeline TokenSlice

	// SubstitutionDepth is the number of unclosed substitution scopes
	// at the cursor position. 0 = cursor at top level. When > 0, all
	// other fields (Words, CurrentWord, etc.) describe the innermost
	// substitution's command, not the outer command.
	SubstitutionDepth int
}

// SplitForCompletion parses s and returns a CompletionContext describing
// the completion state at the end of the string, using the given format.
//
// When the cursor is inside an unclosed substitution scope (e.g. inside
// $(...), the context describes the innermost substitution's command,
// not the outer command.
func SplitForCompletion(s string, format Format) *CompletionContext {
	f, ok := formatImplFor(format)
	if !ok {
		return &CompletionContext{QuotingState: START_STATE}
	}
	tokens, err := Split(s, format)
	if err != nil || len(tokens) == 0 {
		return &CompletionContext{QuotingState: START_STATE}
	}

	// If cursor is inside an unclosed command substitution, build the
	// context from the inner tokens.
	if scope := innermostUnclosedCommandScope(tokens); scope >= 0 {
		ctx := buildCompletionContext(tokens[scope+1:], f)
		ctx.SubstitutionDepth = countUnclosedCommandScopes(tokens)
		return ctx
	}

	return buildCompletionContext(tokens, f)
}

// buildCompletionContext derives the completion context fields from a
// token slice.
func buildCompletionContext(tokens TokenSlice, format formatImpl) *CompletionContext {
	pipeline := tokens.CurrentPipeline()
	filtered := pipeline.FilterRedirects()
	words := filtered.WordsWithSubstitutions()
	wordStrings := words.Strings()

	ctx := &CompletionContext{
		Words:    wordStrings,
		Pipeline: pipeline,
	}

	if len(pipeline) >= 2 {
		prev := pipeline[len(pipeline)-2]
		if prev.WordbreakType.IsRedirect() {
			ctx.IsRedirect = true
		}
	}

	var current *Token
	if ctx.IsRedirect {
		current = &pipeline[len(pipeline)-1]
	} else if len(words) > 0 {
		current = &words[len(words)-1]
	}
	if current != nil {
		ctx.CurrentWord = current.Value
		ctx.RawCurrentWord = current.RawValue
		ctx.QuotingState = current.State

		if expander, ok := format.(variableExpander); ok {
			if ref, ok := expander.VariableRef(*current); ok {
				ctx.VariableRef = &ref
			}
		}
	}

	ctx.RawReplacementWord = ctx.RawCurrentWord
	if splitter, ok := format.(naiveWordSplitter); ok {
		ctx.RawReplacementWord = splitter.NaiveSplitWord(ctx.RawReplacementWord)
	}

	ctx.Prefix = pipeline.WordbreakPrefix()

	lambdaPipeCount := 0
	for _, t := range pipeline {
		if t.Type == WORDBREAK_TOKEN && t.WordbreakType == WORDBREAK_LAMBDA_PIPE {
			lambdaPipeCount++
		}
	}
	ctx.InLambdaParams = lambdaPipeCount%2 == 1

	return ctx
}
