package shlex

// zshFormat implements Format for zsh lexing.
// Extends bash with RC_QUOTES, zsh-specific operators (>>|, ;&, ;|, &|),
// and WORDCHARS/FIGNORE for word breaks.
type zshFormat struct{}

func (zshFormat) Classifier() tokenClassifier {
	return bashFormat{}.Classifier() // zsh uses the same rune classes as bash
}

func (zshFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case ">>|":
		return WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE
	case ";&":
		return WORDBREAK_LIST_FALLTHROUGH
	case ";|":
		return WORDBREAK_LIST_FALLTHROUGH_RETRY
	case "&|":
		return WORDBREAK_LIST_ASYNC_ERRCHECK
	default:
		return bashWordbreakType(raw)
	}
}

func (zshFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (zshFormat) NonEscapingQuoteEscapes() bool          { return true } // RC_QUOTES: '' → '
func (zshFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (zshFormat) EscapeNotBareword() bool                { return true }
func (zshFormat) EscapeNotInEscapingQuote() bool         { return false }
func (zshFormat) EscapingQuoteEscapeChars() map[rune]bool {
	return map[rune]bool{
		'\\': true,
		'`':  true,
		'$':  true,
		'"':  true,
		'\n': true,
	}
}
func (zshFormat) QuoteWord(s string) string { return posixQuoteWord(s) }
func (zshFormat) TripleQuoteSupport() bool  { return false }
func (zshFormat) RawPrefixSupport() bool    { return false }

// IsLineContinuation implements lineContinuationEscaper. Zsh (like bash)
// treats backslash followed by \n or \r as a line continuation.
func (zshFormat) IsLineContinuation(r rune) bool {
	return r == '\n' || r == '\r'
}

// PostProcess reclassifies ( and ) as substitution delimiters and merges
// $ + ( into a single opener token for POSIX command substitution.
func (zshFormat) PostProcess(tokens TokenSlice) TokenSlice {
	return posixSubstitutionPostProcess(tokens)
}

// VariableRef implements variableExpander. Zsh expands `$name` and `${name`;
// `$` is literal inside single quotes and when escaped. Quote stripping at
// insertion time is compsys's job (IPREFIX), not detection's.
func (zshFormat) VariableRef(word Token) (VariableRef, bool) {
	return posixVariableRef(word)
}
