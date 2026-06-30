package shlex

// powershellFormat implements Format for PowerShell lexing.
// Key differences from bash:
//   - Backtick (`) is the escape character, not backslash (\)
//   - ” inside single quotes → literal ' (doubled quote)
//   - "" inside double quotes → literal " (doubled quote)
//   - No single-quote-as-quote for outer quote pairs in the POSIX sense;
//     both ' and " are quote chars
//   - Here-strings (@'...'@, @"..."@) and --% are deferred to Phase 4
type powershellFormat struct{}

// PowershellFormat returns the PowerShell lexical format.
func PowershellFormat() Format { return powershellFormat{} }

func (powershellFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)       // " is escaping quote
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass) // ' is non-escaping
	// PowerShell: backtick is the escape character, not backslash
	t.addRuneClass("`", escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// PowerShell operators: |, ;, >, >>, &&, ||, &
	// Note: & is the call operator, not a background operator
	wordbreakRunes := "|;&><"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
	return t
}

func (powershellFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case ">", ">>":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case "&":
		return WORDBREAK_UNKNOWN // call operator, not a list operator
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (powershellFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (powershellFormat) NonEscapingQuoteEscapes() bool          { return true } // '' → '
func (powershellFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (powershellFormat) EscapeNotBareword() bool                { return true }
