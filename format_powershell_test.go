package shlex

import "testing"

func TestPowershellFormat_DoubledDoubleQuote(t *testing.T) {
	// PowerShell: "" inside double quotes → literal "
	tokens, err := SplitWith(`echo "say ""hello"""`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `say "hello"` {
		t.Errorf(`powershell "": Value = %q, want %q`, last.Value, `say "hello"`)
	}
}

func TestPowershellFormat_BacktickNotBackslash(t *testing.T) {
	// PowerShell: \ is NOT an escape (backtick is). So \ should be a literal word char.
	tokens, err := SplitWith(`echo C:\path`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != `C:\path` {
		t.Errorf("powershell \\ literal: Words = %v, want [echo C:\\path]", words)
	}
}

func TestPowershellFormat_DoubledSingleQuoteSplit(t *testing.T) {
	tokens, err := SplitWith("echo 'don''t'", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "don't" {
		t.Errorf("powershell '' split: Words = %v, want [echo don't]", words)
	}
}

func TestPowershellFormat_BacktickEscapeOutside(t *testing.T) {
	tokens, err := SplitWith("echo `$HOME", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("powershell backtick outside: Words = %v, want [echo $HOME]", words)
	}
}

func TestPowershellFormat_BacktickInDoubleQuotes(t *testing.T) {
	tokens, err := SplitWith("echo \"say `\"hello`\"\"", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("powershell backtick in double: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestPowershellFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("powershell pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestPowershellFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("powershell semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestPowershellFormat_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("powershell &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestPowershellFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("powershell open single: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestPowershellFormat_OpenDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hel`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("powershell open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}
