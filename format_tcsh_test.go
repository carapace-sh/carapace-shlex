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

func TestTcshFormat_ForceOutputBang(t *testing.T) {
	tokens, err := SplitWith("echo foo >! /tmp/bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == ">!" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_OUTPUT_FORCE_BANG {
		t.Errorf("tcsh >!: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_OUTPUT_FORCE_BANG", op.Type, op.WordbreakType)
	}
}

func TestTcshFormat_ForceAppendBang(t *testing.T) {
	tokens, err := SplitWith("echo foo >>! /tmp/bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == ">>!" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE_BANG {
		t.Errorf("tcsh >>!: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_OUTPUT_APPEND_FORCE_BANG", op.Type, op.WordbreakType)
	}
}

func TestTcshFormat_RedirectBothStdoutStderr(t *testing.T) {
	tokens, err := SplitWith("echo foo >& /tmp/bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == ">&" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_OUTPUT_BOTH {
		t.Errorf("tcsh >&: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_OUTPUT_BOTH", op.Type, op.WordbreakType)
	}
}

func TestTcshFormat_PipeWithStderr(t *testing.T) {
	tokens, err := SplitWith("echo foo |& grep bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "|&" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_PIPE_WITH_STDERR {
		t.Errorf("tcsh |&: Type=%v WT=%v, want WORDBREAK_TOKEN/PIPE_WITH_STDERR", op.Type, op.WordbreakType)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh |&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshFormat_HereDoc(t *testing.T) {
	tokens, err := SplitWith("cat << EOF", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "<<" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_HERE_DOC {
		t.Errorf("tcsh <<: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_HERE_DOC", op.Type, op.WordbreakType)
	}
}

func TestTcshFormat_InputDuplicate(t *testing.T) {
	tokens, err := SplitWith("cmd <& 0", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "<&" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_INPUT_DUPLICATE {
		t.Errorf("tcsh <&: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_INPUT_DUPLICATE", op.Type, op.WordbreakType)
	}
}

func TestTcshFormat_EqualsNotWordbreak(t *testing.T) {
	tokens, err := SplitWith("set foo=bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo=bar" {
		t.Errorf("tcsh = not wordbreak: Words = %v, want [set foo=bar]", words)
	}
}

func TestTcshFormat_AtNotWordbreak(t *testing.T) {
	tokens, err := SplitWith("echo @foo", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "@foo" {
		t.Errorf("tcsh @ not wordbreak: Words = %v, want [echo @foo]", words)
	}
}

func TestTcshFormat_BangIsWordbreak(t *testing.T) {
	tokens, err := SplitWith("echo foo >!bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == ">!" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN {
		t.Errorf("tcsh >!bar: expected >! to be WORDBREAK_TOKEN, got Type=%v", op.Type)
	}
}

func TestTcshFormat_NoBashPipeForceOperator(t *testing.T) {
	tokens, err := SplitWith("echo foo >| /tmp/bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && (tok.WordbreakType == WORDBREAK_REDIRECT_OUTPUT_FORCE || tok.WordbreakType == WORDBREAK_PIPE) {
			op = tok
		}
	}
	if op.Type == WORDBREAK_TOKEN && op.WordbreakType == WORDBREAK_REDIRECT_OUTPUT_FORCE {
		t.Errorf("tcsh >|: should not be classified as bash REDIRECT_OUTPUT_FORCE")
	}
}

func TestTcshFormat_NoHereStringOperator(t *testing.T) {
	tokens, err := SplitWith("cmd <<< foo", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var foundHereString bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_REDIRECT_INPUT_STRING {
			foundHereString = true
		}
	}
	if foundHereString {
		t.Errorf("tcsh <<<: should not be classified as REDIRECT_INPUT_STRING (tcsh has no here-string)")
	}
}

func TestTcshFormat_NoBashBothRedirect(t *testing.T) {
	tokens, err := SplitWith("cmd &> /tmp/out", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var foundBashBoth bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_REDIRECT_OUTPUT_BOTH && tok.RawValue == "&>" {
			foundBashBoth = true
		}
	}
	if foundBashBoth {
		t.Errorf("tcsh &>: should not be classified as REDIRECT_OUTPUT_BOTH (tcsh uses >&)")
	}
}
