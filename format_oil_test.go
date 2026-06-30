package shlex

import "testing"

func TestOilFormat(t *testing.T) {
	tokens, err := SplitWith(`echo "hello" world`, OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "hello" || words[2] != "world" {
		t.Errorf("OilFormat: Words = %v, want [echo hello world]", words)
	}
}

func TestOilFormat_SingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hello world'", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("oil single: Words = %v, want [echo hello world]", words)
	}
}

func TestOilFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilFormat_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}
