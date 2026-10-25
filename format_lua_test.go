package shlex

import "testing"

func TestLua_Split(t *testing.T) {
	tokens, err := Split(`argmatcher 'don\'t fail'`, Lua)
	if err != nil {
		t.Fatal(err)
	}
	if actual, expected := tokens.Words().Strings(), []string{"argmatcher", "don't fail"}; !stringSlicesEqual(actual, expected) {
		t.Errorf("Words()\nactual  : %#v\nexpected: %#v", actual, expected)
	}
	if actual, expected := tokens[len(tokens)-1].State, IN_WORD_STATE; actual != expected {
		t.Errorf("State\nactual  : %v\nexpected: %v", actual, expected)
	}

	tokens, err = Split("a\fb c", Lua)
	if err != nil {
		t.Fatal(err)
	}
	if actual, expected := tokens.Words().Strings(), []string{"a", "b", "c"}; !stringSlicesEqual(actual, expected) {
		t.Errorf("Words() with formfeed\nactual  : %#v\nexpected: %#v", actual, expected)
	}

	tokens, err = Split(`argmatcher git :addarg(x, checkpoint)`, Lua)
	if err != nil {
		t.Fatal(err)
	}
	if actual, expected := tokens.Words().Strings(), []string{"argmatcher", "git", ":addarg(x,", "checkpoint)"}; !stringSlicesEqual(actual, expected) {
		t.Errorf("Words()\nactual  : %#v\nexpected: %#v", actual, expected)
	}
	expectedWordbreakType := map[string]WordbreakType{
		"(": WORDBREAK_BRACKET,
		")": WORDBREAK_BRACKET,
		",": WORDBREAK_UNKNOWN,
	}
	for _, token := range tokens {
		if token.Type != WORDBREAK_TOKEN {
			continue
		}
		if expected, ok := expectedWordbreakType[token.Value]; ok && token.WordbreakType != expected {
			t.Errorf("WordbreakType of %q\nactual  : %v\nexpected: %v", token.Value, token.WordbreakType, expected)
		}
	}
}

func TestLua_Complete(t *testing.T) {
	ctx := Complete(`argmatcher git :addarg(checkpoint`, Lua)
	if actual, expected := ctx.Words, []string{"argmatcher", "git", ":addarg(checkpoint"}; !stringSlicesEqual(actual, expected) {
		t.Errorf("Words()\nactual  : %#v\nexpected: %#v", actual, expected)
	}
	if actual, expected := ctx.CurrentWord, ":addarg(checkpoint"; actual != expected {
		t.Errorf("CurrentWord\nactual  : %q\nexpected: %q", actual, expected)
	}
}

func TestJoin_Lua(t *testing.T) {
	tests := map[string][]string{
		`argmatcher git`:  {"argmatcher", "git"},
		`x "a b"`:         {"x", "a b"},
		`x "a\"b"`:        {"x", `a"b`},
		`x "a\\b"`:        {"x", `a\b`},
		`x "a$(touch x)"`: {"x", "a$(touch x)"},
		`x "a\nb"`:        {"x", "a\nb"},
		`x "123abc"`:      {"x", "123abc"},
		`x "café"`:        {"x", "café"},
	}
	for expected, words := range tests {
		if actual := Join(words, Lua); actual != expected {
			t.Errorf("Join(lua)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
