package shlex

import "testing"

func TestOilFormat_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilFormat_Background(t *testing.T) {
	tokens, err := SplitWith("echo foo & echo bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil &: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilFormat_PipeWithStderr(t *testing.T) {
	tokens, err := SplitWith("echo foo |& grep bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil |&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilFormat_RedirectOutput(t *testing.T) {
	tokens, err := SplitWith("echo foo > file.txt", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	ctx := SplitForCompletion("echo foo > file.txt", OilFormat())
	if !ctx.IsRedirect {
		t.Error("oil >: IsRedirect = false, want true")
	}
	_ = tokens
}

func TestOilFormat_RedirectAppend(t *testing.T) {
	ctx := SplitForCompletion("echo foo >> file.txt", OilFormat())
	if !ctx.IsRedirect {
		t.Error("oil >>: IsRedirect = false, want true")
	}
}

func TestOilFormat_HereDoc(t *testing.T) {
	tokens, err := SplitWith("cat << EOF", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_REDIRECT_HERE_DOC {
			found = true
		}
	}
	if !found {
		t.Error("oil <<: no WORDBREAK_REDIRECT_HERE_DOC token found")
	}
}

func TestOilFormat_HereString(t *testing.T) {
	tokens, err := SplitWith(`cat <<< "string"`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_REDIRECT_INPUT_STRING {
			found = true
		}
	}
	if !found {
		t.Error("oil <<<: no WORDBREAK_REDIRECT_INPUT_STRING token found")
	}
}

func TestOilFormat_CommandSubstitution(t *testing.T) {
	tokens, err := SplitWith("echo $(date)", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[1] != "$(date)" {
		t.Errorf("oil $(): Words = %v, want [echo $(date)]", words)
	}
}

func TestOilFormat_ArithmeticExpansion(t *testing.T) {
	tokens, err := SplitWith("echo $((1+2))", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[1] != "$((1+2))" {
		t.Errorf("oil $((): Words = %v, want [echo $((1+2))]", words)
	}
}

func TestOilFormat_ProcessSubstitution(t *testing.T) {
	tokens, err := SplitWith("echo <(grep foo)", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[1] != "<(grep foo)" {
		t.Errorf("oil <(): Words = %v, want [echo <(grep foo)]", words)
	}
}

func TestOilFormat_EscapedSpace(t *testing.T) {
	tokens, err := SplitWith(`echo foo\ bar`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo bar" {
		t.Errorf("oil escaped space: Words = %v, want [echo foo bar]", words)
	}
}

func TestOilFormat_DoubleQuoteBackslash(t *testing.T) {
	tokens, err := SplitWith(`echo "a\$b"`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a$b" {
		t.Errorf("oil double-quote backslash: Words = %v, want [echo a$b]", words)
	}
}

func TestOilFormat_LineContinuation(t *testing.T) {
	// backslash-newline = line continuation. The space before the
	// backslash is kept, so words are "echo" and "foo bar".
	tokens, err := SplitWith("echo foo \\\nbar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "foo" || words[2] != "bar" {
		t.Errorf("oil line continuation: Words = %v, want [echo foo bar]", words)
	}
}

func TestOilFormat_EmptyQuotes(t *testing.T) {
	tokens, err := SplitWith(`echo "" ''`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[1] != "" || words[2] != "" {
		t.Errorf("oil empty quotes: Words = %v, want [echo ]", words)
	}
}

func TestOilFormat_AdjacentQuotes(t *testing.T) {
	tokens, err := SplitWith(`echo a"b"c'd'`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "abcd" {
		t.Errorf("oil adjacent quotes: Words = %v, want [echo abcd]", words)
	}
}

func TestOilFormat_Comment(t *testing.T) {
	tokens, err := SplitWith("echo foo # comment", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("oil comment: Words = %v, want [echo foo]", words)
	}
}

func TestOilFormat_EscapeAtEOF(t *testing.T) {
	tokens, err := SplitWith(`echo foo\`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	if len(words) != 2 {
		t.Fatalf("oil escape EOF: Words = %v, want 2 words", words)
	}
	if words[1].State != ESCAPING_STATE {
		t.Errorf("oil escape EOF: State = %v, want ESCAPING_STATE", words[1].State)
	}
}
