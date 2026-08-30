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
	if last.Value != "hello\nworld" {
		t.Errorf("nushell double: Value = %q, want %q", last.Value, "hello\nworld")
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

// --- Escape sequence tests ---

func TestNushellFormat_EscapeSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"newline", `"hello\nworld"`, "hello\nworld"},
		{"tab", `"col1\tcol2"`, "col1\tcol2"},
		{"carriage_return", `"test\rmore"`, "test\rmore"},
		{"backslash", `"path\\file"`, "path\\file"},
		{"double_quote", `"say \"hello\""`, `say "hello"`},
		{"single_quote", `"it's \"quoted\""`, `it's "quoted"`},
		{"bell", `"x\ay"`, "x\a" + "y"},
		{"backspace", `"x\by"`, "x\by"},
		{"escape", `"x\ey"`, "x\x1by"},
		{"form_feed", `"x\fy"`, "x\fy"},
		{"null", `"x\0y"`, "x\x00y"},
		{"slash", `"C:\/Users"`, "C:/Users"},
		{"dollar", `"\$"`, "$"},
		{"caret", `"\^"`, "^"},
		{"hash", `"\#"`, "#"},
		{"pipe", `"\|"`, "|"},
		{"tilde", `"\~"`, "~"},
		{"paren_left", `"\("`, "("},
		{"paren_right", `"\)"`, ")"},
		{"brace_left", `"\{"`, "{"},
		{"brace_right", `"\}"`, "}"},
		{"unrecognized_keeps_backslash", `"\q"`, `\q`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := SplitWith("echo "+tc.input, NushellFormat())
			if err != nil {
				t.Fatal(err)
			}
			words := tokens.Words()
			last := words[len(words)-1]
			if last.Value != tc.want {
				t.Errorf("Value = %q, want %q", last.Value, tc.want)
			}
		})
	}
}

func TestNushellFormat_EscapeInSingleQuote(t *testing.T) {
	// Single-quoted strings have no escape processing
	tokens, err := SplitWith(`echo 'hello\nworld'`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("nushell single no-escape: Words = %v, want [echo hello\\nworld]", words)
	}
}

func TestNushellFormat_EscapeInBacktick(t *testing.T) {
	// Backtick strings have no escape processing
	tokens, err := SplitWith(`echo `+"`hello\\nworld`", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("nushell backtick no-escape: Words = %v, want [echo hello\\nworld]", words)
	}
}

func TestNushellFormat_OpenDoubleQuoteWithEscape(t *testing.T) {
	// Open double quote with escape at EOF — should stay in QUOTING_ESCAPING_STATE
	tokens, err := SplitWith(`echo "hello\`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != ESCAPING_QUOTED_STATE {
		t.Errorf("nushell open double with escape: State = %v, want ESCAPING_QUOTED_STATE", last.State)
	}
}

// --- Stream redirect operator tests ---

func TestNushellFormat_StreamRedirect_Out(t *testing.T) {
	tokens, err := SplitWith("cat foo out> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell out>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_StreamRedirect_Err(t *testing.T) {
	tokens, err := SplitWith("cat foo err> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell err>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_StreamRedirect_OE(t *testing.T) {
	tokens, err := SplitWith("cat foo o+e> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell o+e>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_StreamRedirect_OutErr(t *testing.T) {
	tokens, err := SplitWith("cat foo out+err> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell out+err>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_StreamRedirect_Short(t *testing.T) {
	tokens, err := SplitWith("cat foo o> bar e> baz", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell o> e>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_StreamPipe_EPipe(t *testing.T) {
	tokens, err := SplitWith("cat foo e>| bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("nushell e>|: %d pipelines, want 2", len(pipelines))
	}
}

func TestNushellFormat_StreamPipe_OEPipe(t *testing.T) {
	tokens, err := SplitWith("cat foo o+e>| bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("nushell o+e>|: %d pipelines, want 2", len(pipelines))
	}
}

func TestNushellFormat_StreamPipe_ErrPipe(t *testing.T) {
	tokens, err := SplitWith("cat foo err>| bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("nushell err>|: %d pipelines, want 2", len(pipelines))
	}
}

func TestNushellFormat_StreamRedirect_IsRedirect(t *testing.T) {
	// Verify that completion context detects redirect after out>
	tokens, err := SplitWith("cat foo out> ", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	ctx := SplitForCompletion("cat foo out> ", NushellFormat())
	if !ctx.IsRedirect {
		t.Errorf("nushell out> completion: IsRedirect = false, want true")
	}
	// Verify the redirect was filtered from words
	if len(tokens.CurrentPipeline().FilterRedirects().Words().Strings()) != 2 {
		words := tokens.CurrentPipeline().FilterRedirects().Words().Strings()
		t.Errorf("nushell out> completion: Words = %v, want 2 words", words)
	}
}

func TestNushellFormat_NonStreamWordNotMerged(t *testing.T) {
	// A regular word like "foo" before > should NOT be merged as a stream redirect
	tokens, err := SplitWith("echo foo > bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("nushell plain >: Words = %v, want [echo foo]", words)
	}
}

func TestNushellFormat_StreamRedirectWithAppend(t *testing.T) {
	// out>> should be recognized (out + >>)
	tokens, err := SplitWith("cat foo out>> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell out>>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_QuotedStreamWordNotMerged(t *testing.T) {
	// A quoted word like 'out' before > should NOT be merged as a stream
	// redirect — it's a string literal, not an operator.
	tests := []struct {
		name  string
		input string
	}{
		{"single_quoted", "echo 'out'>bar"},
		{"double_quoted", `echo "out">bar`},
		{"backtick_quoted", "echo `out`>bar"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := SplitWith(tc.input, NushellFormat())
			if err != nil {
				t.Fatal(err)
			}
			filtered := tokens.CurrentPipeline().FilterRedirects()
			words := filtered.Words().Strings()
			if len(words) != 2 || words[0] != "echo" || words[1] != "out" {
				t.Errorf("Words = %v, want [echo out]", words)
			}
		})
	}
}

// --- Backslash outside quotes (bareword) tests ---

func TestNushellFormat_BackslashBareword(t *testing.T) {
	// Nushell: backslash outside quotes is a regular word character, not an escape.
	// foo\bar should be a single word.
	tokens, err := SplitWith(`echo foo\bar`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != `foo\bar` {
		t.Errorf("nushell backslash bareword: Words = %v, want [echo foo\\bar]", words)
	}
}

func TestNushellFormat_BackslashSpaceNotEscape(t *testing.T) {
	// foo\ bar should be two words: foo\ and bar (backslash doesn't escape space)
	tokens, err := SplitWith(`echo foo\ bar`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != `foo\` || words[2] != "bar" {
		t.Errorf("nushell backslash-space: Words = %v, want [echo foo\\ bar]", words)
	}
}

// --- < not a wordbreak test ---

func TestNushellFormat_LessThanNotWordbreak(t *testing.T) {
	// Nushell does not support input redirection with <.
	// foo<bar should be a single word (comparison operator context).
	tokens, err := SplitWith("echo foo<bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo<bar" {
		t.Errorf("nushell < not wordbreak: Words = %v, want [echo foo<bar]", words)
	}
}

// --- Additional stream redirect alias tests ---

func TestNushellFormat_StreamRedirect_ErrOut(t *testing.T) {
	tokens, err := SplitWith("cat foo err+out> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell err+out>: Words = %v, want [cat foo]", words)
	}
}

func TestNushellFormat_StreamRedirect_EO(t *testing.T) {
	tokens, err := SplitWith("cat foo e+o> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
		t.Errorf("nushell e+o>: Words = %v, want [cat foo]", words)
	}
}

// --- Append redirect classification tests ---

func TestNushellFormat_AppendRedirect(t *testing.T) {
	// >> alone should be WORDBREAK_REDIRECT_OUTPUT_APPEND
	tokens, err := SplitWith("echo foo >> bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("nushell >>: Words = %v, want [echo foo]", words)
	}
}

func TestNushellFormat_StreamRedirectAppendBoth(t *testing.T) {
	// out+err>> and o+e>> should be append-both redirects
	for _, input := range []string{
		"cat foo out+err>> bar",
		"cat foo o+e>> bar",
		"cat foo err+out>> bar",
		"cat foo e+o>> bar",
	} {
		t.Run(input, func(t *testing.T) {
			tokens, err := SplitWith(input, NushellFormat())
			if err != nil {
				t.Fatal(err)
			}
			filtered := tokens.CurrentPipeline().FilterRedirects()
			words := filtered.Words().Strings()
			if len(words) != 2 || words[0] != "cat" || words[1] != "foo" {
				t.Errorf("Words = %v, want [cat foo]", words)
			}
		})
	}
}

// --- Pipe variant classification tests ---

func TestNushellFormat_StreamPipe_OutPipeIsPlainPipe(t *testing.T) {
	// out>| in nushell is a plain Pipe (stdout to pipe = normal piping)
	tokens, err := SplitWith("echo foo out>| bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("nushell out>|: %d pipelines, want 2", len(pipelines))
	}
}

func TestNushellFormat_StreamPipe_AllPipeVariants(t *testing.T) {
	// All pipe variants should split into 2 pipelines
	tests := []struct {
		name  string
		input string
	}{
		{"err_pipe", "cat foo err>| bar"},
		{"e_pipe", "cat foo e>| bar"},
		{"oe_pipe", "cat foo o+e>| bar"},
		{"eo_pipe", "cat foo e+o>| bar"},
		{"errout_pipe", "cat foo err+out>| bar"},
		{"outerr_pipe", "cat foo out+err>| bar"},
		{"out_pipe", "cat foo out>| bar"},
		{"o_pipe", "cat foo o>| bar"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := SplitWith(tc.input, NushellFormat())
			if err != nil {
				t.Fatal(err)
			}
			pipelines := tokens.Pipelines()
			if len(pipelines) != 2 {
				t.Errorf("%s: %d pipelines, want 2", tc.name, len(pipelines))
			}
		})
	}
}

// --- Completion context redirect detection ---

func TestNushellFormat_CompletionRedirectAfterErr(t *testing.T) {
	ctx := SplitForCompletion("cat foo err> ", NushellFormat())
	if !ctx.IsRedirect {
		t.Errorf("nushell err> completion: IsRedirect = false, want true")
	}
}

func TestNushellFormat_CompletionRedirectAfterOutAppend(t *testing.T) {
	ctx := SplitForCompletion("cat foo out>> ", NushellFormat())
	if !ctx.IsRedirect {
		t.Errorf("nushell out>> completion: IsRedirect = false, want true")
	}
}
