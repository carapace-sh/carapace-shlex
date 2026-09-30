package shlex

import "testing"

func TestQuote(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		input  string
		value  string
		want   string
	}{
		{name: "close double quote", format: Bash, input: `echo "he`, value: "llo world", want: `"llo world"`},
		{name: "escape dollar inside double quotes", format: Bash, input: `echo "he`, value: "a$b", want: `"a\$b"`},
		{name: "close single quote posix", format: Bash, input: `echo 'he`, value: "a'b", want: `'a'"'"'b'`},
		{name: "close single quote fish", format: Fish, input: `echo 'he`, value: "a'b", want: `'a\'b'`},
		{name: "close single quote zsh", format: Zsh, input: `echo 'he`, value: "a'b", want: `'a''b'`},
		{name: "close single quote elvish", format: Elvish, input: `echo 'he`, value: "a'b", want: `'a''b'`},
		{name: "close double quote cmd", format: Cmd, input: `echo "he`, value: `a"b`, want: `"a""b"`},
		{name: "close double quote powershell", format: Powershell, input: `echo "he`, value: `a"b`, want: "\"a`\"b\""},
		{name: "bareword quoted as complete word", format: Bash, input: "echo he", value: "a b", want: `"a b"`},
		{name: "bareword plain stays plain", format: Bash, input: "echo he", value: "plain", want: "plain"},
		{name: "bareword elvish single quotes", format: Elvish, input: "echo he", value: "a b", want: `'a b'`},
		{name: "stop parsing stays raw", format: Powershell, input: "echo --% x", value: "a|b", want: "a|b"},
		{name: "close triple single quote xonsh", format: Xonsh, input: `echo '''he`, value: "a'b", want: `'''a\'b'''`},
		{name: "close triple double quote xonsh", format: Xonsh, input: `echo """he`, value: `a"b`, want: `"""a\"b"""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Complete(tt.input, tt.format)
			if got := ctx.Quote(tt.value); got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
