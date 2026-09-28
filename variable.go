package shlex

// VariableRef describes a variable reference ending the current word.
type VariableRef struct {
	// Name is the variable name typed so far. Empty when the cursor is
	// directly after the sigil.
	Name string

	// Brace reports the `${` form (as opposed to the plain `$` form).
	Brace bool

	// Span is the rune span of the `$`/`${` opener in the input.
	Span Span
}

// VariableExpander is implemented by formats whose variable references can
// be detected lexically. The detection runs on the lexer's final word, so
// quote and escape state are authoritative: a `$` inside single quotes is
// literal, an escaped `$` is literal, and the text following the last
// unescaped `$` must be a valid variable-name prefix for the reference to
// be detected (which keeps closed expansions like `${HOME}` and command
// substitutions like `$(...)` out of variable completion).
//
// Formats without variable expansion (or with forms this detection does not
// cover) simply do not implement the interface.
type VariableExpander interface {
	VariableRef(word Token) (VariableRef, bool)
}

// NaiveWordSplitter is implemented by formats whose completion interface
// splits words naively on delimiter characters instead of lexically (bash's
// COMP_WORDS model). For those shells the text the shell replaces on
// insertion is only the raw text after the last delimiter character, which
// may differ from the lexer's word (`"text $HO` is one lexer word but
// completes as `$HO`).
type NaiveWordSplitter interface {
	NaiveSplitWord(raw string) string
}

// posixVariableRef detects a variable reference ending the given word using
// POSIX expansion rules: `$name` and `${name`, backslash escapes, and no
// expansion inside single quotes.
func posixVariableRef(word Token) (VariableRef, bool) {
	if word.State == QUOTING_STATE { // single quotes: `$` is literal
		return VariableRef{}, false
	}

	runes := []rune(word.RawValue)
	i := -1
	for j := len(runes) - 1; j >= 0; j-- {
		if runes[j] == '$' {
			i = j
			break
		}
	}
	if i < 0 {
		return VariableRef{}, false
	}

	escaped := false
	for k := i; k > 0 && runes[k-1] == '\\'; k-- {
		escaped = !escaped
	}
	if escaped {
		return VariableRef{}, false
	}

	rest := runes[i+1:]
	brace := len(rest) > 0 && rest[0] == '{'
	if brace {
		rest = rest[1:]
	}
	for _, r := range rest {
		if !isPosixNameRune(r) {
			return VariableRef{}, false
		}
	}

	span := word.Span
	start := span.Start + i
	end := start + 1
	if brace {
		end++
	}
	return VariableRef{
		Name:  string(rest),
		Brace: brace,
		Span:  Span{Start: start, End: end},
	}, true
}

func isPosixNameRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		return true
	default:
		return false
	}
}

// naiveSplitWord returns the raw text of raw after its last delimiter
// character, where delimiters are every rune the classifier treats as
// space, wordbreak, or quote. The escape character is not a delimiter,
// matching bash's COMP_WORDS splitting; a mid-word comment rune is not
// either (the classifier marks `#` unconditionally, but bash only starts
// a comment at a word boundary).
func naiveSplitWord(classifier tokenClassifier, raw string) string {
	runes := []rune(raw)
	for i := len(runes) - 1; i >= 0; i-- {
		switch classifier.ClassifyRune(runes[i]) {
		case spaceRuneClass, wordbreakRuneClass,
			escapingQuoteRuneClass, nonEscapingQuoteRuneClass:
			return string(runes[i+1:])
		}
	}
	return raw
}
