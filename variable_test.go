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
		{name: "empty input", input: "", format: BashFormat()},
		{name: "no dollar", input: "echo text", format: BashFormat()},
		{name: "dollar only", input: "echo $", format: BashFormat(), wantName: "", wantRef: true},
		{name: "dollar with name", input: "echo $HO", format: BashFormat(), wantName: "HO", wantRef: true},
		{name: "brace only", input: "echo ${", format: BashFormat(), wantName: "", wantBrace: true, wantRef: true},
		{name: "brace with name", input: "echo ${HO", format: BashFormat(), wantName: "HO", wantBrace: true, wantRef: true},
		{
			name: "quoted text then dollar", input: `echo "text$HO`,
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "quoted text then brace", input: `echo "test${`,
			format: BashFormat(), wantName: "", wantBrace: true, wantRef: true,
		},
		{
			// the lexer word is `text $HO`; expansion semantics are the
			// same as the unsplit case
			name: "quoted text with space", input: `echo "text $HO`,
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "single quotes keep dollar literal", input: `echo 'text$HO`,
			format: BashFormat(),
		},
		{
			name: "closed quote then dollar", input: `echo "text"$HO`,
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "closed brace is not a reference", input: `echo ${HOME}`,
			format: BashFormat(),
		},
		{
			name: "dollar before non-name char", input: `echo a$-x`,
			format: BashFormat(),
		},
		{
			name: "escaped dollar is literal", input: `echo "\$HO`,
			format: BashFormat(),
		},
		{
			name: "escaped backslash then dollar", input: `echo "\\$HO`,
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "command substitution is not a variable", input: `echo $(`,
			format: BashFormat(),
		},
		{
			name: "fish has no brace form", input: "echo ${",
			format: FishFormat(),
		},
		{
			name: "fish without expander", input: "echo $HO",
			format: FishFormat(),
		},
		{
			name: "zsh quoted", input: `echo "text$HO`,
			format: ZshFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "redirect target", input: "echo >$HO",
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			// `#` mid-word is not a bash wordbreak; the reference is
			// still detected and the whole word is replaced
			name: "hash mid-word", input: "echo a#c$HO",
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "pid parameter is not a reference", input: "echo $$",
			format: BashFormat(),
		},
		{
			name: "pid parameter mid-word is not a reference", input: "echo a$$",
			format: BashFormat(),
		},
		{
			name: "pid parameter then reference", input: "echo $$$HO",
			format: BashFormat(), wantName: "HO", wantRef: true,
		},
		{
			name: "closed brace then reference", input: `echo ${HO}x$NE`,
			format: BashFormat(), wantName: "NE", wantRef: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SplitForCompletion(tt.input, tt.format)
			if tt.wantRef {
				if ctx.VariableRef == nil {
					t.Fatalf("VariableRef = nil, want Name=%q Brace=%v", tt.wantName, tt.wantBrace)
				}
				if ctx.VariableRef.Name != tt.wantName || ctx.VariableRef.Brace != tt.wantBrace {
					t.Errorf("VariableRef = %+v, want Name=%q Brace=%v", *ctx.VariableRef, tt.wantName, tt.wantBrace)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SplitForCompletion(tt.input, BashFormat())
			if ctx.RawReplacementWord != tt.want {
				t.Errorf("RawReplacementWord = %q, want %q", ctx.RawReplacementWord, tt.want)
			}
		})
	}
}

func TestZshKeepsWholeRawWord(t *testing.T) {
	ctx := SplitForCompletion(`echo "text $HO`, ZshFormat())
	if ctx.RawReplacementWord != ctx.RawCurrentWord {
		t.Errorf("RawReplacementWord = %q, want RawCurrentWord %q", ctx.RawReplacementWord, ctx.RawCurrentWord)
	}
}
