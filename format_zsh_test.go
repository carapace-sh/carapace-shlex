package shlex

import "testing"

func TestZshFormat_RCQuotes(t *testing.T) {
	// With RC_QUOTES, '' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it''s'", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "it's" {
		t.Errorf("RC_QUOTES: Value = %q, want %q", last.Value, "it's")
	}
	if last.State != IN_WORD_STATE {
		t.Errorf("RC_QUOTES: State = %v, want IN_WORD_STATE", last.State)
	}
}

func TestZshFormat_NoRCQuotes(t *testing.T) {
	// Without RC_QUOTES (bash), '' closes then reopens → words merge to "its"
	tokens, err := SplitWith("echo 'it''s'", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "its" {
		t.Errorf("bash: Value = %q, want %q", last.Value, "its")
	}
}

func TestZshFormat_OpenQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != QUOTING_STATE {
		t.Errorf("open quote: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestZshFormat_DoubleQuoteEscape(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("zsh double escape: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestZshFormat_RCQuotesLonger(t *testing.T) {
	tokens, err := SplitWith("echo 'it''s a test'", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's a test" {
		t.Errorf("zsh RC_QUOTES longer: Words = %v, want [echo it's a test]", words)
	}
}

func TestZshFormat_Operators(t *testing.T) {
	tests := []struct {
		input    string
		wantType WordbreakType
		wantRaw  string
	}{
		{"echo foo >| bar", WORDBREAK_REDIRECT_OUTPUT_FORCE, ">|"},
		{"echo foo >>| bar", WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE, ">>|"},
		{"echo foo ;& bar", WORDBREAK_LIST_FALLTHROUGH, ";&"},
		{"echo foo ;| bar", WORDBREAK_LIST_FALLTHROUGH_RETRY, ";|"},
		{"echo foo &| bar", WORDBREAK_LIST_ASYNC_ERRCHECK, "&|"},
		{"echo foo ;; bar", WORDBREAK_LIST_SEQUENTIAL_DOUBLE, ";;"},
	}
	for _, tt := range tests {
		tokens, err := SplitWith(tt.input, ZshFormat())
		if err != nil {
			t.Fatalf("zsh operator %q: %v", tt.wantRaw, err)
		}
		var found *Token
		for i := range tokens {
			if tokens[i].Type == WORDBREAK_TOKEN && tokens[i].RawValue == tt.wantRaw {
				found = &tokens[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("zsh operator %q: not found in tokens %v", tt.wantRaw, tokens)
		}
		if found.WordbreakType != tt.wantType {
			t.Errorf("zsh operator %q: WordbreakType = %v, want %v", tt.wantRaw, found.WordbreakType, tt.wantType)
		}
	}
}

func TestZshFormat_ForceRedirectIsRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo >| bar", ZshFormat())
	if !ctx.IsRedirect {
		t.Errorf("zsh >|: IsRedirect = false, want true")
	}
}

func TestZshFormat_ForceAppendRedirectIsRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo >>| bar", ZshFormat())
	if !ctx.IsRedirect {
		t.Errorf("zsh >>|: IsRedirect = false, want true")
	}
}

func TestZshFormat_FallthroughIsPipelineDelimiter(t *testing.T) {
	tokens, err := SplitWith("echo foo ;& bar", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("zsh ;&: Pipelines = %d, want 2", len(pipelines))
	}
}
