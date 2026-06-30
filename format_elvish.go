package shlex

// elvishFormat implements Format for elvish lexing.
// Key differences from bash:
// - ” inside single quotes → literal ' (same as zsh RC_QUOTES)
// - \ is NOT an escape character outside quotes (it's a bareword char)
// - No POSIX list operators (no &&, ||, &)
type elvishFormat struct{}

// ElvishFormat returns the elvish lexical format.
func ElvishFormat() Format { return elvishFormat{} }

func (elvishFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	// Elvish operators: |, >, <, >>, >>?, <>>, ;
	// No &, &&, || — & is for map literals
	t.addWordbreaks("|><;")
	return t
}

func (elvishFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ">", ">>", ">>?", "<>", "<":
		return WORDBREAK_REDIRECT_OUTPUT // simplified; elvish redirects
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (elvishFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (elvishFormat) NonEscapingQuoteEscapes() bool          { return true } // '' → '
func (elvishFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (elvishFormat) EscapeNotBareword() bool                { return false }
func (elvishFormat) QuoteWord(s string) string              { return elvishQuoteWord(s) }
