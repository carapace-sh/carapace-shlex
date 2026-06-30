package shlex

// fishFormat implements Format for fish lexing.
// Key differences from bash:
// - \' and \\ are escapes inside single quotes (NonEscapingQuoteEscapes)
// - Keyword operators: and, or, not (bare words acting as operators)
// - No word splitting on variable expansion (doesn't affect lexing)
// - Narrower escape set in double quotes (\" \$ \\ and \+newline only)
type fishFormat struct{}

// FishFormat returns the fish lexical format.
func FishFormat() Format { return fishFormat{} }

func (fishFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// Fish operators: |, ;, <, >, >>, >>?, >?, <>&
	// No &&, ||, & — fish uses keyword operators (and, or, not) instead
	// No @, =, : as wordbreaks (different from bash)
	wordbreakRunes := "|;<>"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
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

func (fishFormat) NonEscapingQuoteEscapes() bool { return true }  // ' and \\ inside single quotes
func (fishFormat) NonEscapingQuoteBackslashEscapes() bool { return true }
func (fishFormat) EscapeNotBareword() bool { return true }
