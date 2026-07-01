package shlex

import "strings"

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

// elvishQuoteWord quotes a word for elvish.
// Elvish single quotes: ” → literal '. No escapes inside single quotes.
func elvishQuoteWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, "'\" \t\r\n") {
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
