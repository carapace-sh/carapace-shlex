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
	// Cmd: ^ is LITERAL inside double quotes — it does not escape.
	// "say ^" → ^ is literal, " closes the quote.
	// Outside quotes, ^" → literal " (caret escapes).
	tokens, err := SplitWith(`echo "say ^"hello^""`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	// "say ^"  → quote contains "say ^", then " closes quote
	// hello    → bareword outside quotes
	// ^"       → caret escapes the quote → literal "
	// "        → this final quote opens a new quoted region (unterminated)
	// Words() merges adjacent tokens, so the whole thing is one word.
	if last.Value != `say ^hello"` {
		t.Errorf("cmd caret in quotes: Value = %q, want %q", last.Value, `say ^hello"`)
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

func TestCmdFormat_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("cmd ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestCmdFormat_Redirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo > bar", CmdFormat())
	if !ctx.IsRedirect {
		t.Errorf("cmd redirect: IsRedirect = false, want true")
	}
	if ctx.CurrentWord != "bar" {
		t.Errorf("cmd redirect: CurrentWord = %q, want %q", ctx.CurrentWord, "bar")
	}
}

func TestCmdFormat_OpenDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hel`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("cmd open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}

func TestCmdFormat_CaretAtEOF(t *testing.T) {
	tokens, err := SplitWith("echo foo^", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != ESCAPING_STATE {
		t.Errorf("cmd caret EOF: State = %v, want ESCAPING_STATE", last.State)
	}
	if last.Value != "foo" {
		t.Errorf("cmd caret EOF: Value = %q, want %q", last.Value, "foo")
	}
}

func TestCmdFormat_CaretLiteralInQuotes(t *testing.T) {
	// Cmd: ^ is literal inside double quotes — does not escape the next char.
	// "hello^world" should produce hello^world, not helloworld.
	tokens, err := SplitWith(`echo "hello^world"`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello^world" {
		t.Errorf("cmd caret literal in quotes: Words = %v, want [echo hello^world]", words)
	}
}

func TestCmdFormat_DoubleCaretLiteralInQuotes(t *testing.T) {
	// Cmd: ^^ inside quotes is literal ^^ (both carets), not a single ^.
	tokens, err := SplitWith(`echo "hello^^world"`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello^^world" {
		t.Errorf("cmd double caret in quotes: Words = %v, want [echo hello^^world]", words)
	}
}

func TestCmdFormat_LineContinuation(t *testing.T) {
	// Cmd: ^ at end of line is a line continuation — ^\n is consumed
	tokens, err := SplitWith("echo foo^\nbar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foobar" {
		t.Errorf("cmd line continuation: Words = %v, want [echo foobar]", words)
	}
}

func TestCmdFormat_LineContinuationCRLF(t *testing.T) {
	// Cmd: ^ at end of line with CRLF is a line continuation
	tokens, err := SplitWith("echo foo^\r\nbar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foobar" {
		t.Errorf("cmd line continuation CRLF: Words = %v, want [echo foobar]", words)
	}
}

func TestCmdFormat_ParenGrouping(t *testing.T) {
	// Cmd: ( and ) are grouping operators
	tokens, err := SplitWith("(echo foo) & echo bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("cmd parens: %d pipelines, want 2", len(pipelines))
	}
}

func TestCmdFormat_ParenBeforeCommand(t *testing.T) {
	// Cmd: ( and ) are wordbreak operators; with spaces they separate from words
	// They are not redirect operators, so FilterRedirects keeps them.
	// Words() does not merge non-adjacent tokens.
	tokens, err := SplitWith("( echo hello )", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipeline := tokens.CurrentPipeline()
	words := pipeline.Words().Strings()
	if len(words) != 4 || words[0] != "(" || words[1] != "echo" || words[2] != "hello" || words[3] != ")" {
		t.Errorf("cmd paren before cmd: Words = %v, want [( echo hello )]", words)
	}
}

func TestCmdFormat_CommaDelimiter(t *testing.T) {
	// Cmd: comma is a word delimiter (like space)
	tokens, err := SplitWith("echo hello,world", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "hello" || words[2] != "world" {
		t.Errorf("cmd comma: Words = %v, want [echo hello world]", words)
	}
}

func TestCmdFormat_CommaInQuotes(t *testing.T) {
	// Cmd: comma inside double quotes is literal
	tokens, err := SplitWith(`echo "hello,world"`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello,world" {
		t.Errorf("cmd comma in quotes: Words = %v, want [echo hello,world]", words)
	}
}

func TestCmdFormat_StreamRedirect2(t *testing.T) {
	// Cmd: 2> should be recognized as a stream redirect (stderr)
	tokens, err := SplitWith("echo foo 2> bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("cmd 2>: %d pipelines, want 1", len(pipelines))
	}
	// The 2> should be a redirect, so filtered words should not include "2"
	filtered := pipelines[0].FilterRedirects().Words().Strings()
	if len(filtered) != 2 || filtered[0] != "echo" || filtered[1] != "foo" {
		t.Errorf("cmd 2> filtered: Words = %v, want [echo foo]", filtered)
	}
}

func TestCmdFormat_StreamRedirectMerge(t *testing.T) {
	// Cmd: 2>&1 should be recognized as a stream merge redirect
	tokens, err := SplitWith("echo foo 2>&1 bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("cmd 2>&1: %d pipelines, want 1", len(pipelines))
	}
	filtered := pipelines[0].FilterRedirects().Words().Strings()
	if len(filtered) != 2 || filtered[0] != "echo" || filtered[1] != "foo" {
		t.Errorf("cmd 2>&1 filtered: Words = %v, want [echo foo]", filtered)
	}
}

func TestCmdFormat_StreamRedirectCompletion(t *testing.T) {
	// Cmd: completing after 2> should detect redirect
	ctx := SplitForCompletion("echo foo 2> bar", CmdFormat())
	if !ctx.IsRedirect {
		t.Errorf("cmd 2> completion: IsRedirect = false, want true")
	}
	if ctx.CurrentWord != "bar" {
		t.Errorf("cmd 2> completion: CurrentWord = %q, want %q", ctx.CurrentWord, "bar")
	}
}

func TestCmdFormat_CaretLineContinuationAtEOF(t *testing.T) {
	// Cmd: ^ at EOF (no newline) should enter ESCAPING_STATE, not line continuation
	tokens, err := SplitWith("echo foo^", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != ESCAPING_STATE {
		t.Errorf("cmd caret EOF: State = %v, want ESCAPING_STATE", last.State)
	}
}
