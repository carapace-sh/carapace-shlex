package shlex

// innermostUnclosedCommandScope returns the TokenSlice index of the opener
// of the innermost unclosed command substitution scope, or -1 if the cursor
// is at top level. Arithmetic ($((...))) and backtick scopes are not
// command scopes — they don't produce an inner completion context.
func innermostUnclosedCommandScope(tokens TokenSlice) int {
	depth := 0
	lastOpen := -1

	for i, t := range tokens {
		switch t.WordbreakType {
		case WORDBREAK_SUBSTITUTION_OPEN:
			if isArithmeticOpener(t) {
				continue
			}
			depth++
			lastOpen = i
		case WORDBREAK_SUBSTITUTION_CLOSE:
			if isArithmeticCloser(t) {
				continue
			}
			depth--
			if depth == 0 {
				lastOpen = -1
			}
		}
	}

	if depth > 0 {
		return lastOpen
	}
	return -1
}

func isArithmeticOpener(t Token) bool {
	return len(t.RawValue) >= 2 && t.RawValue[len(t.RawValue)-2] == '('
}

func isArithmeticCloser(t Token) bool {
	return len(t.RawValue) >= 2 && t.RawValue[0] == ')' && t.RawValue[1] == ')'
}
