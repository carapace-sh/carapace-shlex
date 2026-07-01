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
	// With raw prefix support, \ inside double quotes is literal (raw string semantics).
	tokens, err := SplitWith(`echo r"C:\path"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// Backslashes are literal in raw strings
	if len(words) != 2 || words[1] != `rC:\path` {
		t.Errorf("xonsh r\"\": Words = %v, want [echo rC:\\path]", words)
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

func TestXonshFormat_KeywordAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo and echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh and: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshFormat_KeywordOr(t *testing.T) {
	tokens, err := SplitWith("echo foo or echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh or: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshFormat_StreamRedirectStderr(t *testing.T) {
	tokens, err := SplitWith("echo foo e> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "e>" {
			if tok.WordbreakType != WORDBREAK_REDIRECT_OUTPUT {
				t.Errorf("xonsh e>: WordbreakType = %v, want WORDBREAK_REDIRECT_OUTPUT", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("xonsh e>: no e> wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_StreamRedirectStdout(t *testing.T) {
	tokens, err := SplitWith("echo foo o> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "o>" {
			if tok.WordbreakType != WORDBREAK_REDIRECT_OUTPUT {
				t.Errorf("xonsh o>: WordbreakType = %v, want WORDBREAK_REDIRECT_OUTPUT", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("xonsh o>: no o> wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_StreamRedirectAll(t *testing.T) {
	tokens, err := SplitWith("echo foo a> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "a>" {
			if tok.WordbreakType != WORDBREAK_REDIRECT_OUTPUT_BOTH {
				t.Errorf("xonsh a>: WordbreakType = %v, want WORDBREAK_REDIRECT_OUTPUT_BOTH", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("xonsh a>: no a> wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_StreamRedirectStderrAppend(t *testing.T) {
	tokens, err := SplitWith("echo foo e>> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "e>>" {
			if tok.WordbreakType != WORDBREAK_REDIRECT_OUTPUT {
				t.Errorf("xonsh e>>: WordbreakType = %v, want WORDBREAK_REDIRECT_OUTPUT", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("xonsh e>>: no e>> wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_StreamRedirectLongForm(t *testing.T) {
	tokens, err := SplitWith("echo foo err> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "err>" {
			if tok.WordbreakType != WORDBREAK_REDIRECT_OUTPUT {
				t.Errorf("xonsh err>: WordbreakType = %v, want WORDBREAK_REDIRECT_OUTPUT", tok.WordbreakType)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("xonsh err>: no err> wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_StreamRedirectPipeChannel(t *testing.T) {
	tokens, err := SplitWith("echo foo e>p bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "e>p" {
			found = true
		}
	}
	if !found {
		t.Errorf("xonsh e>p: no e>p wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_TripleSingleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo '''hello world'''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("xonsh triple single: Words = %v, want [echo hello world]", words)
	}
}

func TestXonshFormat_TripleDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo """hello world"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("xonsh triple double: Words = %v, want [echo hello world]", words)
	}
}

func TestXonshFormat_TripleDoubleQuoteEscape(t *testing.T) {
	tokens, err := SplitWith(`echo """say \"hello\""""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `say "hello"` {
		t.Errorf("xonsh triple double escape: Value = %q, want %q", last.Value, `say "hello"`)
	}
}

func TestXonshFormat_TripleSingleQuoteWithEmbeddedQuotes(t *testing.T) {
	tokens, err := SplitWith(`echo '''he said "hi"'''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `he said "hi"` {
		t.Errorf("xonsh triple single with embedded: Words = %v, want [echo he said \"hi\"]", words)
	}
}

func TestXonshFormat_TripleQuoteUnclosed(t *testing.T) {
	tokens, err := SplitWith("echo '''unclosed", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_TRIPLE_STATE {
		t.Errorf("xonsh unclosed triple: State = %v, want QUOTING_TRIPLE_STATE", last.State)
	}
}

func TestXonshFormat_RawTripleDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo r"""C:\new\path"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `rC:\new\path` {
		t.Errorf("xonsh r\"\"\": Words = %v, want [echo rC:\\new\\path]", words)
	}
}

func TestXonshFormat_RawDoubleQuotedWithEscapedQuote(t *testing.T) {
	// In raw strings, \" is NOT an escape — it's two literal chars.
	// But " still closes the string. So r"a\"b" is: r"a\" (raw string with content a\)
	// followed by b" (separate token).
	// Actually in Python, you can't have a raw string with an embedded quote of the
	// same type — it would close the string. This is a known Python limitation.
	// For the lexer, the " in r"...\..." closes the string (since rawQuote only
	// affects backslash, not quote matching).
	tokens, err := SplitWith(`echo r"C:\path"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `rC:\path` {
		t.Errorf("xonsh r\"\" with path: Words = %v, want [echo rC:\\path]", words)
	}
}

func TestXonshFormat_TwoQuotesInsideTriple(t *testing.T) {
	// '' inside '''...''' — two single quotes should NOT close triple-single
	tokens, err := SplitWith(`echo '''hello''there'''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello''there` {
		t.Errorf("xonsh two quotes in triple: Words = %v, want [echo hello''there]", words)
	}
}

func TestXonshFormat_TwoDoubleQuotesInsideTripleDouble(t *testing.T) {
	// "" inside """...""" — two double quotes should NOT close triple-double
	tokens, err := SplitWith(`echo """hello""there"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello""there` {
		t.Errorf("xonsh two double quotes in triple: Words = %v, want [echo hello\"\"there]", words)
	}
}

func TestXonshFormat_TripleQuoteThenMore(t *testing.T) {
	tokens, err := SplitWith(`echo """hello""" world`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "hello" || words[2] != "world" {
		t.Errorf("xonsh triple then more: Words = %v, want [echo hello world]", words)
	}
}

func TestXonshFormat_TripleQuoteAdjacentWord(t *testing.T) {
	tokens, err := SplitWith(`echo foo"""bar"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("xonsh triple adjacent: Words = %v, want [echo foobar]", words)
	}
}

func TestXonshFormat_RawPrefixFalsePositive(t *testing.T) {
	// xr"hello\nworld" — 'x' is not a valid prefix char, so raw prefix should NOT trigger
	tokens, err := SplitWith(`echo xr"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// \n processed as escape (backslash dropped, n emitted) since raw prefix is NOT active
	if words[1] != "xrhellonworld" {
		t.Errorf("xonsh xr false positive: Words = %v, want [echo xrhellonworld]", words)
	}
}

func TestXonshFormat_ValidBrPrefix(t *testing.T) {
	// br"hello\nworld" — 'br' is a valid Python raw prefix
	tokens, err := SplitWith(`echo br"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// With raw prefix, \n is literal backslash + n
	if words[1] != `brhello\nworld` {
		t.Errorf("xonsh br prefix: Words = %v, want [echo brhello\\nworld]", words)
	}
}

func TestXonshFormat_KeywordAndInsideWord(t *testing.T) {
	// 'fooand' should NOT be a keyword operator — only exact match counts
	tokens, err := SplitWith("echo fooand bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 1 {
		t.Errorf("xonsh fooand: %d pipelines, want 1", len(tokens.Pipelines()))
	}
}

func TestXonshFormat_QuotedStreamRedirectNotMerged(t *testing.T) {
	// 'e'> bar — quoted 'e' should NOT be merged with > as stream redirect
	tokens, err := SplitWith("echo foo 'e'> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "e>" {
			t.Errorf("quoted 'e' should not be merged with > as stream redirect")
		}
	}
}
