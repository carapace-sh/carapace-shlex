package shlex

import "testing"

func TestXonshFormat_RawPrefixRb(t *testing.T) {
	// rb"hello\nworld" — rb is a valid Python raw prefix, should be detected
	tokens, err := SplitWith(`echo rb"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// With raw prefix, \n is literal backslash + n
	if words[1] != `rbhello\nworld` {
		t.Errorf("rb prefix: Words = %v, want [echo rbhello\\nworld]", words)
	}
}

func TestXonshFormat_RawPrefixRf(t *testing.T) {
	// rf"hello\nworld" — rf is a valid Python raw prefix, should be detected
	tokens, err := SplitWith(`echo rf"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `rfhello\nworld` {
		t.Errorf("rf prefix: Words = %v, want [echo rfhello\\nworld]", words)
	}
}

func TestXonshFormat_RawPrefixRbf(t *testing.T) {
	// rbf"hello\nworld" — rbf is a valid Python 3 raw prefix, should be detected
	tokens, err := SplitWith(`echo rbf"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `rbfhello\nworld` {
		t.Errorf("rbf prefix: Words = %v, want [echo rbfhello\\nworld]", words)
	}
}

func TestXonshFormat_RawPrefixUppercaseR(t *testing.T) {
	// R"hello\nworld" — uppercase R prefix, should be detected as raw
	tokens, err := SplitWith(`echo R"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `Rhello\nworld` {
		t.Errorf("R prefix: Words = %v, want [echo Rhello\\nworld]", words)
	}
}

func TestXonshFormat_RawPrefixBR(t *testing.T) {
	// BR"hello\nworld" — uppercase BR prefix, should be detected as raw
	tokens, err := SplitWith(`echo BR"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `BRhello\nworld` {
		t.Errorf("BR prefix: Words = %v, want [echo BRhello\\nworld]", words)
	}
}

func TestXonshFormat_RawPrefixFalsePositiveRR(t *testing.T) {
	// rr"hello\nworld" — rr is NOT a valid Python prefix (duplicate r),
	// should NOT be detected as raw
	tokens, err := SplitWith(`echo rr"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// Without raw prefix, \n is processed as escape (backslash dropped, n emitted)
	if words[1] != `rrhellonworld` {
		t.Errorf("rr false positive: Words = %v, want [echo rrhellonworld]", words)
	}
}

func TestXonshFormat_RawPrefixFalsePositiveBB(t *testing.T) {
	// bb"hello\nworld" — bb is NOT valid Python prefix (duplicate b),
	// should NOT be detected as raw
	tokens, err := SplitWith(`echo bb"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `bbhellonworld` {
		t.Errorf("bb false positive: Words = %v, want [echo bbhellonworld]", words)
	}
}

func TestXonshFormat_NonRawPrefixF(t *testing.T) {
	// f"hello\nworld" — f-string, NOT raw, escapes should be processed
	tokens, err := SplitWith(`echo f"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `fhellonworld` {
		t.Errorf("f prefix: Words = %v, want [echo fhellonworld]", words)
	}
}

func TestXonshFormat_NonRawPrefixB(t *testing.T) {
	// b"hello\nworld" — bytes prefix, NOT raw, escapes should be processed
	tokens, err := SplitWith(`echo b"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `bhellonworld` {
		t.Errorf("b prefix: Words = %v, want [echo bhellonworld]", words)
	}
}

func TestXonshFormat_RawTripleRb(t *testing.T) {
	// rb"""hello\nworld""" — raw triple-double with rb prefix
	tokens, err := SplitWith(`echo rb"""hello\nworld"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != `rbhello\nworld` {
		t.Errorf("rb triple: Words = %v, want [echo rbhello\\nworld]", words)
	}
}

func TestXonshFormat_EmptyTripleDouble(t *testing.T) {
	// """""" — six double quotes = empty triple-quoted string
	tokens, err := SplitWith(`echo """"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != "" {
		t.Errorf("empty triple-double: Words = %v, want [echo ]", words)
	}
}

func TestXonshFormat_EmptyTripleSingle(t *testing.T) {
	// '''''' — six single quotes = empty triple-quoted string
	tokens, err := SplitWith(`echo ''''''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if words[1] != "" {
		t.Errorf("empty triple-single: Words = %v, want [echo ]", words)
	}
}

func TestXonshFormat_TripleQuoteAtEOFNoContent(t *testing.T) {
	// """ at EOF with no content — should be in QUOTING_TRIPLE_ESCAPING_STATE
	tokens, err := SplitWith(`echo """`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	if len(words) != 2 {
		t.Fatalf("Words = %v, want 2 words", words)
	}
	if words[1].State != QUOTING_TRIPLE_ESCAPING_STATE {
		t.Errorf("State = %v, want QUOTING_TRIPLE_ESCAPING_STATE", words[1].State)
	}
}

func TestXonshFormat_TripleSingleAtEOFNoContent(t *testing.T) {
	// ''' at EOF with no content — should be in QUOTING_TRIPLE_STATE
	tokens, err := SplitWith(`echo '''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	if len(words) != 2 {
		t.Fatalf("Words = %v, want 2 words", words)
	}
	if words[1].State != QUOTING_TRIPLE_STATE {
		t.Errorf("State = %v, want QUOTING_TRIPLE_STATE", words[1].State)
	}
}

func TestXonshFormat_KeywordUppercase(t *testing.T) {
	// AND/OR uppercase — Python keywords are case-sensitive, should NOT split
	tokens, err := SplitWith("echo foo AND echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("Pipelines = %d, want 1 (AND uppercase should not split)", len(pipelines))
	}
}

func TestXonshFormat_KeywordAtEOF(t *testing.T) {
	// "and" at EOF — shlex keyword matching checks RawValue=="and"
	// This should split (shlex doesn't enforce trailing whitespace)
	tokens, err := SplitWith("echo foo and", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	// "and" is reclassified as a keyword operator, splitting the pipeline
	if len(pipelines) != 2 {
		t.Errorf("Pipelines = %d, want 2 (and at EOF should split)", len(pipelines))
	}
}

func TestXonshFormat_StreamRedirectAtEOF(t *testing.T) {
	// e> at EOF — should be recognized as redirect
	ctx := SplitForCompletion("echo foo e>", XonshFormat())
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true (e> at EOF)")
	}
}

func TestXonshFormat_AppendPipeChannel(t *testing.T) {
	// e>>p — append pipe channel, classified as redirect (not pipeline delimiter)
	tokens, err := SplitWith("echo foo e>>p bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "e>>p" {
			found = true
			if !tok.WordbreakType.IsRedirect() {
				t.Errorf("e>>p WordbreakType should be a redirect, got %v", tok.WordbreakType)
			}
		}
	}
	if !found {
		t.Errorf("e>>p: no e>>p wordbreak token found in %v", tokens)
	}
}

func TestXonshFormat_LongFormPipeChannel(t *testing.T) {
	// err>p — long-form pipe channel, classified as redirect
	tokens, err := SplitWith("echo foo err>p bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "err>p" {
			found = true
			if !tok.WordbreakType.IsRedirect() {
				t.Errorf("err>p WordbreakType should be a redirect, got %v", tok.WordbreakType)
			}
		}
	}
	if !found {
		t.Errorf("err>p: no err>p wordbreak token found in %v", tokens)
	}
}
