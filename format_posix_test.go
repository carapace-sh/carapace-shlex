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
	// Without RC_QUOTES (bash), '' closes then reopens → two words
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
	// Open single quote at end
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

func TestOilFormat(t *testing.T) {
	// OSH is bash-compatible
	tokens, err := SplitWith(`echo "hello" world`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "hello" || words[2] != "world" {
		t.Errorf("OilFormat: Words = %v, want [echo hello world]", words)
	}
}

func TestTcshFormat(t *testing.T) {
	// tcsh uses same grammar as bash
	tokens, err := SplitWith("echo foo | grep bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("TcshFormat: %d pipelines, want 2", len(pipelines))
	}
}

func TestTcshFormat_BackslashQuote(t *testing.T) {
	// tcsh $'...' ANSI-C quoting lexes same as bash
	tokens, err := SplitWith("echo $'hello'", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "$hello" {
		t.Errorf("TcshFormat $'': Words = %v, want [echo $hello]", words)
	}
}
