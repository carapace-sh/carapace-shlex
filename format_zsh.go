package shlex

// zshFormat implements Format for zsh lexing.
// Extends bash with RC_QUOTES (” → ' inside single quotes).
type zshFormat struct{}

// ZshFormat returns the zsh lexical format.
// RC_QUOTES is enabled (zsh's default for ” inside single quotes).
func ZshFormat() Format { return zshFormat{} }

func (zshFormat) Classifier() tokenClassifier {
	return bashFormat{}.Classifier() // zsh uses the same rune classes as bash
}

func (zshFormat) ClassifyOperator(raw string) WordbreakType {
	return bashWordbreakType(raw) // zsh uses the same operator grammar as bash
}

func (zshFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (zshFormat) NonEscapingQuoteEscapes() bool          { return true } // RC_QUOTES: '' → '
func (zshFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (zshFormat) EscapeNotBareword() bool                { return true }
func (zshFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (zshFormat) QuoteWord(s string) string              { return posixQuoteWord(s) }
