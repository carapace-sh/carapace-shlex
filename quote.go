package shlex

import (
	"strings"
	"unicode"
)

// posixQuoteWord quotes a word for POSIX shells (bash, zsh, oil, tcsh).
// Uses double-quote wrapping with escape sequences for $, `, ", \, and
// newline (the only control char that backslash escapes inside double quotes
// in POSIX shells). Tab and CR are emitted literally inside double quotes
// since they are safe there and not backslash-escaped by the shell.
// Safe words (no special chars) are returned as-is.
func posixQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, `"' `+"`$\n\r\t\\") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '$':
			b.WriteString(`\$`)
		case '`':
			b.WriteString("\\`")
		case '\n':
			b.WriteString(`\` + "\n")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// fishQuoteWord quotes a word for fish.
// Fish double quotes escape only ", $, \, and newline.
// Backtick is a regular character in fish (not command substitution).
func fishQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, `"' `+"$\n\r\t\\") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '$':
			b.WriteString(`\$`)
		case '\n':
			b.WriteString("\\\n")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// hilbishQuoteWord quotes a word for hilbish.
// Matches hilbish's own completion insertion style (complete.go
// escapeFilename via charEscapeMap): metacharacters are backslash-escaped
// in barewords rather than the word being quoted. Backslash and semicolon
// are escaped too even though hilbish's charEscapeMap omits them — without
// them a joined line would not round-trip (a raw \ is dropped by the sh
// parser, a raw ; splits the command).
func hilbishQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, "\"'` ()[]$&*><|\\;") {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"', '\'', '`', ' ', '(', ')', '[', ']', '$', '&', '*', '>', '<', '|', '\\', ';':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString("\\\n")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// elvishQuoteWord quotes a word for elvish.
// Elvish barewords allow: letters, digits, -_:~./\@%+! and non-ASCII printable.
// Single quotes: two consecutive single-quote chars produce one literal quote.
// Empty string uses single-quoted empty (matching elvish Quote).
func elvishQuoteWord(s string) string {
	if s == "" {
		return `''`
	}
	if isElvishBareword(s) {
		return s
	}
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		if r == '\'' {
			b.WriteString("''")
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// isElvishBareword reports whether s is a valid elvish bareword (strict context).
// Matches elvish's allowedInBareword with strictExpr context: = and , are
// not allowed, and <>*^ are not allowed (those are CmdExpr-only).
func isElvishBareword(s string) bool {
	if s[0] == '~' {
		return false
	}
	for _, r := range s {
		if !isElvishBarewordRune(r) {
			return false
		}
	}
	return true
}

// isElvishBarewordRune reports whether r is allowed in an elvish bareword
// in strict expression context.
func isElvishBarewordRune(r rune) bool {
	return isElvishVariableNameRune(r) ||
		r == '.' || r == '/' || r == '\\' || r == '@' ||
		r == '%' || r == '+' || r == '!'
}

// isElvishVariableNameRune reports whether r is allowed in an elvish
// variable name (also the base for bareword characters).
func isElvishVariableNameRune(r rune) bool {
	return r >= 0x80 && unicode.IsPrint(r) ||
		'0' <= r && r <= '9' ||
		'a' <= r && r <= 'z' ||
		'A' <= r && r <= 'Z' ||
		r == '-' || r == '_' || r == ':' || r == '~'
}

// nushellQuoteWord quotes a word for nushell.
// Uses double-quote wrapping with C-style escapes for \ and ".
func nushellQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " {}()[]<>$&\"'|;#`\n\r\t\\") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// powershellQuoteWord quotes a word for PowerShell.
// Uses single-quote wrapping (verbatim, ” for literal ').
func powershellQuoteWord(s string) string {
	if s == "" {
		return `''`
	}
	if !strings.ContainsAny(s, " '\"`$&|;<>(){}\n\r\t") {
		return s
	}
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		if r == '\'' {
			b.WriteString("''")
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// xonshQuoteWord quotes a word for xonsh.
// Uses Python single-quote wrapping with \ escapes.
func xonshQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, "'\" \t\r\n\\") {
		return s
	}
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// cmdQuoteWord quotes a word for cmd.exe.
// Uses double-quote wrapping for spaces. Since cmd.exe has no escape
// mechanism inside double quotes (^ is literal inside quotes), a literal
// " is handled by closing the quote, escaping the " with ^, and reopening:
// "hello"^"world" — this produces the literal text hello"world.
func cmdQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \"&|<>()^,\t\n\r") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		if r == '"' {
			b.WriteString(`"^"`)
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// quoteInserter is an optional interface for formats whose insertion
// quoting differs from the POSIX-style default (PowerShell escapes with
// the backtick, cmd doubles the quote and treats ^ literally inside
// double quotes).
type quoteInserter interface {
	QuoteInsertion(state LexerState, value string) string
}

// QuoteInsertion implements quoteInserter for PowerShell. Inside double
// quotes the escape character is the backtick, not the backslash; other
// states use the default rules.
func (powershellFormat) QuoteInsertion(state LexerState, value string) string {
	if state == QUOTING_ESCAPING_STATE {
		var b strings.Builder
		b.WriteByte('"')
		for _, r := range value {
			switch r {
			case '"':
				b.WriteString("`\"")
			case '`':
				b.WriteString("``")
			case '$':
				b.WriteString("`$")
			default:
				b.WriteRune(r)
			}
		}
		b.WriteByte('"')
		return b.String()
	}
	return quoteInsertionDefault(powershellFormat{}, state, value)
}

// QuoteInsertion implements quoteInserter for cmd. The escape character
// is literal inside double quotes, so the quote itself is doubled; other
// states use the default rules.
func (cmdFormat) QuoteInsertion(state LexerState, value string) string {
	if state == QUOTING_ESCAPING_STATE {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return quoteInsertionDefault(cmdFormat{}, state, value)
}

func quoteInsertion(f formatImpl, state LexerState, value string) string {
	if qi, ok := f.(quoteInserter); ok {
		return qi.QuoteInsertion(state, value)
	}
	return quoteInsertionDefault(f, state, value)
}

// quoteInsertionDefault quotes value for insertion at a cursor in the
// given lexer state using the format's quoting primitives: quoting
// states are closed with the format's own escape rules, and barewords
// are quoted as a complete word (the QuoteWord rules).
func quoteInsertionDefault(f formatImpl, state LexerState, value string) string {
	if _, ok := f.(stopParsingToken); ok && state == STOP_PARSING_STATE {
		return value // raw mode: nothing is quoted
	}

	switch state {
	case QUOTING_STATE: // inside single quotes: close them
		switch {
		case f.NonEscapingQuoteBackslashEscapes():
			return `'` + strings.ReplaceAll(value, `'`, `\'`) + `'`
		case f.NonEscapingQuoteEscapes():
			return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
		default:
			return `'` + strings.ReplaceAll(value, `'`, `'"'"'`) + `'`
		}

	case QUOTING_ESCAPING_STATE: // inside double quotes: close them
		if f.EscapeNotInEscapingQuote() {
			// the escape character is literal inside double quotes
			// (cmd): the quote itself is doubled instead
			return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
		}
		return `"` + escapeDoubleQuote(f, value) + `"`

	case QUOTING_TRIPLE_STATE: // inside '''...''': close them
		return `'''` + strings.ReplaceAll(value, `'`, `\'`) + `'''`

	case QUOTING_TRIPLE_ESCAPING_STATE: // inside """...""": close them
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		return `"""` + strings.ReplaceAll(escaped, `"`, `\"`) + `"""`

	default: // bareword: quote as a complete word
		return f.QuoteWord(value)
	}
}

// escapeDoubleQuote escapes value for the inside of a double-quoted
// string: the quote itself, the escape character, and the format's
// documented escape set (e.g. `$` in bash and fish).
func escapeDoubleQuote(f formatImpl, value string) string {
	chars := f.EscapingQuoteEscapeChars()
	var b strings.Builder
	for _, r := range value {
		switch {
		case r == '"', r == '\\', chars != nil && chars[r]:
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
