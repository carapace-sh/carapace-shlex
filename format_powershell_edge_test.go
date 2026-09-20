package shlex

import "testing"

func TestPowershellFormat_UnclosedBlockComment(t *testing.T) {
	tokens, err := SplitWith("echo <# unclosed comment", PowershellFormat())
	if err != nil {
		t.Fatalf("Unclosed block comment should not error: %v", err)
	}
	words := tokens.Words().Strings()
	if len(words) != 1 || words[0] != "echo" {
		t.Errorf("Words = %v, want [echo]", words)
	}
}

func TestPowershellFormat_BlockCommentAtStart(t *testing.T) {
	tokens, err := SplitWith("<# comment #> echo foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("Words = %v, want [echo foo]", words)
	}
}

func TestPowershellFormat_ZeroLengthBlockComment(t *testing.T) {
	tokens, err := SplitWith("echo <##> foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("Words = %v, want [echo foo]", words)
	}
}

func TestPowershellFormat_StopParsingAtEOF(t *testing.T) {
	tokens, err := SplitWith("echo --%", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "--%" {
		t.Errorf("Words = %v, want [echo --%%]", words)
	}
}

func TestPowershellFormat_StopParsingPipeInQuotes(t *testing.T) {
	tokens, err := SplitWith(`echo --% "foo | bar"`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("Pipelines = %d, want 1 (pipe inside quotes after stop-parsing token should be literal)", len(pipelines))
	}
}

func TestPowershellFormat_StopParsingUnclosedQuotePipe(t *testing.T) {
	tokens, err := SplitWith(`echo --% "foo | bar`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("Pipelines = %d, want 1 (pipe inside unclosed quote after stop-parsing token should be literal)", len(pipelines))
	}
}

func TestPowershellFormat_BacktickAtEOFInDoubleQuotes(t *testing.T) {
	tokens, err := SplitWith("echo \"hel`", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	if len(words) != 2 {
		t.Fatalf("Words = %v, want 2 words", words)
	}
	if words[1].Value != "hel" {
		t.Errorf("Value = %q, want hel (backtick dropped at EOF)", words[1].Value)
	}
}

func TestPowershellFormat_BacktickEscapePipe(t *testing.T) {
	tokens, err := SplitWith("echo `|", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "|" {
		t.Errorf("Words = %v, want [echo |] (backtick escapes pipe)", words)
	}
}

func TestPowershellFormat_BacktickEscapeOperators(t *testing.T) {
	ops := []string{"&", ";", ">", "<", "(", ")"}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			tokens, err := SplitWith("echo `"+op, PowershellFormat())
			if err != nil {
				t.Fatal(err)
			}
			words := tokens.Words().Strings()
			if len(words) != 2 || words[1] != op {
				t.Errorf("Words = %v, want [echo %s] (backtick escapes %s)", words, op, op)
			}
		})
	}
}

func TestPowershellFormat_DoubleBacktick(t *testing.T) {
	tokens, err := SplitWith("echo ``foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "`foo" {
		t.Errorf("Words = %v, want [echo `foo] (double backtick = literal)", words)
	}
}

func TestPowershellFormat_BacktickLineContinuationInDoubleQuotes(t *testing.T) {
	tokens, err := SplitWith("echo \"foo`\nbar\"", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("Words = %v, want [echo foobar] (backtick+newline = line continuation)", words)
	}
}

func TestPowershellFormat_TripleDoubleQuoteAtStart(t *testing.T) {
	tokens, err := SplitWith(`echo """hello"`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// """ at start: first " opens, second " peeks third " → doubled, emit literal "
	// third " → peek h → not ", close. Then hello, then " opens new quote (unclosed).
	if words[1] != `"hello` {
		t.Errorf("Words = %v, want [echo \"hello]", words)
	}
}

func TestPowershellFormat_QuadrupleDoubleQuote(t *testing.T) {
	// """" → 1st opens, 2nd peeks 3rd (match) → literal ", 3rd consumed by peek,
	// 4th peeks EOF → close. Value = one literal ".
	tokens, err := SplitWith(`echo """"`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `"` {
		t.Errorf("Words = %v, want [echo \"]", words)
	}
}

func TestPowershellFormat_TripleSingleQuoteAtStart(t *testing.T) {
	tokens, err := SplitWith("echo '''hello'", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != "'hello" {
		t.Errorf("Words = %v, want [echo 'hello]", words)
	}
}

func TestPowershellFormat_DoubleOrPipelineSplit(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("Pipelines = %d, want 2 (|| splits)", len(pipelines))
	}
}
