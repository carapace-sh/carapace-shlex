package shlex

// Variable describes a variable reference ending the current word.
type Variable struct {
	// Name is the variable name typed so far. Empty when the cursor is
	// directly after the sigil.
	Name string

	brace bool // `${` form (as opposed to the plain `$` form)

	// replacement is the raw text the shell's completion interface
	// replaces on insertion, ending in the sigil and the typed name. It
	// equals the raw current word unless the format has a naive word
	// interface (bash: the COMP_WORDS suffix, so `"text $HO` yields
	// `$HO`); shells whose completion API resolves quoting themselves
	// receive the whole raw word and strip the prefix.
	replacement string

	// span is the absolute rune span of the replaced region.
	span Span
}

// Insert returns the replacement word with the completed variable
// reference in place, preserving the reference's form (`$HOME` or
// `${HOME}`). Consumers prefix it with everything before the replaced
// region.
// Span returns the rune span of the replaced region in the input — the
// text Insert replaces (the whole raw word, or its naive suffix for
// formats with a naive word interface).
func (v Variable) Span() Span {
	return v.span
}

func (v Variable) Insert(completed string) string {
	sigil, closer := "$", ""
	if v.brace {
		sigil, closer = "${", "}"
	}
	return v.replacement[:len(v.replacement)-len(v.Name)-len(sigil)] + sigil + completed + closer
}

// variableExpander is implemented by formats whose variable references can
// be detected lexically. The detection runs on the lexer's final word, so
// quote and escape state are authoritative: a `$` inside single quotes is
// literal, an escaped `$` is literal, and a reference ends the word only
// when the expansion containing the last `$` is still open at the end of
// the word (closed expansions like `${HOME}`, special parameters like
// `$$`, and positional parameters like `$1x` do not count).
//
// Formats without variable expansion (or with forms this detection does not
// cover) simply do not implement the interface.
type variableExpander interface {
	Variable(word Token) (Variable, bool)
}

// naiveWordSplitter is implemented by formats whose completion interface
// splits words naively on delimiter characters instead of lexically (bash's
// COMP_WORDS model). For those shells the text the shell replaces on
// insertion is only the raw text after the last delimiter character, which
// may differ from the lexer's word (`"text $HO` is one lexer word but
// completes as `$HO`).
type naiveWordSplitter interface {
	NaiveSplitWord(raw string) string
}

// variableRules describes a format's variable expansion: which sigil
// forms exist, which states expand, and how names are spelled.
type variableRules struct {
	// brace reports the `${name` form (bash and zsh have it; fish and
	// elvish do not).
	brace bool

	// expands reports whether `$` expands in the given lexer state
	// (nushell and xonsh expand only in barewords).
	expands func(state LexerState) bool

	// barewordEscapes reports whether the escape character escapes `$` in
	// barewords. When false (elvish: backslash is a literal bareword
	// character), escapes are only honored inside escaping quotes.
	barewordEscapes bool

	// nameStart and nameRune classify the characters of a variable name;
	// nameStart rejects digit-leading names (`$1x` is the positional
	// parameter `$1` plus a literal `x` in POSIX shells).
	nameStart func(r rune) bool
	nameRune  func(r rune) bool
}

// posixVariableRef detects a variable reference ending the given word using
// POSIX expansion rules: `$name` and `${name`, backslash escapes, and no
// expansion inside single quotes.
func posixVariableRef(word Token) (Variable, bool) {
	return variableRef(word, variableRules{
		brace:           true,
		expands:         func(state LexerState) bool { return state != QUOTING_STATE },
		barewordEscapes: true,
		nameStart:       isPosixNameStart,
		nameRune:        isPosixNameRune,
	})
}

// variableRef detects a variable reference ending the given word.
//
// The word is parsed left to right so a `$` consumed as the special
// character of a preceding parameter is not mistaken for an opener
// (`$$` is the PID parameter, while `$$$HO` still opens `$HO`).
func variableRef(word Token, rules variableRules) (Variable, bool) {
	if !rules.expands(word.State) {
		return Variable{}, false
	}

	runes := []rune(word.RawValue)
	opener := -1
	for i := 0; i < len(runes); i++ {
		if runes[i] != '$' {
			continue
		}
		escaped := false
		for k := i; k > 0 && runes[k-1] == '\\'; k-- {
			escaped = !escaped
		}
		if escaped && (rules.barewordEscapes || word.State == QUOTING_ESCAPING_STATE) {
			continue
		}
		if i == len(runes)-1 { // trailing sigil
			opener = i
			break
		}
		switch next := runes[i+1]; {
		case next == '{' && rules.brace:
			if end := closedBraceEnd(runes, i+1); end >= 0 {
				i = end
				continue
			}
			opener = i // unclosed brace: the rest is the name region
			i = len(runes)
		case rules.nameRune(next):
			j := i + 1
			for j < len(runes) && rules.nameRune(runes[j]) {
				j++
			}
			if j == len(runes) { // name runs to the end: trailing opener
				opener = i
			}
			i = j - 1
		default:
			i++ // special parameter: consume `$` and its character
		}
	}
	if opener < 0 {
		return Variable{}, false
	}

	rest := runes[opener+1:]
	brace := len(rest) > 0 && rest[0] == '{'
	if brace {
		rest = rest[1:]
	}
	if len(rest) > 0 && !rules.nameStart(rest[0]) {
		return Variable{}, false
	}
	for _, r := range rest {
		if !rules.nameRune(r) {
			return Variable{}, false
		}
	}

	return Variable{
		Name:  string(rest),
		brace: brace,
	}, true
}

// closedBraceEnd returns the index of the `}` closing the brace expansion
// whose `{` is at index start, or -1 when unclosed.
func closedBraceEnd(runes []rune, start int) int {
	for i := start + 1; i < len(runes); i++ {
		if runes[i] == '}' {
			return i
		}
	}
	return -1
}

func isPosixNameStart(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		return true
	default:
		return false
	}
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
