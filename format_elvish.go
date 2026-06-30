package shlex

// elvishFormat implements Format for elvish lexing.
// Key differences from bash:
// - '' inside single quotes → literal ' (same as zsh RC_QUOTES)
// - \ is NOT an escape character outside quotes (it's a bareword char)
// - No POSIX list operators (no &&, ||, &)
type elvishFormat struct{}

// ElvishFormat returns the elvish lexical format.
func ElvishFormat() Format { return elvishFormat{} }

func (elvishFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass)
	// Elvish: \ is a bareword character, not an escape outside quotes.
	// It IS an escape inside double quotes, so we still classify it as
	// escapeRuneClass — the state machine only uses it in ESCAPING_STATE
	// which is entered from IN_WORD_STATE. For elvish, we need to NOT
	// enter ESCAPING_STATE from IN_WORD_STATE.
	// TODO: this requires a format flag to disable bareword escaping.
	// For now, classify \ as escape to make double-quote escapes work,
	// and handle the bareword case in the state machine.
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// Elvish operators: |, >, <, >>, >>?, <>>, ;
	// No &, &&, || — & is for map literals
	wordbreakRunes := "|><;"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
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

func (elvishFormat) NonEscapingQuoteEscapes() bool { return true }  // '' → '
func (elvishFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
