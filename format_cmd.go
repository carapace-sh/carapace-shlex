package shlex

// cmdFormat implements Format for cmd.exe (with clink) lexing.
// Key differences from POSIX:
// - Caret (^) is the escape character, not backslash (\)
// - Double quotes (") are the only quote — simple toggle, no \ escapes inside
// - No single quotes — ' is a literal word character
// - & is a command separator (like ; in POSIX), not background
// - ; is NOT a separator (literal character)
// - REM and :: are comments (keyword/two-rune, not rune-based)
// - % is a word character (variable expansion sigil)
// - \ is a literal word character (Windows paths)
type cmdFormat struct{}

// CmdFormat returns the cmd.exe lexical format.
// REM/:: keyword comments are not yet handled (deferred).
func CmdFormat() Format { return cmdFormat{} }

func (cmdFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	// Cmd: only " is a quote. ' is a regular word char.
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass) // " is the escaping quote
	// Cmd: ^ is the escape character, not \
	t.addRuneClass("^", escapeRuneClass)
	// Cmd: # is not a comment (that's REM/::). Don't classify it as comment.
	// REM/:: comments need keyword detection (deferred).

	// Cmd operators: |, &, <, >
	// Note: & is a command separator (like ; in POSIX), not background
	// ; is NOT a separator in cmd
	wordbreakRunes := "|&<>"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
	return t
}

func (cmdFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case "&":
		// Cmd: & is a command separator (like ; in POSIX)
		return WORDBREAK_LIST_SEQUENTIAL
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case ">", ">>":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (cmdFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (cmdFormat) NonEscapingQuoteEscapes() bool { return false }
func (cmdFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (cmdFormat) EscapeNotBareword() bool { return true } // ^ is always an escape
