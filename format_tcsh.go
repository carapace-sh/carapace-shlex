package shlex

// tcshFormat implements Format for tcsh lexing.
// Tcsh is POSIX-family with the same operator grammar as bash.
// backslash_quote and $'...' are handled by the state machine
// (backslash is already the escape char, $ is a word char before quotes).
type tcshFormat struct{}

// TcshFormat returns the tcsh lexical format.
// Uses the same rune classes and operator grammar as bash.
func TcshFormat() Format { return tcshFormat{} }

func (tcshFormat) Classifier() tokenClassifier {
	return bashFormat{}.Classifier()
}

func (tcshFormat) ClassifyOperator(raw string) WordbreakType {
	return bashWordbreakType(raw)
}

func (tcshFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (tcshFormat) NonEscapingQuoteEscapes() bool { return false }
func (tcshFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (tcshFormat) EscapeNotBareword() bool { return true }
