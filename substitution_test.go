package shlex

import "testing"

func TestSubstitution_BashCommandSubstitution(t *testing.T) {
	// echo $(echo test) → two words: "echo" and "$(echo test)"
	tokens, err := SplitWith("echo $(echo test)", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$(echo test)" {
		t.Errorf("Words = %v, want [echo $(echo test)]", words)
	}
}

func TestSubstitution_BashArithmetic(t *testing.T) {
	// echo $((1+2)) → two words: "echo" and "$((1+2))"
	tokens, err := SplitWith("echo $((1+2))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$((1+2))" {
		t.Errorf("Words = %v, want [echo $((1+2))]", words)
	}
}

func TestSubstitution_BashProcessSubstitution(t *testing.T) {
	// echo <(grep foo) → two words: "echo" and "<(grep foo)"
	tokens, err := SplitWith("echo <(grep foo)", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "<(grep foo)" {
		t.Errorf("Words = %v, want [echo <(grep foo)]", words)
	}
}

func TestSubstitution_PipelineDoesNotSplitInsideSubstitution(t *testing.T) {
	// echo foo $(bar | grep x) baz → one pipeline, inner pipe preserved
	tokens, err := SplitWith("echo foo $(bar | grep x) baz", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("Pipelines = %d, want 1", len(pipelines))
	}
	words := pipelines[0].WordsWithSubstitutions().Strings()
	if len(words) != 4 || words[0] != "echo" || words[1] != "foo" ||
		words[2] != "$(bar | grep x)" || words[3] != "baz" {
		t.Errorf("Words = %v, want [echo foo $(bar | grep x) baz]", words)
	}
}

func TestSubstitution_NestedCommandSubstitution(t *testing.T) {
	// echo $(echo $(echo test)) → two words
	tokens, err := SplitWith("echo $(echo $(echo test))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$(echo $(echo test))" {
		t.Errorf("Words = %v, want [echo $(echo $(echo test))]", words)
	}
}

func TestSubstitution_CompletionInsideSubstitution(t *testing.T) {
	// Cursor inside $(git ch → inner context: Words=["git","ch"], depth=1
	ctx := SplitForCompletion("echo $(git ch", BashFormat())
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %v, want 1", ctx.SubstitutionDepth)
	}
	if ctx.SubstitutionKind != SUBSTITUTION_COMMAND {
		t.Errorf("SubstitutionKind = %v, want SUBSTITUTION_COMMAND", ctx.SubstitutionKind)
	}
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.CurrentWord != "ch" {
		t.Errorf("CurrentWord = %q, want \"ch\"", ctx.CurrentWord)
	}
}

func TestSubstitution_CompletionInsideNestedSubstitution(t *testing.T) {
	// Cursor inside nested $(echo $(git ch → depth=2, inner Words=["git","ch"]
	ctx := SplitForCompletion("echo $(echo $(git ch", BashFormat())
	if ctx.SubstitutionDepth != 2 {
		t.Errorf("SubstitutionDepth = %v, want 2", ctx.SubstitutionDepth)
	}
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
}

func TestSubstitution_CompletionInsideArithmetic(t *testing.T) {
	// Cursor inside $((1+2 → arithmetic, depth=1, but no inner command context
	ctx := SplitForCompletion("echo $((1+2", BashFormat())
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %v, want 1", ctx.SubstitutionDepth)
	}
	if ctx.SubstitutionKind != SUBSTITUTION_ARITHMETIC {
		t.Errorf("SubstitutionKind = %v, want SUBSTITUTION_ARITHMETIC", ctx.SubstitutionKind)
	}
	// Arithmetic doesn't create an inner command context
	if len(ctx.Words) != 1 || ctx.Words[0] != "echo" {
		t.Errorf("Words = %v, want [echo]", ctx.Words)
	}
}

func TestSubstitution_CompletionClosedSubstitution(t *testing.T) {
	// Cursor after closed substitution: echo $(echo test)
	// SubstitutionDepth=0, normal outer context
	ctx := SplitForCompletion("echo $(echo test)", BashFormat())
	if ctx.SubstitutionDepth != 0 {
		t.Errorf("SubstitutionDepth = %v, want 0", ctx.SubstitutionDepth)
	}
	if len(ctx.Words) != 2 || ctx.Words[0] != "echo" || ctx.Words[1] != "$(echo test)" {
		t.Errorf("Words = %v, want [echo $(echo test)]", ctx.Words)
	}
}

func TestSubstitution_CompletionInsideSubstitutionWithInnerPipe(t *testing.T) {
	// Cursor inside $(bar | grep x → inner context with pipeline
	ctx := SplitForCompletion("echo foo $(bar | grep x", BashFormat())
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %v, want 1", ctx.SubstitutionDepth)
	}
	// Inner pipeline: "bar | grep x" — CurrentPipeline should be "grep x"
	if len(ctx.Words) != 2 || ctx.Words[0] != "grep" || ctx.Words[1] != "x" {
		t.Errorf("Words = %v, want [grep x]", ctx.Words)
	}
	if ctx.CurrentWord != "x" {
		t.Errorf("CurrentWord = %q, want \"x\"", ctx.CurrentWord)
	}
}

func TestSubstitution_BashBacktickSubstitution(t *testing.T) {
	// echo `echo test` → backtick is not a wordbreak, so the lexer
	// can't merge the content into one word. The backtick characters
	// are embedded in the word values. This is a known limitation:
	// backtick substitution is detected (depth/kind) but inner words
	// are not split.
	tokens, err := SplitWith("echo `echo test`", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	// Backtick not a wordbreak: `echo and test` are separate words
	if len(words) != 3 || words[0] != "echo" || words[1] != "`echo" || words[2] != "test`" {
		t.Errorf("Words = %v, want [echo `echo test`]", words)
	}
}

func TestSubstitution_BashUnclosedBacktick(t *testing.T) {
	// echo `echo test → unclosed backtick, detect-only
	ctx := SplitForCompletion("echo `echo test", BashFormat())
	// Backtick detection: SubstitutionDepth should be > 0
	// (exact value may vary, but should be at least 1)
	if ctx.SubstitutionDepth < 1 {
		t.Errorf("SubstitutionDepth = %v, want >= 1", ctx.SubstitutionDepth)
	}
	if ctx.SubstitutionKind != SUBSTITUTION_BACKTICK {
		t.Errorf("SubstitutionKind = %v, want SUBSTITUTION_BACKTICK", ctx.SubstitutionKind)
	}
}

func TestSubstitution_ElvishOutputCapture(t *testing.T) {
	// echo (echo test) → two words in elvish
	tokens, err := SplitWith("echo (echo test)", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "(echo test)" {
		t.Errorf("Words = %v, want [echo (echo test)]", words)
	}
}

func TestSubstitution_ElvishOutputCaptureCompletion(t *testing.T) {
	// Cursor inside (ls → inner context: Words=["ls"], depth=1
	ctx := SplitForCompletion("echo (ls", ElvishFormat())
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %v, want 1", ctx.SubstitutionDepth)
	}
	if len(ctx.Words) != 1 || ctx.Words[0] != "ls" {
		t.Errorf("Words = %v, want [ls]", ctx.Words)
	}
}

func TestSubstitution_TokenReclassification(t *testing.T) {
	// Verify that ( and ) are reclassified as substitution delimiters
	tokens, err := SplitWith("echo $(test)", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN {
			switch tok.Value {
			case "$(":
				if tok.WordbreakType != WORDBREAK_SUBSTITUTION_OPEN {
					t.Errorf("Token %q: WordbreakType = %v, want WORDBREAK_SUBSTITUTION_OPEN", tok.Value, tok.WordbreakType)
				}
			case ")":
				if tok.WordbreakType != WORDBREAK_SUBSTITUTION_CLOSE {
					t.Errorf("Token %q: WordbreakType = %v, want WORDBREAK_SUBSTITUTION_CLOSE", tok.Value, tok.WordbreakType)
				}
			}
		}
	}
}

func TestSubstitution_SubstitutionScopes(t *testing.T) {
	// echo $(echo $(echo test)) → two scopes, both closed
	tokens, err := SplitWith("echo $(echo $(echo test))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scopes := tokens.SubstitutionScopes()
	closed := 0
	for _, s := range scopes {
		if s.CloseIndex >= 0 {
			closed++
		}
	}
	if closed != 2 {
		t.Errorf("closed scopes = %d, want 2", closed)
	}
}

func TestSubstitution_SubstitutionScopesUnclosed(t *testing.T) {
	// echo $(echo $(git ch → two scopes, both unclosed
	tokens, err := SplitWith("echo $(echo $(git ch", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scopes := tokens.SubstitutionScopes()
	unclosed := 0
	for _, s := range scopes {
		if s.CloseIndex == -1 && s.OpenIndex >= 0 {
			unclosed++
		}
	}
	if unclosed != 2 {
		t.Errorf("unclosed scopes = %d, want 2", unclosed)
	}
}
