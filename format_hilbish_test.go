package shlex

import "testing"

func TestHilbishFormat(t *testing.T) {
	tokens, err := Split("echo foo | grep bar", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.pipelines()
	if len(pipelines) != 2 {
		t.Errorf("HilbishFormat: %d pipelines, want 2", len(pipelines))
	}
}

func TestHilbishFormat_SingleQuoteLiteral(t *testing.T) {
	tokens, err := Split("echo '$HOME'", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("hilbish single literal: Words = %v, want [echo $HOME]", words)
	}
}

func TestHilbishFormat_BacktickLiteralInSingleQuotes(t *testing.T) {
	tokens, err := Split("echo '`cmd`'", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "`cmd`" {
		t.Errorf("hilbish backtick in single: Words = %v, want [echo `cmd`]", words)
	}
}

func TestHilbishFormat_EscapedDoubleQuoteOutside(t *testing.T) {
	tokens, err := Split(`echo \"hello\"`, Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `"hello"` {
		t.Errorf("hilbish escaped double: Words = %v, want [echo \"hello\"]", words)
	}
}

func TestHilbishFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := Split("echo 'hel", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().currentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("hilbish open single: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestHilbishFormat_Comment(t *testing.T) {
	tokens, err := Split("echo foo # bar", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("hilbish comment: Words = %v, want [echo foo]", words)
	}
}

func TestHilbishFormat_PipeWithStderr(t *testing.T) {
	tokens, err := Split("echo foo |& grep bar", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "|&" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_PIPE_WITH_STDERR {
		t.Errorf("hilbish |&: Type=%v WT=%v, want WORDBREAK_TOKEN/PIPE_WITH_STDERR", op.Type, op.WordbreakType)
	}
	if len(tokens.pipelines()) != 2 {
		t.Errorf("hilbish |&: %d pipelines, want 2", len(tokens.pipelines()))
	}
}

func TestHilbishFormat_RedirectBoth(t *testing.T) {
	tokens, err := Split("echo foo &> /tmp/bar", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "&>" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_OUTPUT_BOTH {
		t.Errorf("hilbish &>: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_OUTPUT_BOTH", op.Type, op.WordbreakType)
	}
}

func TestHilbishFormat_HereDoc(t *testing.T) {
	tokens, err := Split("cat << EOF", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "<<" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_HERE_DOC {
		t.Errorf("hilbish <<: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_HERE_DOC", op.Type, op.WordbreakType)
	}
}

func TestHilbishFormat_HereString(t *testing.T) {
	tokens, err := Split("cat <<< foo", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	var op Token
	for _, tok := range tokens {
		if tok.RawValue == "<<<" {
			op = tok
		}
	}
	if op.Type != WORDBREAK_TOKEN || op.WordbreakType != WORDBREAK_REDIRECT_INPUT_STRING {
		t.Errorf("hilbish <<<: Type=%v WT=%v, want WORDBREAK_TOKEN/REDIRECT_INPUT_STRING", op.Type, op.WordbreakType)
	}
}

func TestHilbishFormat_CaseList(t *testing.T) {
	tokens, err := Split("case foo in a) echo one ;; b) echo two ;; esac", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	var sep Token
	for _, tok := range tokens {
		if tok.RawValue == ";;" {
			sep = tok
		}
	}
	if sep.Type != WORDBREAK_TOKEN || sep.WordbreakType != WORDBREAK_LIST_SEQUENTIAL_DOUBLE {
		t.Errorf("hilbish ;;: Type=%v WT=%v, want WORDBREAK_TOKEN/LIST_SEQUENTIAL_DOUBLE", sep.Type, sep.WordbreakType)
	}
}

func TestHilbishFormat_Substitution(t *testing.T) {
	tokens, err := Split("echo $(date)", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	var opener Token
	for _, tok := range tokens {
		if tok.RawValue == "$(" {
			opener = tok
		}
	}
	if opener.Type != WORDBREAK_TOKEN || opener.WordbreakType != WORDBREAK_SUBSTITUTION_OPEN {
		t.Errorf("hilbish $(: Type=%v WT=%v, want WORDBREAK_TOKEN/SUBSTITUTION_OPEN", opener.Type, opener.WordbreakType)
	}
}

func TestHilbishFormat_Variable(t *testing.T) {
	ctx := Complete(`echo "text$HO`, Hilbish)
	if ctx.Variable == nil || ctx.Variable.Name != "HO" {
		t.Errorf("hilbish variable: Variable = %v, want {HO}", ctx.Variable)
	}
}

func TestHilbishFormat_LineContinuation(t *testing.T) {
	tokens, err := Split("echo foo\\\nbar", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("hilbish line continuation: Words = %v, want [echo foobar]", words)
	}
}

func TestHilbishFormat_Join(t *testing.T) {
	joined := Join([]string{"echo", "hello world"}, Hilbish)
	if joined != `echo hello\ world` {
		t.Errorf("hilbish join: %q, want %q", joined, `echo hello\ world`)
	}
}

func TestHilbishFormat_JoinRoundTrip(t *testing.T) {
	words := []string{"echo", `it's "tricky"; really`, "foo|bar", "a\tb"}
	tokens, err := Split(Join(words, Hilbish), Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	got := tokens.Words().Strings()
	if len(got) != 4 || got[1] != words[1] || got[2] != words[2] || got[3] != words[3] {
		t.Errorf("hilbish join roundtrip: got %v, want %v", got, words)
	}
}

func TestHilbishFormat_AdjacentQuoteSegments(t *testing.T) {
	tokens, err := Split(`echo a''b "foo"bar`, Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[1] != "ab" || words[2] != "foobar" {
		t.Errorf("hilbish adjacent quotes: Words = %v, want [echo ab foobar]", words)
	}
}

func TestHilbishFormat_EmptyQuotes(t *testing.T) {
	tokens, err := Split(`echo ""`, Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "" {
		t.Errorf("hilbish empty quotes: Words = %v, want [echo ]", words)
	}
}

func TestHilbishFormat_OpenDoubleQuoteAtEOF(t *testing.T) {
	tokens, err := Split(`echo "hel`, Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().currentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("hilbish open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}

func TestHilbishFormat_EscapeAtEOF(t *testing.T) {
	tokens, err := Split(`echo hel\`, Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().currentToken()
	if last.State != ESCAPING_STATE {
		t.Errorf("hilbish escape at EOF: State = %v, want ESCAPING_STATE", last.State)
	}
}

func TestHilbishFormat_CommentOnlyAtWordStart(t *testing.T) {
	tokens, err := Split("echo a#b #c", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a#b" {
		t.Errorf("hilbish mid-word #: Words = %v, want [echo a#b]", words)
	}
}

func TestHilbishFormat_RedirectFilterNumericFd(t *testing.T) {
	tokens, err := Split("echo foo 2>&1 | grep -", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.CurrentPipeline().FilterRedirects().Words().Strings()
	if len(words) != 2 || words[0] != "grep" || words[1] != "-" {
		t.Errorf("hilbish 2>&1 filter: Words = %v, want [grep -]", words)
	}
}

func TestHilbishFormat_VariableDetection(t *testing.T) {
	tests := []struct {
		input string
		name  string
	}{
		{`echo "text$HO`, "HO"},
		{`echo ${HO`, "HO"},
		{`echo $HO`, "HO"},
	}
	for _, tt := range tests {
		ctx := Complete(tt.input, Hilbish)
		if ctx.Variable == nil || ctx.Variable.Name != tt.name {
			t.Errorf("hilbish variable %q: Variable = %v, want name %q", tt.input, ctx.Variable, tt.name)
		}
	}
}

func TestHilbishFormat_VariableLiteral(t *testing.T) {
	tests := []string{`echo '$HO'`, `echo \$HO`}
	for _, input := range tests {
		ctx := Complete(input, Hilbish)
		if ctx.Variable != nil {
			t.Errorf("hilbish literal variable %q: Variable = %v, want nil", input, ctx.Variable)
		}
	}
}

func TestHilbishFormat_SpanRuneOffsets(t *testing.T) {
	ctx := Complete("echo café", Hilbish)
	// "echo " is 5 runes; café starts at rune offset 5, length 4
	if ctx.Span.Start != 5 || ctx.Span.End != 9 {
		t.Errorf("hilbish span: Span = %+v, want {5 9}", ctx.Span)
	}
}

func TestHilbishFormat_CRLFLineContinuation(t *testing.T) {
	tokens, err := Split("echo foo\\\r\nbar", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("hilbish CRLF continuation: Words = %v, want [echo foobar]", words)
	}
}

func TestHilbishFormat_ModifierWords(t *testing.T) {
	tokens, err := Split("@priv echo hi", Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "@priv" {
		t.Errorf("hilbish modifier: Words = %v, want [@priv echo hi]", words)
	}
}
