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

func TestFishFormat_SingleQuoteBackslashEscape(t *testing.T) {
	// Fish: \\ inside single quotes → literal \
	tokens, err := SplitWith("echo 'C:\\\\path'", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `C:\path` {
		t.Errorf("fish \\\\ in single quotes: Value = %q, want %q", last.Value, `C:\path`)
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

func TestFishFormat_KeywordOperatorOr(t *testing.T) {
	tokens, err := SplitWith("echo foo or", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens[len(tokens)-1]
	if last.Type != WORDBREAK_TOKEN {
		t.Errorf("fish 'or': Type = %v, want WORDBREAK_TOKEN", last.Type)
	}
	if last.WordbreakType != WORDBREAK_LIST_OR {
		t.Errorf("fish 'or': WordbreakType = %v, want WORDBREAK_LIST_OR", last.WordbreakType)
	}
}

func TestFishFormat_NotNotKeywordOperator(t *testing.T) {
	// "not" is a prefix keyword but NOT a pipeline delimiter
	tokens, err := SplitWith("echo foo not", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens[len(tokens)-1]
	if last.Type != WORD_TOKEN {
		t.Errorf("fish 'not': Type = %v, want WORD_TOKEN (not a delimiter)", last.Type)
	}
}

func TestFishFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != QUOTING_STATE {
		t.Errorf("fish open quote: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestFishFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish semicolon: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_DoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hello world"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("fish double: Words = %v, want [echo hello world]", words)
	}
}

func TestFishFormat_CompletionContext(t *testing.T) {
	ctx := SplitForCompletion("echo foo and grep hel", FishFormat())
	if ctx.CurrentWord != "hel" {
		t.Errorf("fish completion: CurrentWord = %q, want %q", ctx.CurrentWord, "hel")
	}
	if len(ctx.Words) != 2 {
		t.Errorf("fish completion: Words = %v, want 2 words (grep hel)", ctx.Words)
	}
}
