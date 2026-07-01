package shlex

// xonshFormat implements Format for xonsh lexing.
// Xonsh is a Python/shell hybrid. For lexing purposes:
//   - Standard single/double quotes work like bash
//   - Prefix strings (r'...', f'...', p'...', b'...') work because the
//     prefix is a word char and Words() merges the segments
//   - Triple-quotes (”'...”', """...""") detected via 2-rune lookahead
//   - Raw strings (r"...", r"""...""") suppress escape processing in double quotes
//   - Shell operators: |, >, >>, <, ;, &&, ||, &
//   - Keyword operators: and, or (bare words with surrounding whitespace)
//   - Stream redirect operators: e>, e>>, o>, o>>, a>, a>>, err>, out>, all>,
//     and pipe-channel variants (e>p, o>p, a>p) handled in PostProcess
//   - \ is the escape char (Python rules inside quotes, literal in raw strings)
type xonshFormat struct{}

// XonshFormat returns the xonsh lexical format.
// Standard quotes, prefix strings, triple-quotes, raw strings, keyword
// operators, and stream redirects are supported.
func XonshFormat() Format { return xonshFormat{} }

func (xonshFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	// Xonsh operators: |, >, >>, <, ;, &&, ||, &
	t.addWordbreaks("|<>&;")
	return t
}

func (xonshFormat) ClassifyOperator(raw string) WordbreakType {
	return bashWordbreakType(raw) // xonsh uses POSIX operators
}

func (xonshFormat) KeywordOperators() map[string]WordbreakType {
	return map[string]WordbreakType{
		"and": WORDBREAK_LIST_AND,
		"or":  WORDBREAK_LIST_OR,
	}
}

func (xonshFormat) NonEscapingQuoteEscapes() bool           { return false }
func (xonshFormat) NonEscapingQuoteBackslashEscapes() bool  { return true } // \' and \\ inside single quotes (Python syntax)
func (xonshFormat) EscapeNotBareword() bool                 { return true }
func (xonshFormat) EscapeNotInEscapingQuote() bool          { return false }
func (xonshFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (xonshFormat) QuoteWord(s string) string               { return xonshQuoteWord(s) }
func (xonshFormat) TripleQuoteSupport() bool                { return true }
func (xonshFormat) RawPrefixSupport() bool                  { return true }

// xonshStreamRedirects maps the word portion of xonsh stream-redirect
// operators (the part before > or <) to their WordbreakType. The PostProcess
// step merges these with a following > or < wordbreak token.
// See xonsh _REDIR_REGEX: (o(?:ut)?|e(?:rr)?|a(?:ll)?|&?\d?)(>?>|<)(...)
var xonshStreamRedirects = map[string]WordbreakType{
	"e":   WORDBREAK_REDIRECT_OUTPUT,
	"err": WORDBREAK_REDIRECT_OUTPUT,
	"o":   WORDBREAK_REDIRECT_OUTPUT,
	"out": WORDBREAK_REDIRECT_OUTPUT,
	"a":   WORDBREAK_REDIRECT_OUTPUT_BOTH,
	"all": WORDBREAK_REDIRECT_OUTPUT_BOTH,
}

// PostProcess merges xonsh stream-redirect operators. The tokenizer produces
// e.g. `e` as a WORD_TOKEN and `>` (or `>>`) as a WORDBREAK_TOKEN because the
// rune-classifier only handles single-rune word breaks. This step detects
// adjacent word+wordbreak sequences like `e>`, `o>>`, `a>`, `err>`, `out>`
// and reclassifies them as single WORDBREAK_TOKENs with the appropriate
// WordbreakType. Also handles the pipe-channel variants `e>p`, `o>p`, `a>p`.
func (xonshFormat) PostProcess(tokens TokenSlice) TokenSlice {
	result := make(TokenSlice, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		// Look for bare WORD_TOKEN immediately followed by WORDBREAK_TOKEN
		// starting with '>' or '<'. Only merge bare words (Value == RawValue)
		// — quoted words like 'e' or "out" are string literals, not operators.
		if t.Type == WORD_TOKEN && t.Value == t.RawValue && i+1 < len(tokens) {
			next := tokens[i+1]
			if next.Type == WORDBREAK_TOKEN && next.adjoins(t) && len(next.RawValue) > 0 &&
				(next.RawValue[0] == '>' || next.RawValue[0] == '<') {
				if wbType, ok := xonshStreamRedirects[t.Value]; ok {
					merged := Token{
						Type:          WORDBREAK_TOKEN,
						Value:         t.Value + next.Value,
						RawValue:      t.RawValue + next.RawValue,
						Span:          Span{Start: t.Span.Start, End: next.Span.End},
						State:         next.State,
						WordbreakType: wbType,
					}

					// Check for pipe-channel suffix: e>p, o>p, a>p
					// The 'p' would be a separate adjacent WORD_TOKEN
					if i+2 < len(tokens) {
						afterNext := tokens[i+2]
						if afterNext.Type == WORD_TOKEN && afterNext.Value == "p" &&
							afterNext.Value == afterNext.RawValue && afterNext.adjoins(next) {
							merged.Value += "p"
							merged.RawValue += "p"
							merged.Span.End = afterNext.Span.End
							result = append(result, merged)
							i += 2
							continue
						}
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
