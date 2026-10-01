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
	words := []string{"echo", `it's "tricky"; really`, "foo|bar"}
	tokens, err := Split(Join(words, Hilbish), Hilbish)
	if err != nil {
		t.Fatal(err)
	}
	got := tokens.Words().Strings()
	if len(got) != 3 || got[1] != words[1] || got[2] != words[2] {
		t.Errorf("hilbish join roundtrip: got %v, want %v", got, words)
	}
}
