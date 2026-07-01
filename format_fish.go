package shlex

// fishFormat implements Format for fish lexing.
// Key differences from bash:
// - \' and \\ are escapes inside single quotes (NonEscapingQuoteBackslashEscapes)
// - Keyword operators: and, or (bare words acting as pipeline delimiters)
// - `not` is a prefix keyword but not a pipeline delimiter, so not in KeywordOperators
// - No word splitting on variable expansion (doesn't affect lexing)
// - Narrower escape set in double quotes (\" \$ \\ and \+newline only)
// - Supports &&, ||, & (background), |&, &|, &>, &>>, &>?, &>>?, >&, >|, <>, >?, >>?, <?, <>&
type fishFormat struct{}

// FishFormat returns the fish lexical format.
func FishFormat() Format { return fishFormat{} }

func (fishFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	// Fish operators: |, ;, <, >, &, ?
	// & is included for &&, ||, &, |&, &|, &>, &>>, &>?, >&, <>&
	// ? is part of redirect operators (>? >>? <?) — deprecated glob char
	t.addWordbreaks("|;<>&?")
	return t
}

func (fishFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case "|&", "&|":
		return WORDBREAK_PIPE_WITH_STDERR
	case ">|":
		return WORDBREAK_PIPE // pipe with explicit fd (e.g. 2>|)
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case "&":
		return WORDBREAK_LIST_ASYNC
	case ">", ">>", ">>?", ">?":
		return WORDBREAK_REDIRECT_OUTPUT
	case "&>", "&>?":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH
	case "&>>", "&>>?":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND
	case "<", "<>&", "<?":
		return WORDBREAK_REDIRECT_INPUT
	case "<>":
		return WORDBREAK_REDIRECT_INPUT_OUTPUT
	case ">&":
		return WORDBREAK_REDIRECT_INPUT_DUPLICATE // fd redirection (e.g. >&2)
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

// EscapingQuoteEscapeChars returns the limited set of characters that
// backslash can escape inside fish double quotes: ", $, \, and newline.
// All other \X sequences are literal (both characters emitted).
func (fishFormat) EscapingQuoteEscapeChars() map[rune]bool {
	return fishDoubleQuoteEscapes
}

var fishDoubleQuoteEscapes = map[rune]bool{
	'"':  true,
	'$':  true,
	'\\': true,
	'\n': true,
}

func (fishFormat) QuoteWord(s string) string { return fishQuoteWord(s) }
func (fishFormat) TripleQuoteSupport() bool  { return false }
func (fishFormat) RawPrefixSupport() bool    { return false }
