package shlex

import "testing"

func TestVariableRef(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		format    Format
		wantName  string
		wantBrace bool
		wantRef   bool
	}{
		{name: "empty input", input: "", format: Bash},
		{name: "no dollar", input: "echo text", format: Bash},
		{name: "dollar only", input: "echo $", format: Bash, wantName: "", wantRef: true},
		{name: "dollar with name", input: "echo $HO", format: Bash, wantName: "HO", wantRef: true},
		{name: "brace only", input: "echo ${", format: Bash, wantName: "", wantBrace: true, wantRef: true},
		{name: "brace with name", input: "echo ${HO", format: Bash, wantName: "HO", wantBrace: true, wantRef: true},
		{
			name: "quoted text then dollar", input: `echo "text$HO`,
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "quoted text then brace", input: `echo "test${`,
			format: Bash, wantName: "", wantBrace: true, wantRef: true,
		},
		{
			// the lexer word is `text $HO`; expansion semantics are the
			// same as the unsplit case
			name: "quoted text with space", input: `echo "text $HO`,
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "single quotes keep dollar literal", input: `echo 'text$HO`,
			format: Bash,
		},
		{
			name: "closed quote then dollar", input: `echo "text"$HO`,
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "closed brace is not a reference", input: `echo ${HOME}`,
			format: Bash,
		},
		{
			name: "dollar before non-name char", input: `echo a$-x`,
			format: Bash,
		},
		{
			name: "escaped dollar is literal", input: `echo "\$HO`,
			format: Bash,
		},
		{
			name: "escaped backslash then dollar", input: `echo "\\$HO`,
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "command substitution is not a variable", input: `echo $(`,
			format: Bash,
		},
		{
			name: "fish has no brace form", input: "echo ${",
			format: Fish,
		},
		{
			name: "fish detects variables", input: "echo $HO",
			format: Fish, wantName: "HO", wantRef: true,
		},
		{
			name: "fish quoted dollar expands", input: `echo "text$HO`,
			format: Fish, wantName: "HO", wantRef: true,
		},
		{
			name: "elvish detects variables", input: "echo $HO",
			format: Elvish, wantName: "HO", wantRef: true,
		},
		{
			// backslash is a literal bareword character in elvish:
			// `\$HO` is a backslash followed by the expansion
			name: "elvish bareword backslash does not escape", input: `echo \$HO`,
			format: Elvish, wantName: "HO", wantRef: true,
		},
		{
			name: "elvish quoted backslash escapes", input: `echo "\$HO`,
			format: Elvish,
		},
		{
			name: "elvish single quotes keep dollar literal", input: `echo 'text$HO`,
			format: Elvish,
		},
		{
			name: "nushell detects variables", input: "echo $HO",
			format: Nushell, wantName: "HO", wantRef: true,
		},
		{
			name: "nushell hyphenated name", input: "echo $my-var",
			format: Nushell, wantName: "my-var", wantRef: true,
		},
		{
			// nushell quotes are literal strings, not interpolations
			name: "nushell double quotes keep dollar literal", input: `echo "text$HO`,
			format: Nushell,
		},
		{
			name: "nushell single quotes keep dollar literal", input: `echo 'text$HO`,
			format: Nushell,
		},
		{
			name: "nushell has no brace form", input: "echo ${",
			format: Nushell,
		},
		{
			name: "xonsh detects variables", input: "echo $HO",
			format: Xonsh, wantName: "HO", wantRef: true,
		},
		{
			name: "xonsh brace form", input: "echo ${HO",
			format: Xonsh, wantName: "HO", wantBrace: true, wantRef: true,
		},
		{
			// xonsh strings are Python literals: `$` is literal everywhere
			name: "xonsh double quotes keep dollar literal", input: `echo "text$HO`,
			format: Xonsh,
		},
		{
			name: "xonsh triple quotes keep dollar literal", input: `echo '''text$HO'''`,
			format: Xonsh,
		},
		{
			name: "xonsh command substitution is not a variable", input: `echo $(`,
			format: Xonsh,
		},
		{
			name: "tcsh detects variables", input: "echo $HO",
			format: Tcsh, wantName: "HO", wantRef: true,
		},
		{
			name: "tcsh brace form", input: "echo ${HO",
			format: Tcsh, wantName: "HO", wantBrace: true, wantRef: true,
		},
		{
			name: "tcsh single quotes keep dollar literal", input: `echo 'text$HO`,
			format: Tcsh,
		},
		{
			name: "zsh quoted", input: `echo "text$HO`,
			format: Zsh, wantName: "HO", wantRef: true,
		},
		{
			name: "redirect target", input: "echo >$HO",
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "pid parameter then reference", input: "echo $$$HO",
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "positional parameter", input: "echo $1",
			format: Bash,
		},
		{
			name: "positional parameter plus literal", input: "echo $9x",
			format: Bash,
		},
		{
			name: "brace positional parameter", input: "echo ${1",
			format: Bash,
		},
		{
			name: "positional parameter then reference", input: "echo $1x$HO",
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "inside substitution uses absolute span", input: "echo $(echo $HO",
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			// `#` mid-word is not a bash wordbreak; the reference is
			// still detected and the whole word is replaced
			name: "hash mid-word", input: "echo a#c$HO",
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "pid parameter is not a reference", input: "echo $$",
			format: Bash,
		},
		{
			name: "pid parameter mid-word is not a reference", input: "echo a$$",
			format: Bash,
		},
		{
			name: "pid parameter then reference", input: "echo $$$HO",
			format: Bash, wantName: "HO", wantRef: true,
		},
		{
			name: "closed brace then reference", input: `echo ${HO}x$NE`,
			format: Bash, wantName: "NE", wantRef: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Complete(tt.input, tt.format)
			if tt.wantRef {
				if ctx.Variable == nil {
					t.Fatalf("Variable = nil, want Name=%q Brace=%v", tt.wantName, tt.wantBrace)
				}
				if ctx.Variable.Name != tt.wantName || ctx.Variable.brace != tt.wantBrace {
					t.Errorf("Variable = %+v, want Name=%q Brace=%v", *ctx.Variable, tt.wantName, tt.wantBrace)
				}
				return
			}
			if ctx.Variable != nil {
				t.Errorf("Variable = %+v, want nil", *ctx.Variable)
			}
		})
	}
}

func TestVariableReplacement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// bash's naive COMP_WORDS split keeps quoted text before a
		// wordbreak in the replaced word, but not text after it
		{name: "quoted word", input: `echo "text$HO`, want: "text$HO"},
		{name: "quoted word with space", input: `echo "text $HO`, want: "$HO"},
		{name: "plain word", input: "echo $HO", want: "$HO"},
		{name: "brace form", input: `echo "test${HO`, want: "test${HO"},
		{name: "hash mid-word is not a wordbreak", input: "echo a#c$HO", want: "a#c$HO"},
		{name: "redirect target", input: "echo >$HO", want: "$HO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Complete(tt.input, Bash)
			if ctx.Variable == nil {
				t.Fatalf("Variable = nil, want Replacement %q", tt.want)
			}
			if ctx.Variable.replacement != tt.want {
				t.Errorf("Variable.replacement = %q, want %q", ctx.Variable.replacement, tt.want)
			}
		})
	}
}

func TestZshKeepsWholeRawWord(t *testing.T) {
	ctx := Complete(`echo "text $HO`, Zsh)
	if ctx.Variable == nil {
		t.Fatal("Variable = nil")
	}
	if ctx.Variable.replacement != ctx.RawCurrentWord {
		t.Errorf("Variable.replacement = %q, want RawCurrentWord %q", ctx.Variable.replacement, ctx.RawCurrentWord)
	}
}

func TestVariableInsert(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		format Format
		want   string
	}{
		{name: "plain form", input: "echo $HO", format: Bash, want: "$HOME"},
		{name: "quoted word", input: `echo "text$HO`, format: Bash, want: "text$HOME"},
		{name: "quoted text with space", input: `echo "text $HO`, format: Bash, want: "$HOME"},
		{name: "brace form closes brace", input: `echo "test${HO`, format: Bash, want: "test${HOME}"},
		{name: "brace only", input: "echo ${", format: Bash, want: "${HOME}"},
		{name: "trailing sigil", input: "echo $", format: Bash, want: "$HOME"},
		{name: "hyphenated nushell name", input: "echo $my-var", format: Nushell, want: "$HOME"},
		{name: "elvish keeps bareword backslash", input: `echo \$HO`, format: Elvish, want: `\$HOME`},
		{name: "redirect target", input: "echo >$HO", format: Bash, want: "$HOME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Complete(tt.input, tt.format)
			if ctx.Variable == nil {
				t.Fatalf("Variable = nil")
			}
			if got := ctx.Variable.Insert("HOME"); got != tt.want {
				t.Errorf("Insert = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVariableSpan(t *testing.T) {
	ctx := Complete(`echo "text $HO`, Bash)
	if ctx.Variable == nil {
		t.Fatal("Variable = nil")
	}
	// the naive replacement region `$HO` ends at the word's end
	if got := ctx.Variable.Span(); got != (Span{Start: ctx.Span.End - 3, End: ctx.Span.End}) {
		t.Errorf("Span = %v, want {%d %d}", got, ctx.Span.End-3, ctx.Span.End)
	}

	// the naive replacement region excludes the opening quote
	ctx = Complete(`echo "text$HO`, Bash)
	if got := ctx.Variable.Span(); got != (Span{Start: ctx.Span.End - 7, End: ctx.Span.End}) {
		t.Errorf("Span = %v, want {%d %d}", got, ctx.Span.End-7, ctx.Span.End)
	}
}
