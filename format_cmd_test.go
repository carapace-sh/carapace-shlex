package shlex

import "testing"

func TestCmdFormat_DoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hello world"`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("cmd double: Words = %v, want [echo hello world]", words)
	}
}

func TestCmdFormat_NoSingleQuote(t *testing.T) {
	// Cmd: ' is a literal character, not a quote
	tokens, err := SplitWith("echo 'hello'", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "'hello'" {
		t.Errorf("cmd no single: Words = %v, want [echo 'hello']", words)
	}
}

func TestCmdFormat_CaretEscape(t *testing.T) {
	// Cmd: ^ escapes the next character
	tokens, err := SplitWith("echo hello^&world", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello&world" {
		t.Errorf("cmd caret: Words = %v, want [echo hello&world]", words)
	}
}

func TestCmdFormat_CaretEscapePipe(t *testing.T) {
	tokens, err := SplitWith("echo ^|", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "|" {
		t.Errorf("cmd caret pipe: Words = %v, want [echo |]", words)
	}
}

func TestCmdFormat_BackslashLiteral(t *testing.T) {
	// Cmd: \ is a literal character (Windows paths)
	tokens, err := SplitWith(`echo C:\path\to\file`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != `C:\path\to\file` {
		t.Errorf("cmd backslash: Words = %v, want [echo C:\\path\\to\\file]", words)
	}
}

func TestCmdFormat_AmpSeparator(t *testing.T) {
	// Cmd: & is a command separator (like ; in POSIX)
	tokens, err := SplitWith("echo foo & echo bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("cmd &: %d pipelines, want 2", len(pipelines))
	}
}

func TestCmdFormat_NoSemicolonSeparator(t *testing.T) {
	// Cmd: ; is NOT a separator — it's a literal character
	tokens, err := SplitWith("echo foo;bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo;bar" {
		t.Errorf("cmd ;: Words = %v, want [echo foo;bar]", words)
	}
}

func TestCmdFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | findstr bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("cmd pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestCmdFormat_DoubleAnd(t *testing.T) {
	// Cmd: && is conditional and
	tokens, err := SplitWith("echo foo && echo bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("cmd &&: %d pipelines, want 2", len(pipelines))
	}
}

func TestCmdFormat_CaretInQuotes(t *testing.T) {
	// Cmd: ^ escapes inside double quotes (^" → literal ")
	tokens, err := SplitWith(`echo "say ^"hello^""`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `say "hello"` {
		t.Errorf("cmd caret in quotes: Value = %q, want %q", last.Value, `say "hello"`)
	}
}

func TestCmdFormat_PercentNotWordbreak(t *testing.T) {
	// Cmd: % is a word character (variable expansion), not a word break
	tokens, err := SplitWith("echo %PATH%", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "%PATH%" {
		t.Errorf("cmd %%: Words = %v, want [echo %%PATH%%]", words)
	}
}
