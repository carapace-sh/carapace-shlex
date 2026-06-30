package shlex

import "testing"

func TestFishFormat_SingleQuoteEscape(t *testing.T) {
	// Fish: \' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it\\'s'", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "it's" {
		t.Errorf("fish \\' in single quotes: Value = %q, want %q", last.Value, "it's")
	}
	if last.State != IN_WORD_STATE {
		t.Errorf("fish \\' in single quotes: State = %v, want IN_WORD_STATE", last.State)
	}
}

func TestFishFormat_KeywordOperators(t *testing.T) {
	// Fish: "and" and "or" are keyword operators that split pipelines
	tokens, err := SplitWith("echo foo and echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish keyword operators: %d pipelines, want 2 (and splits)", len(pipelines))
	}
}

func TestFishFormat_KeywordOperatorAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo and", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	// "and" should be a WORDBREAK_TOKEN
	last := tokens[len(tokens)-1]
	if last.Type != WORDBREAK_TOKEN {
		t.Errorf("fish 'and': Type = %v, want WORDBREAK_TOKEN", last.Type)
	}
	if last.WordbreakType != WORDBREAK_LIST_AND {
		t.Errorf("fish 'and': WordbreakType = %v, want WORDBREAK_LIST_AND", last.WordbreakType)
	}
}

func TestElvishFormat_DoubledQuote(t *testing.T) {
	// Elvish: '' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it''s'", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "it's" {
		t.Errorf("elvish '': Value = %q, want %q", last.Value, "it's")
	}
}

func TestElvishFormat_NoListOperators(t *testing.T) {
	// Elvish has no && or || — they're regular characters (word chars)
	tokens, err := SplitWith("echo foo", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("elvish: Words = %v, want [echo foo]", words)
	}
}

func TestElvishFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("elvish pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestPowershellFormat_BacktickEscape(t *testing.T) {
	// PowerShell: backtick is the escape character
	tokens, err := SplitWith("echo `\"hello`\"", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	// The backtick escapes the " inside the double-quoted string
	// Token should contain "hello" (with the quotes as part of the word)
	if last.Value != `"hello"` {
		t.Errorf("powershell backtick: Value = %q, want %q", last.Value, `"hello"`)
	}
}

func TestPowershellFormat_DoubledSingleQuote(t *testing.T) {
	// PowerShell: '' inside single quotes → literal '
	tokens, err := SplitWith("echo 'don''t'", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "don't" {
		t.Errorf("powershell '': Value = %q, want %q", last.Value, "don't")
	}
}

func TestPowershellFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("Get-Process | Select-Object Name", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("powershell pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestIonFormat_StderrPipe(t *testing.T) {
	// Ion: ^| is stderr pipe (ion-unique)
	tokens, err := SplitWith("echo foo ^| grep bar", IonFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("ion ^|: %d pipelines, want 2", len(pipelines))
	}
}

func TestIonFormat_CombinedPipe(t *testing.T) {
	// Ion: &| is stdout+stderr pipe (ion-unique)
	tokens, err := SplitWith("echo foo &| grep bar", IonFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("ion &|: %d pipelines, want 2", len(pipelines))
	}
}

func TestIonFormat_AtNotWordbreak(t *testing.T) {
	// Ion: @ is an array sigil, not a wordbreak
	tokens, err := SplitWith("echo @items", IonFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "@items" {
		t.Errorf("ion @: Words = %v, want [echo @items]", words)
	}
}

func TestIonFormat_StderrRedirect(t *testing.T) {
	// Ion: ^> redirects stderr to file
	tokens, err := SplitWith("echo foo ^> errors.log", IonFormat())
	if err != nil {
		t.Fatal(err)
	}
	// The ^> should be classified as a redirect
	found := false
	for _, tok := range tokens {
		if tok.RawValue == "^>" && tok.WordbreakType.IsRedirect() {
			found = true
		}
	}
	if !found {
		t.Errorf("ion ^>: redirect token not found")
	}
}
