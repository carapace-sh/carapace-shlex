package shlex

// xonshFormat implements Format for xonsh lexing.
// Xonsh is a Python/shell hybrid. For lexing purposes:
// - Standard single/double quotes work like bash
// - Prefix strings (r'...', f'...', p'...', b'...') work because the
//   prefix is a word char and Words() merges the segments
// - Triple-quotes ('''...''', """...""""") need 3-rune lookahead (deferred)
// - Shell operators: |, >, >>, <, ;, &&, ||, &
// - \ is the escape char (Python rules inside quotes)
type xonshFormat struct{}

// XonshFormat returns the xonsh lexical format.
// Standard quotes and prefix strings are supported.
// Triple-quotes are deferred.
func XonshFormat() Format { return xonshFormat{} }

func (xonshFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// Xonsh operators: |, >, >>, <, ;, &&, ||, &
	wordbreakRunes := "|<>&;"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
	return t
}

func (xonshFormat) ClassifyOperator(raw string) WordbreakType {
	return bashWordbreakType(raw) // xonsh uses POSIX operators
}

func (xonshFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (xonshFormat) NonEscapingQuoteEscapes() bool { return false }
func (xonshFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (xonshFormat) EscapeNotBareword() bool { return true }
