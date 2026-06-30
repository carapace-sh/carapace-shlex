package shlex

import "os"

// bashFormat implements Format for POSIX/bash lexing.
// This is the default format and reproduces v1 behavior exactly.
type bashFormat struct{}

// BashFormat returns the POSIX/bash lexical format.
// It reads COMP_WORDBREAKS from the environment at Classifier() call time.
func BashFormat() Format { return bashFormat{} }

func (bashFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)

	wordbreakRunes := BASH_WORDBREAKS
	if wordbreaks := os.Getenv("COMP_WORDBREAKS"); wordbreaks != "" {
		wordbreakRunes = wordbreaks
	}
	t.addWordbreaks(wordbreakRunes)

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
