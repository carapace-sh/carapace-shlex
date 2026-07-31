package shlex

import "encoding/json"

// SubstitutionKind classifies the type of substitution scope.
type SubstitutionKind int

const (
	// SubstitutionCommand is $(...), (...), or similar command/output capture.
	SubstitutionCommand SubstitutionKind = iota
	// SubstitutionArithmetic is $((...)).
	SubstitutionArithmetic
	// SubstitutionBacktick is `...` (POSIX backtick command substitution).
	SubstitutionBacktick
)

var substitutionKinds = map[SubstitutionKind]string{
	SubstitutionCommand:    "SubstitutionCommand",
	SubstitutionArithmetic: "SubstitutionArithmetic",
	SubstitutionBacktick:   "SubstitutionBacktick",
}

func (k SubstitutionKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(substitutionKinds[k])
}

// SubstitutionScope describes a single substitution nesting level in the
// token stream.
type SubstitutionScope struct {
	OpenIndex  int // TokenSlice index of the opener token
	CloseIndex int // TokenSlice index of the closer token, or -1 if unclosed
	Kind       SubstitutionKind
	Depth      int // nesting depth at this scope (1 = outermost)
}

// SubstitutionScopes returns all substitution scopes in the token slice,
// ordered by open position. Unclosed scopes (cursor inside) have
// CloseIndex == -1. Backtick scopes are detected by scanning WORD_TOKEN
// RawValue for unescaped backticks and have OpenIndex == -1 (since there
// is no single opener token in the stream).
func (t TokenSlice) SubstitutionScopes() []SubstitutionScope {
	var scopes []SubstitutionScope
	var stack []SubstitutionScope

	for i, token := range t {
		switch {
		case token.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN:
			kind := SubstitutionCommand
			// Detect arithmetic $(( : the opener RawValue contains two '('
			if len(token.RawValue) >= 2 && token.RawValue[len(token.RawValue)-2] == '(' {
				kind = SubstitutionArithmetic
			}
			scope := SubstitutionScope{
				OpenIndex:  i,
				CloseIndex: -1,
				Kind:       kind,
				Depth:      len(stack) + 1,
			}
			stack = append(stack, scope)

		case token.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE:
			if len(stack) > 0 {
				scope := &stack[len(stack)-1]
				scope.CloseIndex = i
				scopes = append(scopes, *scope)
				stack = stack[:len(stack)-1]
			}
		}
	}

	// Any unclosed scopes remain on the stack
	for j := range stack {
		s := stack[j]
		scopes = append(scopes, s)
	}

	// Scan for backtick substitution in WORD_TOKEN RawValues.
	// This is detect-only: backtick content is already merged into
	// word tokens by the tokenizer, so we can't extract inner tokens.
	scopes = append(scopes, t.detectBacktickScopes()...)

	return scopes
}

// detectBacktickScopes scans WORD_TOKENs for unescaped backticks.
// An odd count of backticks in a single word's RawValue indicates an
// unclosed backtick substitution starting at that word. The scope has
// OpenIndex == -1 (no single opener token) and Kind == SubstitutionBacktick.
func (t TokenSlice) detectBacktickScopes() []SubstitutionScope {
	var scopes []SubstitutionScope
	for i, token := range t {
		if token.Type != WORD_TOKEN {
			continue
		}
		count := countUnescapedBackticks(token.RawValue)
		if count%2 == 1 {
			scopes = append(scopes, SubstitutionScope{
				OpenIndex:  -1,
				CloseIndex: -1,
				Kind:       SubstitutionBacktick,
				Depth:      1,
			})
			_ = i // index kept for potential future use
		}
	}
	return scopes
}

// countUnescapedBackticks counts backtick characters in s that are not
// preceded by an escape character (backslash).
func countUnescapedBackticks(s string) int {
	count := 0
	runes := []rune(s)
	for i, r := range runes {
		if r == '`' {
			if i > 0 && runes[i-1] == '\\' {
				continue // escaped backtick
			}
			count++
		}
	}
	return count
}

// innermostUnclosedScope returns the unclosed substitution scope with the
// greatest depth, or nil if all scopes are closed. When the cursor is
// inside a substitution, this is the scope whose content forms the inner
// completion context.
func innermostUnclosedScope(scopes []SubstitutionScope) *SubstitutionScope {
	var result *SubstitutionScope
	for i := range scopes {
		s := &scopes[i]
		if s.CloseIndex == -1 {
			if result == nil || s.Depth > result.Depth {
				result = s
			}
		}
	}
	return result
}

// countUnclosedScopes returns the number of scopes with CloseIndex == -1.
func countUnclosedScopes(scopes []SubstitutionScope) int {
	count := 0
	for _, s := range scopes {
		if s.CloseIndex == -1 {
			count++
		}
	}
	return count
}
