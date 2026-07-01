package shlex

// tcshFormat implements Format for tcsh lexing.
// Tcsh is POSIX-family (csh heritage) with operator grammar similar to bash
// but with key differences:
//   - >& for combined stdout+stderr redirect (bash uses &>)
//   - no <<< here-string, no ;; case operators, no &> redirect
//   - = and @ are not wordbreak characters (unlike bash)
//   - >! and >>! are parser-level constructs, not lexer operators:
//     '!' is a regular word char (_PUN) in tcsh's lexer; the parser in
//     syn3() recognizes > followed by ! as noclobber override.
//
// backslash_quote and $'...' are handled by the state machine
// (backslash is already the escape char, $ is a word char before quotes).
type tcshFormat struct{}

// TcshFormat returns the tcsh lexical format.
func TcshFormat() Format { return tcshFormat{} }

// TCSH_WORDBREAKS are the wordbreak characters for tcsh, derived from the
// _META character class in tcsh's _cmap table (sh.char.c).
// Unlike bash, tcsh does not include = or @ as wordbreak characters.
// ! is NOT a wordbreak: it is _PUN (punctuation) in tcsh's lexer, not _META.
// The >! and >>! noclobber operators are recognized at the parser level.
const TCSH_WORDBREAKS = "><;|&()"

func (tcshFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	t.addWordbreaks(TCSH_WORDBREAKS)
	return t
}

func (tcshFormat) ClassifyOperator(raw string) WordbreakType {
	return tcshWordbreakType(raw)
}

func (tcshFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (tcshFormat) NonEscapingQuoteEscapes() bool          { return false }
func (tcshFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (tcshFormat) EscapeNotBareword() bool                { return true }
func (tcshFormat) EscapingQuoteEscapeChars() map[rune]bool {
	return map[rune]bool{
		'\\': true,
		'`':  true,
		'$':  true,
		'"':  true,
		'\n': true,
	}
}
func (tcshFormat) QuoteWord(s string) string { return posixQuoteWord(s) }
func (tcshFormat) TripleQuoteSupport() bool  { return false }
func (tcshFormat) RawPrefixSupport() bool    { return false }
