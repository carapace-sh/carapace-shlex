package shlex

import "testing"

func TestSubstitution_BashCommandSubstitution(t *testing.T) {
	tokens, err := Split("echo $(echo test)", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.wordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$(echo test)" {
		t.Errorf("Words = %v, want [echo $(echo test)]", words)
	}
}

func TestSubstitution_BashArithmetic(t *testing.T) {
	tokens, err := Split("echo $((1+2))", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.wordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$((1+2))" {
		t.Errorf("Words = %v, want [echo $((1+2))]", words)
	}
}

func TestSubstitution_BashProcessSubstitution(t *testing.T) {
	tokens, err := Split("echo <(grep foo)", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.wordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "<(grep foo)" {
		t.Errorf("Words = %v, want [echo <(grep foo)]", words)
	}
}

func TestSubstitution_PipelineDoesNotSplitInsideSubstitution(t *testing.T) {
	tokens, err := Split("echo foo $(bar | grep x) baz", Bash)
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.pipelines()
	if len(pipelines) != 1 {
		t.Errorf("Pipelines = %d, want 1", len(pipelines))
	}
	words := pipelines[0].wordsWithSubstitutions().Strings()
	if len(words) != 4 || words[0] != "echo" || words[1] != "foo" ||
		words[2] != "$(bar | grep x)" || words[3] != "baz" {
		t.Errorf("Words = %v, want [echo foo $(bar | grep x) baz]", words)
	}
}

func TestSubstitution_NestedCommandSubstitution(t *testing.T) {
	tokens, err := Split("echo $(echo $(echo test))", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.wordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$(echo $(echo test))" {
		t.Errorf("Words = %v, want [echo $(echo $(echo test))]", words)
	}
}

func TestSubstitution_CompletionInsideSubstitution(t *testing.T) {
	ctx := Complete("echo $(git ch", Bash)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.CurrentWord != "ch" {
		t.Errorf("CurrentWord = %q, want \"ch\"", ctx.CurrentWord)
	}
}

func TestSubstitution_CompletionInsideNestedSubstitution(t *testing.T) {
	ctx := Complete("echo $(echo $(git ch", Bash)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
}

func TestSubstitution_CompletionInsideArithmetic(t *testing.T) {
	ctx := Complete("echo $((1+2", Bash)
	if len(ctx.Words) != 1 || ctx.Words[0] != "echo" {
		t.Errorf("Words = %v, want [echo]", ctx.Words)
	}
}

func TestSubstitution_CompletionClosedSubstitution(t *testing.T) {
	ctx := Complete("echo $(echo test)", Bash)
	if len(ctx.Words) != 2 || ctx.Words[0] != "echo" || ctx.Words[1] != "$(echo test)" {
		t.Errorf("Words = %v, want [echo $(echo test)]", ctx.Words)
	}
}

func TestSubstitution_CompletionInsideSubstitutionWithInnerPipe(t *testing.T) {
	ctx := Complete("echo foo $(bar | grep x", Bash)
	if len(ctx.Words) != 2 || ctx.Words[0] != "grep" || ctx.Words[1] != "x" {
		t.Errorf("Words = %v, want [grep x]", ctx.Words)
	}
	if ctx.CurrentWord != "x" {
		t.Errorf("CurrentWord = %q, want \"x\"", ctx.CurrentWord)
	}
}

func TestSubstitution_BashBacktickSubstitution(t *testing.T) {
	tokens, err := Split("echo `echo test`", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.wordsWithSubstitutions().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "`echo" || words[2] != "test`" {
		t.Errorf("Words = %v, want [echo `echo test`]", words)
	}
}

func TestSubstitution_ElvishOutputCapture(t *testing.T) {
	tokens, err := Split("echo (echo test)", Elvish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.wordsWithSubstitutions().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "(echo test)" {
		t.Errorf("Words = %v, want [echo (echo test)]", words)
	}
}

func TestSubstitution_ElvishOutputCaptureCompletion(t *testing.T) {
	ctx := Complete("echo (ls", Elvish)
	if len(ctx.Words) != 1 || ctx.Words[0] != "ls" {
		t.Errorf("Words = %v, want [ls]", ctx.Words)
	}
}

func TestSubstitution_TokenReclassification(t *testing.T) {
	tokens, err := Split("echo $(test)", Bash)
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

func TestSubstitution_CompletionSubstitutionDepth(t *testing.T) {
	ctx := Complete("echo $(git ch", Bash)
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}

	ctx = Complete("echo $(echo $(git ch", Bash)
	if ctx.SubstitutionDepth != 2 {
		t.Errorf("SubstitutionDepth = %d, want 2", ctx.SubstitutionDepth)
	}

	ctx = Complete("echo test", Bash)
	if ctx.SubstitutionDepth != 0 {
		t.Errorf("SubstitutionDepth = %d, want 0", ctx.SubstitutionDepth)
	}

	ctx = Complete("echo $(echo test)", Bash)
	if ctx.SubstitutionDepth != 0 {
		t.Errorf("SubstitutionDepth = %d, want 0 (closed)", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_CompletionRedirectInsideSubstitution(t *testing.T) {
	ctx := Complete("echo $(cat >", Bash)
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
	if !ctx.IsRedirect {
		t.Errorf("IsRedirect = false, want true")
	}
}

func TestSubstitution_ZshCommandSubstitution(t *testing.T) {
	ctx := Complete("echo $(git ch", Zsh)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_FishCommandSubstitution(t *testing.T) {
	ctx := Complete("echo (git ch", Fish)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_NushellSubexpression(t *testing.T) {
	ctx := Complete("echo (git ch", Nushell)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_PowerShellSubexpression(t *testing.T) {
	ctx := Complete("echo $(git ch", Powershell)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_ProcessSubstitution(t *testing.T) {
	ctx := Complete("echo <(git ch", Bash)
	if len(ctx.Words) != 2 || ctx.Words[0] != "git" || ctx.Words[1] != "ch" {
		t.Errorf("Words = %v, want [git ch]", ctx.Words)
	}
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_ClosedSubstitutionDoesNotAffectPrefix(t *testing.T) {
	ctx := Complete("echo $(echo test) foo", Bash)
	if ctx.SubstitutionDepth != 0 {
		t.Errorf("SubstitutionDepth = %d, want 0", ctx.SubstitutionDepth)
	}
	if len(ctx.Words) != 3 || ctx.Words[0] != "echo" || ctx.Words[1] != "$(echo test)" || ctx.Words[2] != "foo" {
		t.Errorf("Words = %v, want [echo $(echo test) foo]", ctx.Words)
	}
}
