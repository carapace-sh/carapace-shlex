package shlex

import "testing"

func TestSubstitution_BashCommandSubstitution(t *testing.T) {
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
	ctx := SplitForCompletion("echo $(git ch", BashFormat())
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.CurrentWord != "ch" {
		t.Errorf("CurrentWord = %q, want \"ch\"", ctx.CurrentWord)
	}
}

func TestSubstitution_CompletionInsideNestedSubstitution(t *testing.T) {
	ctx := SplitForCompletion("echo $(echo $(git ch", BashFormat())
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
}

func TestSubstitution_CompletionInsideArithmetic(t *testing.T) {
	ctx := SplitForCompletion("echo $((1+2", BashFormat())
	if len(ctx.Words) != 1 || ctx.Words[0] != "echo" {
		t.Errorf("Words = %v, want [echo]", ctx.Words)
	}
}

func TestSubstitution_CompletionClosedSubstitution(t *testing.T) {
	ctx := SplitForCompletion("echo $(echo test)", BashFormat())
	if len(ctx.Words) != 2 || ctx.Words[0] != "echo" || ctx.Words[1] != "$(echo test)" {
		t.Errorf("Words = %v, want [echo $(echo test)]", ctx.Words)
	}
}

func TestSubstitution_CompletionInsideSubstitutionWithInnerPipe(t *testing.T) {
	ctx := SplitForCompletion("echo foo $(bar | grep x", BashFormat())
	if len(ctx.Words) != 2 || ctx.Words[0] != "grep" || ctx.Words[1] != "x" {
		t.Errorf("Words = %v, want [grep x]", ctx.Words)
	}
	if ctx.CurrentWord != "x" {
		t.Errorf("CurrentWord = %q, want \"x\"", ctx.CurrentWord)
	}
}

func TestSubstitution_BashBacktickSubstitution(t *testing.T) {
	tokens, err := SplitWith("echo `echo test`", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "`echo" || words[2] != "test`" {
		t.Errorf("Words = %v, want [echo `echo test`]", words)
	}
}

func TestSubstitution_ElvishOutputCapture(t *testing.T) {
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
	ctx := SplitForCompletion("echo (ls", ElvishFormat())
	if len(ctx.Words) != 1 || ctx.Words[0] != "ls" {
		t.Errorf("Words = %v, want [ls]", ctx.Words)
	}
}

func TestSubstitution_TokenReclassification(t *testing.T) {
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
