package shlex

// fishFormat implements Format for fish lexing.
// Key differences from bash:
// - \' and \\ are escapes inside single quotes (NonEscapingQuoteBackslashEscapes)
// - Keyword operators: and, or (bare words acting as pipeline delimiters)
// - `not` is a prefix keyword but not a pipeline delimiter, so not in KeywordOperators
// - No word splitting on variable expansion (doesn't affect lexing)
// - Narrower escape set in double quotes (\" \$ \\ and \+newline only)
type fishFormat struct{}

// FishFormat returns the fish lexical format.
func FishFormat() Format { return fishFormat{} }

func (fishFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	// Fish operators: |, ;, <, >, >>, >>?, >?, <>&
	// No &&, ||, & — fish uses keyword operators (and, or, not) instead
	t.addWordbreaks("|;<>")
	return t
}

func (fishFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case ">", ">>", ">>?", ">?":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<", "<>&":
		return WORDBREAK_REDIRECT_INPUT
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (fishFormat) KeywordOperators() map[string]WordbreakType {
	return map[string]WordbreakType{
		"and": WORDBREAK_LIST_AND,
		"or":  WORDBREAK_LIST_OR,
	}
}

func (fishFormat) NonEscapingQuoteEscapes() bool          { return true } // ' and \\ inside single quotes
func (fishFormat) NonEscapingQuoteBackslashEscapes() bool { return true }
func (fishFormat) EscapeNotBareword() bool                { return true }
func (fishFormat) QuoteWord(s string) string              { return fishQuoteWord(s) }
