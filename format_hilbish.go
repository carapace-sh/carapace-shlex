package shlex

// hilbishFormat implements Format for hilbish lexing.
// Hilbish's shell runner (snail) wraps mvdan.cc/sh with the parser's
// default language variant (LangBash), so the sh mode follows the full
// bash operator grammar (|&, <<<, &>, ;;&, ...). Verified against
// golibs/snail/snail.go and mvdan.cc/sh/v3/syntax parser defaults.
//
// Known limitations (flat state machine):
//   - the default hybrid runner tries Lua first (nature/runner.lua), so
//     input may be Lua code — Lua long strings ([[...]]) and string
//     escapes are not lexed; Lua strings share ' and " with sh anyway
//   - @-modifiers (@priv, @dir=...) at line start lex as plain words
type hilbishFormat struct{}

// HILBISH_WORDBREAKS are the wordbreak characters for hilbish: the sh
// metacharacters handled by its interpreter, without bash's COMP_WORDBREAKS
// customization (@ = : are not word breaks for hilbish).
const HILBISH_WORDBREAKS = "><;|&()"

func (hilbishFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	t.addWordbreaks(HILBISH_WORDBREAKS)
	return t
}

func (hilbishFormat) ClassifyOperator(raw string) WordbreakType {
	return bashWordbreakType(raw)
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
func (hilbishFormat) QuoteWord(s string) string { return hilbishQuoteWord(s) }
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
