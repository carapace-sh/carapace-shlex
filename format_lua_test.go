package shlex

import "testing"

func TestJoin_Lua(t *testing.T) {
	tests := map[string][]string{
		`argmatcher git`:  {"argmatcher", "git"},
		`x "a b"`:         {"x", "a b"},
		`x "a\"b"`:        {"x", `a"b`},
		`x "a\\b"`:        {"x", `a\b`},
		`x "a$(touch x)"`: {"x", "a$(touch x)"},
	}
	for expected, words := range tests {
		if actual := Join(words, Lua); actual != expected {
			t.Errorf("Join(lua)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		name     string
		format   Format
		input    string
		expected string
	}{
		{name: "posix dollar/quote", format: Bash, input: "a$()b'c", expected: `a\$()b'c`},
		{name: "posix doublequote/backslash", format: Bash, input: "a\"b\\c", expected: `a\"b\\c`},
		{name: "posix fallback for unknown", format: "no-such-format", input: "a$(touch x)", expected: `a\$(touch x)`},
		{name: "posix fallback for cmd", format: Cmd, input: `a"b`, expected: `a\"b`},
		{name: "fish", format: Fish, input: "a$()b\"c\\d", expected: `a\$()b\"c\\d`},
		{name: "elvish", format: Elvish, input: "a'b", expected: `a''b`},
		{name: "powershell", format: Powershell, input: `demo'; IEX 'x`, expected: `demo''; IEX ''x`},
		{name: "xonsh", format: Xonsh, input: `demo'; touch 'x`, expected: `demo\'; touch \'x`},
		{name: "lua", format: Lua, input: `demo'; touch 'x`, expected: `demo\'; touch \'x`},
		{name: "lua passes metacharacters", format: Lua, input: "a$(touch x)", expected: "a$(touch x)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Escape(tt.input, tt.format); got != tt.expected {
				t.Errorf("Escape(%#v, %v) = %#v, want %#v", tt.input, tt.format, got, tt.expected)
			}
		})
	}
}
