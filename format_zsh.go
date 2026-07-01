package shlex

// zshFormat implements Format for zsh lexing.
// Extends bash with RC_QUOTES, zsh-specific operators (>>|, ;&, ;|, &|),
// and WORDCHARS/FIGNORE for word breaks.
type zshFormat struct{}

// ZshFormat returns the zsh lexical format.
// RC_QUOTES is enabled (zsh's default for ” inside single quotes).
func ZshFormat() Format { return zshFormat{} }

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
func (zshFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (zshFormat) QuoteWord(s string) string              { return posixQuoteWord(s) }
