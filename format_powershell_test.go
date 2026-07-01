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

func TestPowershellFormat_BacktickLineContinuation(t *testing.T) {
	// backtick + newline should be consumed as line continuation, not part of word
	tokens, err := SplitWith("echo foo`\nbar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("powershell line continuation: Words = %v, want [echo foobar]", words)
	}
}

func TestPowershellFormat_BacktickLineContinuationCRLF(t *testing.T) {
	tokens, err := SplitWith("echo foo`\r\nbar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("powershell line continuation CRLF: Words = %v, want [echo foobar]", words)
	}
}

func TestPowershellFormat_BacktickLineContinuationStartOfWord(t *testing.T) {
	// backtick + newline at start of word — word continues on next line
	tokens, err := SplitWith("echo `\nbar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "bar" {
		t.Errorf("powershell line continuation start: Words = %v, want [echo bar]", words)
	}
}

func TestPowershellFormat_BlockComment(t *testing.T) {
	tokens, err := SplitWith("echo <# multi\nline\ncomment #> foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("powershell block comment: Words = %v, want [echo foo]", words)
	}
}

func TestPowershellFormat_BlockCommentSingleLine(t *testing.T) {
	tokens, err := SplitWith("echo <# inline comment #> foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("powershell block comment inline: Words = %v, want [echo foo]", words)
	}
}

func TestPowershellFormat_StopParsingToken(t *testing.T) {
	// After --%, everything is literal until newline or |
	tokens, err := SplitWith("echo --% /grant Dom\\HVAdmin:(CI)(OI)F", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// --% should be a word, then the rest is raw text as one word
	if len(words) < 3 {
		t.Errorf("powershell --%%: Words = %v, expected at least 3 words", words)
	}
	if words[0] != "echo" {
		t.Errorf("powershell --%%: first word = %q, want echo", words[0])
	}
	if words[1] != "--%" {
		t.Errorf("powershell --%%: second word = %q, want --%%", words[1])
	}
}

func TestPowershellFormat_StopParsingPipeDelim(t *testing.T) {
	// After --%, | is still a pipeline delimiter
	tokens, err := SplitWith("echo --% foo | Select-String bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("powershell --%% pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestPowershellFormat_StopParsingRawContent(t *testing.T) {
	// After --%, content like (CI) should be literal, not split
	tokens, err := SplitWith("icacls X: --% /grant Dom\\HVAdmin:(CI)(OI)F", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// The raw content after --% should be one word
	if len(words) != 4 {
		t.Errorf("powershell --%% raw: Words = %v, want 4 words", words)
	}
	if words[2] != "--%" {
		t.Errorf("powershell --%% raw: third word = %q, want --%%", words[2])
	}
	rawContent := words[3]
	if rawContent != "/grant Dom\\HVAdmin:(CI)(OI)F" {
		t.Errorf("powershell --%% raw: content = %q, want /grant Dom\\HVAdmin:(CI)(OI)F", rawContent)
	}
}

func TestPowershellFormat_StreamRedirect2(t *testing.T) {
	// 2> should be recognized as a stream redirect
	tokens, err := SplitWith("echo foo 2> error.txt", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	// Check that 2> is a single WORDBREAK_TOKEN with redirect type
	found := false
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "2>" {
			if !tok.WordbreakType.IsRedirect() {
				t.Errorf("powershell 2>: WordbreakType = %v, want redirect", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("powershell 2>: no merged 2> token found in %v", tokens)
	}
}

func TestPowershellFormat_StreamRedirect2Append(t *testing.T) {
	tokens, err := SplitWith("echo foo 2>> error.txt", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "2>>" {
			if !tok.WordbreakType.IsRedirect() {
				t.Errorf("powershell 2>>: WordbreakType = %v, want redirect", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("powershell 2>>: no merged 2>> token found in %v", tokens)
	}
}

func TestPowershellFormat_StreamRedirectMerge(t *testing.T) {
	// 2>&1 should be recognized as a merged stream redirect
	tokens, err := SplitWith("echo foo 2>&1", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "2>&1" {
			found = true
		}
	}
	if !found {
		t.Errorf("powershell 2>&1: no merged token found in %v", tokens)
	}
}

func TestPowershellFormat_StreamRedirectStar(t *testing.T) {
	// *> should be recognized as all-streams redirect
	tokens, err := SplitWith("echo foo *> output.txt", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "*>" {
			if !tok.WordbreakType.IsRedirect() {
				t.Errorf("powershell *>: WordbreakType = %v, want redirect", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("powershell *>: no merged *> token found in %v", tokens)
	}
}
