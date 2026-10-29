package shlex

import (
	"slices"
	"testing"
)

func TestQuotingPreservesNewline(t *testing.T) {
	for _, format := range []Format{Default, Bash, Zsh, Oil, Tcsh, Fish} {
		t.Run(string(format), func(t *testing.T) {
			joined := Join([]string{"words", "a\nb"}, format)
			tokens, err := Split(joined, format)
			if err != nil {
				t.Fatalf("Split(%q): %v", joined, err)
			}
			words := make([]string, 0, len(tokens))
			for _, token := range tokens {
				if token.Type == WORD_TOKEN {
					words = append(words, token.Value)
				}
			}
			if want := []string{"words", "a\nb"}; !slices.Equal(words, want) {
				t.Errorf("format %v: Split(Join) = %#v, want %#v (joined=%q)", format, words, want, joined)
			}
		})
	}
}
