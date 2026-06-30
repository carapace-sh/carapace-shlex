package shlex

import "testing"

func TestXonshFormat_SingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hello world'", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("xonsh single: Words = %v, want [echo hello world]", words)
	}
}

func TestXonshFormat_RawString(t *testing.T) {
	// r'...' — r prefix + single quote, Words() merges
	// \ inside single quotes is literal (NonEscapingQuoteEscapes is false)
	tokens, err := SplitWith(`echo r'C:\path'`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != `rC:\path` {
		t.Errorf("xonsh r'': Words = %v, want [echo rC:\\path]", words)
	}
}

func TestXonshFormat_RawDoubleQuoted(t *testing.T) {
	// r"..." — r prefix merges with double-quoted segment.
	// Note: \ inside double quotes is consumed as escape (ESCAPING_QUOTED_STATE).
	// The r prefix is a word char and doesn't change quote behavior in the lexer.
	tokens, err := SplitWith(`echo r"C:\path"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// \p is consumed as escape → "p" emitted, backslash dropped
	if len(words) != 2 || words[1] != "rC:path" {
		t.Errorf("xonsh r\"\": Words = %v, want [echo rC:path]", words)
	}
}

func TestXonshFormat_DoubleQuoteEscape(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `say "hello"` {
		t.Errorf("xonsh double escape: Value = %q, want %q", last.Value, `say "hello"`)
	}
}

func TestXonshFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("xonsh pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestXonshFormat_Background(t *testing.T) {
	// Xonsh uses & for background (POSIX-like)
	tokens, err := SplitWith("echo foo &", XonshFormat())
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
		t.Errorf("xonsh &: Type=%v WordbreakType=%v, want WORDBREAK_TOKEN/LIST_ASYNC", amp.Type, amp.WordbreakType)
	}
}

func TestXonshFormat_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshFormat_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("xonsh open single: State = %v, want QUOTING_STATE", last.State)
	}
}
