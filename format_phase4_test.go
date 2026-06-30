package shlex

import "testing"

func TestNushellFormat_SingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hello world'", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("nushell single: Words = %v, want [echo hello world]", words)
	}
}

func TestNushellFormat_DoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hello\nworld"`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "hellonworld" {
		t.Errorf("nushell double: Value = %q, want %q", last.Value, "hellonworld")
	}
}

func TestNushellFormat_BacktickQuote(t *testing.T) {
	// Nushell: backtick is a quote char (not escape)
	tokens, err := SplitWith("echo `hello world`", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("nushell backtick: Words = %v, want [echo hello world]", words)
	}
}

func TestNushellFormat_InterpolatedPrefix(t *testing.T) {
	// $'...' — $ prefix + single quote, Words() merges
	tokens, err := SplitWith("echo $'hello'", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$hello" {
		t.Errorf("nushell $': Words = %v, want [echo $hello]", words)
	}
}

func TestNushellFormat_OpenBacktick(t *testing.T) {
	tokens, err := SplitWith("echo `hel", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != QUOTING_STATE {
		t.Errorf("nushell open backtick: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestNushellFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("nushell pipe: %d pipelines, want 2", len(pipelines))
	}
}

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
	// Find the & token (not the trailing empty word)
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
