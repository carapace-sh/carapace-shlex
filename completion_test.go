package shlex

import "testing"

func TestComplete(t *testing.T) {
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
			format:     Bash,
			wantWord:   "hel",
			wantPrefix: "",
			wantState:  IN_WORD_STATE,
			wantWords:  []string{"echo", "hel"},
		},
		{
			name:       "inside double quotes",
			input:      `echo "hel`,
			format:     Bash,
			wantWord:   "hel",
			wantPrefix: "",
			wantState:  QUOTING_ESCAPING_STATE,
			wantWords:  []string{"echo", "hel"},
		},
		{
			name:       "inside single quotes",
			input:      "echo 'hel",
			format:     Bash,
			wantWord:   "hel",
			wantPrefix: "",
			wantState:  QUOTING_STATE,
			wantWords:  []string{"echo", "hel"},
		},
		{
			name:       "pipeline",
			input:      "echo foo | grep bar",
			format:     Bash,
			wantWord:   "bar",
			wantPrefix: "",
			wantState:  IN_WORD_STATE,
			wantWords:  []string{"grep", "bar"},
		},
		{
			name:         "redirect target",
			input:        "echo foo > bar",
			format:       Bash,
			wantWord:     "bar",
			wantPrefix:   "",
			wantState:    IN_WORD_STATE,
			wantRedirect: true,
			wantWords:    []string{"echo", "foo"},
		},
		{
			name:       "wordbreak prefix with equals",
			input:      "echo foo=bar",
			format:     Bash,
			wantWord:   "foo=bar",
			wantPrefix: "foo=",
			wantState:  IN_WORD_STATE,
			wantWords:  []string{"echo", "foo=bar"},
		},
		{
			name:       "empty input",
			input:      "",
			format:     Bash,
			wantWord:   "",
			wantPrefix: "",
			wantState:  START_STATE,
			wantWords:  []string{""},
		},
		{
			name:       "escape at end",
			input:      `echo foo\`,
			format:     Bash,
			wantWord:   "foo",
			wantPrefix: "",
			wantState:  ESCAPING_STATE,
			wantWords:  []string{"echo", "foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Complete(tt.input, tt.format)
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

func TestUnknownFormat(t *testing.T) {
	if _, err := Split("echo hi", "nosuch"); err == nil {
		t.Error("Split with unknown format = nil error, want error")
	}

	ctx := Complete("echo hi", "nosuch")
	if ctx.QuotingState != START_STATE {
		t.Errorf("QuotingState = %v, want START_STATE", ctx.QuotingState)
	}
	if len(ctx.Words) != 0 || ctx.Tokens != nil {
		t.Errorf("Complete with unknown format = %+v, want empty context", ctx)
	}
}
