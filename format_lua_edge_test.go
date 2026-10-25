package shlex

import (
	"strings"
	"testing"
)

// Edge cases derived from the Lua test suite (github.com/lua/tests,
// literals.lua "testing scanner") where they apply to the word-based
// lexer model.

func TestLua_EscapeSequencesAreLenient(t *testing.T) {
	// Lua's read_string errors on these; the model deliberately treats
	// any backslash as escaping the next rune and drops the backslash,
	// so the letters and digits below are literal.
	tests := map[string]string{
		`"a\099b"`:      "a099b",      // Lua: "a" + byte 99 + "b"
		`"a\x41b"`:      "ax41b",      // Lua: "aA b"... "aAb"
		`"a\u{65}b"`:    "au{65}b",    // Lua: "aeb"
		`"a\g"`:         "ag",         // Lua: invalid escape sequence
		`"a\999b"`:      "a999b",      // Lua: decimal escape too large
		`"abc\z   efg"`: "abcz   efg", // Lua: \z skips whitespace, "abcefg"
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			tokens, err := Split(`x `+input, Lua)
			if err != nil {
				t.Fatal(err)
			}
			words := tokens.Words().Strings()
			if len(words) != 2 || words[1] != expected {
				t.Errorf("Words\nactual  : %#v\nexpected: [x %#v]", words, expected)
			}
		})
	}
}

func TestLua_UnfinishedStringsStayQuoting(t *testing.T) {
	// literals.lua lexerror(..., "<eof>"); for completion the lexer keeps
	// the open quotation state instead.
	tests := map[string]LexerState{
		`x 'alo`: QUOTING_STATE,
		`x "alo`: QUOTING_ESCAPING_STATE,
	}
	for input, expected := range tests {
		t.Run(t.Name()+input, func(t *testing.T) {
			tokens, err := Split(input, Lua)
			if err != nil {
				t.Fatal(err)
			}
			words := tokens.Words().Strings()
			if len(words) != 2 || words[1] != "alo" {
				t.Errorf("Words\nactual  : %#v\nexpected: [x alo]", words)
			}
			if actual := tokens[len(tokens)-1].State; actual != expected {
				t.Errorf("State\nactual  : %v\nexpected: %v", actual, expected)
			}
		})
	}
}

func TestLua_LongBracketsSplitOnWhitespaceOnly(t *testing.T) {
	// Long strings and their bracket levels are not lexed; bracket text
	// without raw spaces stays one word (literals.lua long-string lexxing
	// over {=, [, ], newline} combinations becomes word splitting here).
	tests := map[string][]string{
		`x [=[al]o]]`:        {"x", "[=[al]o]]"},
		`x [==[a b]==]`:      {"x", "[==[a", "b]==]"},
		`x [===[[=[]]=]===]`: {"x", "[===[[=[]]=]===]"},
		`x [[alo] `:          {"x", "[[alo]", ""},
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			tokens, err := Split(input, Lua)
			if err != nil {
				t.Fatal(err)
			}
			if actual := tokens.Words().Strings(); !stringSlicesEqual(actual, expected) {
				t.Errorf("Words\nactual  : %#v\nexpected: %#v", actual, expected)
			}
		})
	}
}

func TestLua_LineEndsAsSeparators(t *testing.T) {
	// literals.lua tests \n, \r, \n\r, \r\n as end-of-line sequences.
	for _, sep := range []string{"\n", "\r", "\n\r", "\r\n"} {
		tokens, err := Split("a"+sep+"b"+sep+"c", Lua)
		if err != nil {
			t.Fatal(err)
		}
		if actual, expected := tokens.Words().Strings(), []string{"a", "b", "c"}; !stringSlicesEqual(actual, expected) {
			t.Errorf("Words with %q separator\nactual  : %#v\nexpected: %#v", sep, actual, expected)
		}
	}
}

func TestLua_NulRoundtrip(t *testing.T) {
	// literals.lua: 'a\0a' with embedded zero bytes.
	word := "a\x00a\x00b"
	tokens, err := Split(`"`+word+`"`, Lua)
	if err != nil {
		t.Fatal(err)
	}
	if actual := tokens.Words().Strings(); !stringSlicesEqual(actual, []string{word}) {
		t.Errorf("Words\nactual  : %#v\nexpected: [%#v]", actual, word)
	}
	if actual := Join([]string{word}, Lua); !strings.HasPrefix(actual, `"`) {
		t.Errorf("Join(%#v) = %#v, want double-quoted", word, actual)
	}
}

func TestLua_IdentifierCharsMatchLuaTables(t *testing.T) {
	// literals.lua scans all 256 byte values for identifier membership.
	// isLuaBareword follows lctype.h's fixed ASCII tables (lislalpha):
	// ASCII letters and underscore; digits cannot lead a name; control
	// chars and >0x7F bytes get quoted.
	for i := 0; i < 256; i++ {
		r := rune(i)
		expected := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if actual := isLuaBareword(string(r)); actual != expected {
			t.Errorf("isLuaBareword(%#v) = %v", string(r), expected)
		}
	}
}
