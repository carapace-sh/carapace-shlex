package shlex

import "os"

// bashFormat implements Format for POSIX/bash lexing.
// This is the default format and reproduces v1 behavior exactly.
type bashFormat struct{}

// BashFormat returns the POSIX/bash lexical format.
// It reads COMP_WORDBREAKS from the environment at Classifier() call time.
func BashFormat() Format { return bashFormat{} }

func (bashFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	wordbreakRunes := BASH_WORDBREAKS
	if wordbreaks := os.Getenv("COMP_WORDBREAKS"); wordbreaks != "" {
		wordbreakRunes = wordbreaks
	}
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)

	return t
}

func (bashFormat) ClassifyOperator(raw string) WordbreakType {
	return bashWordbreakType(raw)
}

func (bashFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (bashFormat) NonEscapingQuoteEscapes() bool { return false }

func (bashFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (bashFormat) EscapeNotBareword() bool                { return true }
func (bashFormat) QuoteWord(s string) string              { return posixQuoteWord(s) }
