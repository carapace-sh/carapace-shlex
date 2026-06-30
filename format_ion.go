package shlex

// ionFormat implements Format for ion lexing.
// Key differences from bash:
// - ^ is a wordbreak (operator prefix for ^>, ^|, ^>>)
// - &|, &> are combined-stream operators (ion-unique)
// - @ is NOT a wordbreak (it's an array sigil)
// - Otherwise same quote/escape rules as bash
type ionFormat struct{}

// IonFormat returns the ion lexical format.
func IonFormat() Format { return ionFormat{} }

func (ionFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// Ion operators: |, ^|, &|, >, >>, <, <<, <<<, ^>, ^>>, &>, &>>, ;, &&, ||, &
	// Note: ^ is an operator prefix (not escape), @ is NOT a wordbreak
	wordbreakRunes := "|^<>&;"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
	return t
}

func (ionFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case "^|": // ion-unique: stderr pipe
		return WORDBREAK_PIPE
	case "&|": // ion-unique: stdout+stderr pipe
		return WORDBREAK_PIPE
	case ">":
		return WORDBREAK_REDIRECT_OUTPUT
	case ">>":
		return WORDBREAK_REDIRECT_OUTPUT_APPEND
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	case "<<<":
		return WORDBREAK_REDIRECT_INPUT_STRING
	case "^>": // ion-unique: stderr to file
		return WORDBREAK_REDIRECT_OUTPUT
	case "^>>": // ion-unique: append stderr
		return WORDBREAK_REDIRECT_OUTPUT_APPEND
	case "&>":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH
	case "&>>":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case "&":
		return WORDBREAK_LIST_ASYNC
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (ionFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (ionFormat) NonEscapingQuoteEscapes() bool { return false }
func (ionFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
