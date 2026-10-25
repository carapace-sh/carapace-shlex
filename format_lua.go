package shlex

import "strings"

// luaFormat implements Format for Lua lexing (cmd-clink snippets are Lua).
// Inside single quotes, a doubled quote and \' are treated as an escaped
// quote (fish/zsh RC_QUOTES style); real Lua instead reads doubled-quote
// sequences as two adjacent string literals, but the doubling approximation
// lets words like 'don\'t' survive lexing.
type luaFormat struct{}

// Lua has no redirections or pipelines; the structural punctuation is
// only needed for word separation. Comments (--) or long strings
// ([[ ... ]]) are not lexed.
const LUA_WORDBREAKS = ";(){},.=:"

func (luaFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	t.addRuneClass("\f\v", spaceRuneClass)
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

// QuoteInsertion implements quoteInserter. Raw newlines are invalid inside
// Lua short strings regardless of the quote style, and
// EscapingQuoteEscapeChars returns nil, so the default insertion would
// emit an unescaped newline; escape quote, backslash, and newline instead.
func (luaFormat) QuoteInsertion(state LexerState, value string) string {
	switch state {
	case QUOTING_STATE:
		return `'` + luaEscapeInterior(value, '\'') + `'`
	case QUOTING_ESCAPING_STATE:
		return `"` + luaEscapeInterior(value, '"') + `"`
	default:
		return quoteInsertionDefault(luaFormat{}, state, value)
	}
}

// luaEscapeInterior escapes value for the inside of a Lua string literal:
// the string's own quote character, the backslash, and newlines as \n.
func luaEscapeInterior(value string, quote rune) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case quote:
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (luaFormat) IsBareWord(s string) bool {
	// isLuaBareword accepts the empty identifier
	return s != "" && isLuaBareword(s)
}
