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
	// The \n should be consumed as an escape (the next rune is literal)
	// The value should contain the literal characters after escape processing
	if last.State != IN_WORD_STATE {
		t.Errorf("elvish double-quote escape: State = %v, want IN_WORD_STATE", last.State)
	}
}

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
