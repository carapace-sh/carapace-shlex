package shlex

import "testing"

func TestTcshFormat(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("TcshFormat: %d pipelines, want 2", len(pipelines))
	}
}

func TestTcshFormat_BackslashQuote(t *testing.T) {
	tokens, err := SplitWith("echo $'hello'", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$hello" {
		t.Errorf("TcshFormat $'': Words = %v, want [echo $hello]", words)
	}
}

func TestTcshFormat_SingleQuoteLiteral(t *testing.T) {
	tokens, err := SplitWith("echo '$HOME'", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("tcsh single literal: Words = %v, want [echo $HOME]", words)
	}
}

func TestTcshFormat_BacktickLiteralInSingleQuotes(t *testing.T) {
	tokens, err := SplitWith("echo '`cmd`'", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "`cmd`" {
		t.Errorf("tcsh backtick in single: Words = %v, want [echo `cmd`]", words)
	}
}

func TestTcshFormat_EscapedDoubleQuoteOutside(t *testing.T) {
	tokens, err := SplitWith(`echo \"hello\"`, TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `"hello"` {
		t.Errorf("tcsh escaped double: Words = %v, want [echo \"hello\"]", words)
	}
}

func TestTcshFormat_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshFormat_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshFormat_Background(t *testing.T) {
	tokens, err := SplitWith("echo foo &", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var amp Token
	for _, tok := range tokens {
		if tok.RawValue == "&" {
			amp = tok
		}
	}
	if amp.Type != WORDBREAK_TOKEN || amp.WordbreakType != WORDBREAK_LIST_ASYNC {
		t.Errorf("tcsh &: Type=%v WT=%v, want WORDBREAK_TOKEN/LIST_ASYNC", amp.Type, amp.WordbreakType)
	}
}

func TestTcshFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("tcsh open single: State = %v, want QUOTING_STATE", last.State)
	}
}
