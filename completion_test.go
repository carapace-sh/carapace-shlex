package shlex

import "testing"

func TestSplitForCompletion(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		format       Format
		wantWord     string
		wantPrefix   string
		wantState    LexerState
		wantRedirect bool
		wantWords    []string
	}{
		{
			name:       "simple word",
			input:      "echo hel",
			format:     BashFormat(),
			wantWord:   "hel",
			wantPrefix: "",
			wantState:  IN_WORD_STATE,
			wantWords:  []string{"echo", "hel"},
		},
		{
			name:       "inside double quotes",
			input:      `echo "hel`,
			format:     BashFormat(),
			wantWord:   "hel",
			wantPrefix: "",
			wantState:  QUOTING_ESCAPING_STATE,
			wantWords:  []string{"echo", "hel"},
		},
		{
			name:       "inside single quotes",
			input:      "echo 'hel",
			format:     BashFormat(),
			wantWord:   "hel",
			wantPrefix: "",
			wantState:  QUOTING_STATE,
			wantWords:  []string{"echo", "hel"},
		},
		{
			name:       "pipeline",
			input:      "echo foo | grep bar",
			format:     BashFormat(),
			wantWord:   "bar",
			wantPrefix: "",
			wantState:  IN_WORD_STATE,
			wantWords:  []string{"grep", "bar"},
		},
		{
			name:         "redirect target",
			input:        "echo foo > bar",
			format:       BashFormat(),
			wantWord:     "bar",
			wantPrefix:   "",
			wantState:    IN_WORD_STATE,
			wantRedirect: true,
			wantWords:    []string{"echo", "foo"},
		},
		{
			name:       "wordbreak prefix with equals",
			input:      "echo foo=bar",
			format:     BashFormat(),
			wantWord:   "foo=bar",
			wantPrefix: "foo=",
			wantState:  IN_WORD_STATE,
			wantWords:  []string{"echo", "foo=bar"},
		},
		{
			name:       "empty input",
			input:      "",
			format:     BashFormat(),
			wantWord:   "",
			wantPrefix: "",
			wantState:  START_STATE,
			wantWords:  []string{""},
		},
		{
			name:       "escape at end",
			input:      `echo foo\`,
			format:     BashFormat(),
			wantWord:   "foo",
			wantPrefix: "",
			wantState:  ESCAPING_STATE,
			wantWords:  []string{"echo", "foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := SplitForCompletion(tt.input, tt.format)
			if ctx.CurrentWord != tt.wantWord {
				t.Errorf("CurrentWord = %q, want %q", ctx.CurrentWord, tt.wantWord)
			}
			if ctx.Prefix != tt.wantPrefix {
				t.Errorf("Prefix = %q, want %q", ctx.Prefix, tt.wantPrefix)
			}
			if ctx.QuotingState != tt.wantState {
				t.Errorf("QuotingState = %v, want %v", ctx.QuotingState, tt.wantState)
			}
			if ctx.IsRedirect != tt.wantRedirect {
				t.Errorf("IsRedirect = %v, want %v", ctx.IsRedirect, tt.wantRedirect)
			}
			if len(ctx.Words) != len(tt.wantWords) {
				t.Errorf("Words = %v, want %v", ctx.Words, tt.wantWords)
			} else {
				for i, w := range tt.wantWords {
					if ctx.Words[i] != w {
						t.Errorf("Words[%d] = %q, want %q", i, ctx.Words[i], w)
					}
				}
			}
		})
	}
}
