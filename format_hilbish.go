package shlex

// hilbishFormat implements Format for hilbish lexing.
// Hilbish's shell runner (snail) wraps the mvdan/sh POSIX sh interpreter,
// while its hybrid runner may also execute input as Lua code. Lua long
// strings ([[...]]) are deferred multi-rune openers, like other formats'
// deferred string types.
//
// Key differences from bash:
//   - fixed wordbreaks (no COMP_WORDBREAKS)
//   - POSIX operator grammar only: no <<< here-string, no |&, no &> / &>>
type hilbishFormat struct{}

// HILBISH_WORDBREAKS are the wordbreak characters for hilbish, matching
// the POSIX sh metacharacters handled by its shell interpreter.
const HILBISH_WORDBREAKS = "><;|&()"

func (hilbishFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	t.addWordbreaks(HILBISH_WORDBREAKS)
	return t
}

func (hilbishFormat) ClassifyOperator(raw string) WordbreakType {
	return hilbishWordbreakType(raw)
}

func (hilbishFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (hilbishFormat) NonEscapingQuoteEscapes() bool          { return false }
func (hilbishFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (hilbishFormat) EscapeNotBareword() bool                { return true }
func (hilbishFormat) EscapeNotInEscapingQuote() bool         { return false }
func (hilbishFormat) EscapingQuoteEscapeChars() map[rune]bool {
	return map[rune]bool{
		'\\': true,
		'`':  true,
		'$':  true,
		'"':  true,
		'\n': true,
	}
}
func (hilbishFormat) QuoteWord(s string) string { return posixQuoteWord(s) }
func (hilbishFormat) TripleQuoteSupport() bool  { return false }
func (hilbishFormat) RawPrefixSupport() bool    { return false }

// IsLineContinuation implements lineContinuationEscaper. Hilbish's shell
// interpreter treats backslash followed by \n or \r as a line continuation.
func (hilbishFormat) IsLineContinuation(r rune) bool {
	return r == '\n' || r == '\r'
}

// PostProcess reclassifies ( and ) as substitution delimiters and merges
// $ + ( into a single opener token for POSIX command substitution.
func (hilbishFormat) PostProcess(tokens TokenSlice) TokenSlice {
	return posixSubstitutionPostProcess(tokens)
}

// Variable implements variableExpander. Hilbish's shell interpreter expands
// `$name` and `${name`; `$` is literal inside single quotes and when escaped.
func (hilbishFormat) Variable(word Token) (Variable, bool) {
	return posixVariableRef(word)
}
