package shlex

import "testing"

func TestElvishFormat_BarewordBackslash(t *testing.T) {
	// Elvish: \ is a bareword character outside quotes (not an escape)
	tokens, err := SplitWith(`echo C:\path`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != `C:\path` {
		t.Errorf("elvish bareword \\: Words = %v, want [echo C:\\path]", words)
	}
}

func TestElvishFormat_DoubleQuoteEscape(t *testing.T) {
	// Elvish: \ IS an escape inside double quotes
	tokens, err := SplitWith(`echo "hello\nworld"`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != IN_WORD_STATE {
		t.Errorf("elvish double-quote escape: State = %v, want IN_WORD_STATE", last.State)
	}
}

func TestElvishFormat_DoubledQuoteSplit(t *testing.T) {
	tokens, err := SplitWith("echo 'it''s a test'", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's a test" {
		t.Errorf("elvish '' split: Words = %v, want [echo it's a test]", words)
	}
}

func TestElvishFormat_DoubleQuoteValue(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("elvish double value: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestElvishFormat_AmpNotListOperator(t *testing.T) {
	// & is for map literals in elvish, not a list operator
	tokens, err := SplitWith("echo foo & echo bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("elvish &: %d pipelines, want 1 (& is not a separator)", len(pipelines))
	}
}

func TestElvishFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("elvish pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestElvishFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("elvish semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestElvishFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("elvish open single: State = %v, want QUOTING_STATE", last.State)
	}
}
