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
		wantSpan  bool
		wantStart int // rune offset of the opener in the input
		wantEnd   int
	}{
		{name: "empty input", input: "", format: Bash},
		{name: "no dollar", input: "echo text", format: Bash},
		{name: "dollar only", input: "echo $", format: Bash, wantName: "", wantRef: true},
		{name: "dollar with name", input: "echo $HO", format: Bash, wantName: "HO", wantRef: true, wantSpan: true, wantStart: 5, wantEnd: 6},
		{name: "brace only", input: "echo ${", format: Bash, wantName: "", wantBrace: true, wantRef: true},
		{name: "brace with name", input: "echo ${HO", format: Bash, wantName: "HO", wantBrace: true, wantRef: true, wantSpan: true, wantStart: 5, wantEnd: 7},
		{
			name: "quoted text then dollar", input: `echo "text$HO`,
			format: Bash, wantName: "HO", wantRef: true, wantSpan: true, wantStart: 10, wantEnd: 11,
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
			name: "fish without expander", input: "echo $HO",
			format: Fish,
		},
		{
			name: "zsh quoted", input: `echo "text$HO`,
			format: Zsh, wantName: "HO", wantRef: true,
		},
		{
			name: "redirect target", input: "echo >$HO",
			format: Bash, wantName: "HO", wantRef: true, wantSpan: true, wantStart: 6, wantEnd: 7,
		},
		{
			name: "pid parameter then reference", input: "echo $$$HO",
			format: Bash, wantName: "HO", wantRef: true, wantSpan: true, wantStart: 7, wantEnd: 8,
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
			format: Bash, wantName: "HO", wantRef: true, wantSpan: true, wantStart: 12, wantEnd: 13,
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
				if ctx.VariableRef == nil {
					t.Fatalf("VariableRef = nil, want Name=%q Brace=%v", tt.wantName, tt.wantBrace)
				}
				if ctx.VariableRef.Name != tt.wantName || ctx.VariableRef.Brace != tt.wantBrace {
					t.Errorf("VariableRef = %+v, want Name=%q Brace=%v", *ctx.VariableRef, tt.wantName, tt.wantBrace)
				}
				if tt.wantSpan {
					span := ctx.VariableRef.Span
					if span.Start != tt.wantStart || span.End != tt.wantEnd {
						t.Errorf("VariableRef.Span = %+v, want {%d %d}", span, tt.wantStart, tt.wantEnd)
					}
				}
				return
			}
			if ctx.VariableRef != nil {
				t.Errorf("VariableRef = %+v, want nil", *ctx.VariableRef)
			}
		})
	}
}

func TestRawReplacementWord(t *testing.T) {
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
		{name: "single quoted word", input: `echo 'text$HO`, want: "text$HO"},
		{name: "hash mid-word is not a wordbreak", input: "echo a#c$HO", want: "a#c$HO"},
		{name: "redirect target", input: "echo >$HO", want: "$HO"},
		{name: "wordbreak suffix is empty", input: "echo $@", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Complete(tt.input, Bash)
			if ctx.RawReplacementWord != tt.want {
				t.Errorf("RawReplacementWord = %q, want %q", ctx.RawReplacementWord, tt.want)
			}
		})
	}
}

func TestZshKeepsWholeRawWord(t *testing.T) {
	ctx := Complete(`echo "text $HO`, Zsh)
	if ctx.RawReplacementWord != ctx.RawCurrentWord {
		t.Errorf("RawReplacementWord = %q, want RawCurrentWord %q", ctx.RawReplacementWord, ctx.RawCurrentWord)
	}
}
