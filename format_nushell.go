package shlex

// nushellFormat implements Format for nushell lexing.
// Key differences from bash:
// - Backtick (`) is a quote character (not an escape like PowerShell)
// - $'...' and $"..." are interpolated strings ($ prefix + standard quote)
// - r#'...'# raw strings need multi-rune opener support (deferred)
// - No POSIX list operators (no &&, ||, &)
// - C-style escapes in double quotes (same as bash)
type nushellFormat struct{}

// NushellFormat returns the nushell lexical format.
// Basic quote types (single, double, backtick) are supported.
// Raw strings (r#'...'#) and here-strings are deferred.
func NushellFormat() Format { return nushellFormat{} }

func (nushellFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)       // "
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass) // '
	// Nushell: backtick is a quote character (not an escape)
	t.addRuneClass("`", nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// Nushell operators: |, ;, >, <, >>
	// No &&, ||, & — no POSIX list operators
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

func (nushellFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case ">", ">>":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (nushellFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (nushellFormat) NonEscapingQuoteEscapes() bool          { return false }
func (nushellFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (nushellFormat) EscapeNotBareword() bool                { return true }
