package shlex

import "strings"

// nushellFormat implements Format for nushell lexing.
// Key differences from bash:
//   - Backtick (`) is a quote character (not an escape like PowerShell)
//   - $'...' and $"..." are interpolated strings ($ prefix + standard quote)
//   - C-style escapes in double quotes with a richer set than bash:
//     \" \' \\ \/ \b \f \r \n \t \0 \a \e \( \) \{ \} \$ \^ \# \| \~
//     \xHH and \u{X...} are deferred (see format-nushell.md → Deferred Features)
//   - No POSIX list operators (no &&, ||, &)
//   - Stream redirect operators: out>, err>, out+err>, o>, e>, o+e>
//     and pipe variants: e>|, err>|, o+e>|, out+err>|
//   - r#'...'# raw strings need multi-rune opener support (deferred — see
//     format-nushell.md → Deferred Features)
type nushellFormat struct{}

// NushellFormat returns the nushell lexical format.
// Basic quote types (single, double, backtick) are supported.
// Raw strings (r#'...'#) and here-strings are deferred.
func NushellFormat() Format { return nushellFormat{} }

func (nushellFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)       // "
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass) // '
	// Nushell: backtick is a quote character (not an escape)
	t.addRuneClass("`", nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeRunes, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// Nushell operators: |, ;, >, <, >>
	// No &&, ||, & — no POSIX list operators
	t.addWordbreaks("|;<>")
	return t
}

func (nushellFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case ">", ">>":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (nushellFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (nushellFormat) NonEscapingQuoteEscapes() bool           { return false }
func (nushellFormat) NonEscapingQuoteBackslashEscapes() bool  { return false }
func (nushellFormat) EscapeNotBareword() bool                 { return true }
func (nushellFormat) EscapeNotInEscapingQuote() bool          { return false }
func (nushellFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (nushellFormat) QuoteWord(s string) string               { return nushellQuoteWord(s) }
func (nushellFormat) TripleQuoteSupport() bool                { return false }
func (nushellFormat) RawPrefixSupport() bool                  { return false }

// EscapingQuoteUnescape implements the EscapingQuoteUnescaper interface.
// Nushell double-quoted strings support C-style escapes with a richer set
// than bash. Recognized escapes produce the corresponding character(s);
// unrecognized escapes keep both the backslash and the rune literally
// (nushell itself errors on unrecognized escapes, but for completion being
// lenient is better than failing).
func (nushellFormat) EscapingQuoteUnescape(r rune) (string, bool) {
	switch r {
	case '"':
		return "\"", true
	case '\'':
		return "'", true
	case '\\':
		return "\\", true
	case '/':
		return "/", true
	case 'b':
		return "\b", true
	case 'f':
		return "\f", true
	case 'r':
		return "\r", true
	case 'n':
		return "\n", true
	case 't':
		return "\t", true
	case '0':
		return "\x00", true
	case 'a':
		return "\a", true
	case 'e':
		return "\x1b", true
	case '(':
		return "(", true
	case ')':
		return ")", true
	case '{':
		return "{", true
	case '}':
		return "}", true
	case '$':
		return "$", true
	case '^':
		return "^", true
	case '#':
		return "#", true
	case '|':
		return "|", true
	case '~':
		return "~", true
	default:
		return "", false
	}
}

// nushellStreamRedirects maps the word portion of stream-redirect operators
// (the part before >) to their WordbreakType. The PostProcess step merges
// these with a following > or >| wordbreak token.
var nushellStreamRedirects = map[string]WordbreakType{
	"out":     WORDBREAK_REDIRECT_OUTPUT,
	"err":     WORDBREAK_REDIRECT_OUTPUT,
	"o":       WORDBREAK_REDIRECT_OUTPUT,
	"e":       WORDBREAK_REDIRECT_OUTPUT,
	"out+err": WORDBREAK_REDIRECT_OUTPUT_BOTH,
	"o+e":     WORDBREAK_REDIRECT_OUTPUT_BOTH,
}

// PostProcess merges nushell stream-redirect operators. The tokenizer
// produces e.g. `out` as a WORD_TOKEN and `>` (or `>|`, `>>`) as a
// WORDBREAK_TOKEN because the rune-classifier only handles single-rune
// word breaks. This step detects adjacent word+wordbreak sequences like
// `out>`, `err>`, `o+e>`, `e>|`, `o+e>|` and reclassifies them as single
// WORDBREAK_TOKENs with the appropriate WordbreakType.
func (nushellFormat) PostProcess(tokens TokenSlice) TokenSlice {
	result := make(TokenSlice, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		// Look for bare WORD_TOKEN immediately followed by WORDBREAK_TOKEN starting with '>'
		// Only merge bare words (Value == RawValue) — quoted words like 'out' or "out"
		// are string literals, not stream redirect operators.
		if t.Type == WORD_TOKEN && t.Value == t.RawValue && i+1 < len(tokens) {
			next := tokens[i+1]
			if next.Type == WORDBREAK_TOKEN && next.adjoins(t) && len(next.RawValue) > 0 && next.RawValue[0] == '>' {
				if wbType, ok := nushellStreamRedirects[t.Value]; ok {
					// Check if the wordbreak token includes a pipe suffix (e.g. >|)
					// which makes it a pipe-with-stderr variant
					if strings.Contains(next.RawValue, "|") {
						wbType = WORDBREAK_PIPE_WITH_STDERR
					}

					merged := Token{
						Type:          WORDBREAK_TOKEN,
						Value:         t.Value + next.Value,
						RawValue:      t.RawValue + next.RawValue,
						Span:          Span{Start: t.Span.Start, End: next.Span.End},
						State:         next.State,
						WordbreakType: wbType,
					}

					result = append(result, merged)
					i += 1
					continue
				}
			}
		}

		result = append(result, t)
	}
	return result
}
