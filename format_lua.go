package shlex

// luaFormat implements Format for Lua lexing (cmd-clink snippets are Lua).
// Inside single quotes, a doubled quote and \' are treated as an escaped
// quote (fish/zsh RC_QUOTES style) rather than Lua's adjacent-literal
// concatenation, so words like 'don\'t' survive lexing.
type luaFormat struct{}

// Lua has no redirections or pipelines; the structural punctuation is
// only needed for word separation. Comments (--) or long strings
// ([[ ... ]]) are not lexed.
const LUA_WORDBREAKS = ";(){},.=:"

func (luaFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	t.addWordbreaks(LUA_WORDBREAKS)
	return t
}

func (luaFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case "(", ")":
		return WORDBREAK_BRACKET
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (luaFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (luaFormat) NonEscapingQuoteEscapes() bool          { return true }
func (luaFormat) NonEscapingQuoteBackslashEscapes() bool { return true }
func (luaFormat) EscapeNotBareword() bool                { return true }
func (luaFormat) EscapeNotInEscapingQuote() bool         { return false }

// EscapingQuoteEscapeChars returns nil: inside a Lua string a backslash
// escapes the next character. The simplified model keeps only \", \',
// and \\ (plus \n via luaQuoteWord); hex, decimal, and \z escapes pass
// through as plain characters.
func (luaFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }

func (luaFormat) QuoteWord(s string) string { return luaQuoteWord(s) }
func (luaFormat) TripleQuoteSupport() bool  { return false }
func (luaFormat) RawPrefixSupport() bool    { return false }
