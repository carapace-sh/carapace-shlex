package shlex

// luaFormat implements Format for Lua lexing (cmd-clink snippets are Lua).
//
// Known limitations (POSIX-oriented state machine):
//   - Lua keywords and stdlib names lex as plain words
//   - line comments (--) and block comments (--[[ ... ]]) are not lexed;
//     long strings ([[ ... ]], [=[ ... ]=]) are not supported
//   - hex (\xHH), decimal (\ddd), and \z escapes inside strings are not
//     tokenized specially; backslash still escapes the following rune
type luaFormat struct{}

// LUA_WORDBREAKS are the wordbreak characters for Lua: whitespace plus the
// structural punctuation. Operators are intentionally omitted; the flat
// state machine needs no operator semantics for string quoting.
const LUA_WORDBREAKS = ";(){},.=:[ ]"

func (luaFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	t.addWordbreaks(LUA_WORDBREAKS)
	return t
}

func (luaFormat) ClassifyOperator(raw string) WordbreakType {
	// Lua has no redirections or pipelines.
	switch raw {
	case "(":
		return WORDBREAK_OUTPUT_CAPTURE
	case ")":
		return WORDBREAK_OUTPUT_CAPTURE
	default:
		return WORDBREAK_LIST_SEQUENTIAL
	}
}

func (luaFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (luaFormat) NonEscapingQuoteEscapes() bool          { return false }
func (luaFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (luaFormat) EscapeNotBareword() bool                { return true }
func (luaFormat) EscapeNotInEscapingQuote() bool         { return false }

// EscapingQuoteEscapeChars returns nil: a Lua backslash inside a string
// escapes any following character.
func (luaFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }

func (luaFormat) QuoteWord(s string) string { return luaQuoteWord(s) }
func (luaFormat) TripleQuoteSupport() bool  { return false }
func (luaFormat) RawPrefixSupport() bool    { return false }
