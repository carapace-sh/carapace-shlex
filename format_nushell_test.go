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

func TestNushellFormat_InterpolatedDouble(t *testing.T) {
	tokens, err := SplitWith(`echo $"hello"`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$hello" {
		t.Errorf("nushell $\": Words = %v, want [echo $hello]", words)
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

func TestNushellFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("nushell semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestNushellFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("nushell open single: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestNushellFormat_OpenDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hel`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("nushell open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}
