package shlex

import "testing"

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
		{name: "lua newline", format: Lua, input: "a\nb", expected: `a\nb`},
		{name: "bash newline", format: Bash, input: "a\nb", expected: "a\"'\n'\"b"},
		{name: "tcsh newline", format: Tcsh, input: "a\nb", expected: "a\\\nb"},
		{name: "fish newline stays raw", format: Fish, input: "a\nb", expected: "a\nb"},
		{name: "xonsh newline escape", format: Xonsh, input: "a\nb", expected: `a\nb`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Escape(tt.input, tt.format); got != tt.expected {
				t.Errorf("Escape(%#v, %v) = %#v, want %#v", tt.input, tt.format, got, tt.expected)
			}
		})
	}
}
