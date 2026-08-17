package shlex

// elvishFormat implements Format for elvish lexing.
// Key differences from bash:
// - ” inside single quotes → literal ' (same as zsh RC_QUOTES)
// - \ is NOT an escape character outside quotes (it's a bareword char)
// - No POSIX list operators (no &&, ||, &)
type elvishFormat struct{}

// ElvishFormat returns the elvish lexical format.
func ElvishFormat() Format { return elvishFormat{} }

func (elvishFormat) Classifier() tokenClassifier {
	t := newBaseClassifier(escapeRunes)
	// Elvish operators: |, >, <, >>, >>?, <>>, ;
	// No &, &&, || — & is for map literals
	// ( and ) are output-capture delimiters (always word breaks)
	// [ and ] are list-literal/indexing delimiters (word break at word start,
	// but not when following a word char for indexing — handled in PostProcess)
	t.addWordbreaks("|><;()[]")
	return t
}

func (elvishFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ">", ">>", ">>?", "<>", "<":
		return WORDBREAK_REDIRECT_OUTPUT // simplified; elvish redirects
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case "(", ")":
		return WORDBREAK_OUTPUT_CAPTURE
	case "[", "]":
		return WORDBREAK_BRACKET
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (elvishFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (elvishFormat) NonEscapingQuoteEscapes() bool           { return true } // '' → '
func (elvishFormat) NonEscapingQuoteBackslashEscapes() bool  { return false }
func (elvishFormat) EscapeNotBareword() bool                 { return false }
func (elvishFormat) EscapeNotInEscapingQuote() bool          { return false }
func (elvishFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (elvishFormat) QuoteWord(s string) string               { return elvishQuoteWord(s) }
func (elvishFormat) TripleQuoteSupport() bool                { return false }
func (elvishFormat) RawPrefixSupport() bool                  { return false }

// LineContinuationChar implements LineContinuationWhitespace. Elvish uses ^
// followed by \n or \r\n as whitespace (a word break), not as concatenation.
func (elvishFormat) LineContinuationChar() rune { return '^' }
func (elvishFormat) IsLineContinuationWhitespace(r rune) bool {
	return r == '\n' || r == '\r'
}

// braceState tracks the parser context inside braces.
type braceState int

const (
	braceOutside    braceState = iota // not inside braces
	braceLambdaOpen                   // saw '{' followed by |/space/newline → lambda, no params yet
	braceParams                       // inside {|...| parameter list (between first | and second |)
	braceLambdaBody                   // after closing | of params, in lambda body
	braceBraced                       // inside braced list {a,b}
)

// PostProcess reclassifies WORDBREAK_PIPE tokens that are inside elvish
// lambda parameter lists as WORDBREAK_LAMBDA_PIPE, and reclassifies
// output-capture delimiters ( and ) as substitution delimiters.
func (elvishFormat) PostProcess(tokens TokenSlice) TokenSlice {
	var stack []braceState

	for i := range tokens {
		t := &tokens[i]

		// Reclassify output-capture delimiters as substitution delimiters
		if t.Type == WORDBREAK_TOKEN && t.WordbreakType == WORDBREAK_OUTPUT_CAPTURE {
			if t.Value == "(" {
				t.WordbreakType = WORDBREAK_SUBSTITUTION_OPEN
			} else if t.Value == ")" {
				t.WordbreakType = WORDBREAK_SUBSTITUTION_CLOSE
			}
		}

		if t.Type == WORD_TOKEN && t.Value == "{" {
			isLambda := false
			isLambdaBody := false
			if i+1 < len(tokens) {
				next := tokens[i+1]
				if next.Type == WORDBREAK_TOKEN && next.Value == "|" {
					isLambda = true
				} else if !t.adjoins(next) {
					// { followed by space then a non-| token: lambda with
					// no params, body starts immediately. Any | inside is
					// a real pipeline pipe, not a param delimiter.
					isLambdaBody = true
				}
			} else {
				// { at EOF: incomplete lambda, treat as lambda open
				isLambda = true
			}
			if isLambda {
				stack = append(stack, braceLambdaOpen)
			} else if isLambdaBody {
				stack = append(stack, braceLambdaBody)
			} else {
				stack = append(stack, braceBraced)
			}
			continue
		}

		if t.Type == WORD_TOKEN {
			// Count { and } in non-standalone words to maintain brace stack.
			// This handles cases like $a} where } is embedded in a word
			// and doesn't match the standalone t.Value == "}" check above.
			if t.Value != "{" && t.Value != "}" {
				for _, r := range t.Value {
					switch r {
					case '{':
						stack = append(stack, braceBraced)
					case '}':
						if len(stack) > 0 {
							stack = stack[:len(stack)-1]
						}
					}
				}
			} else if t.Value == "}" {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				continue
			}
		}

		if t.Type == WORDBREAK_TOKEN && t.Value == "|" && len(stack) > 0 {
			scope := &stack[len(stack)-1]
			switch *scope {
			case braceLambdaOpen:
				*scope = braceParams
				t.WordbreakType = WORDBREAK_LAMBDA_PIPE
			case braceParams:
				*scope = braceLambdaBody
				t.WordbreakType = WORDBREAK_LAMBDA_PIPE
			case braceBraced:
				t.WordbreakType = WORDBREAK_LAMBDA_PIPE
			}
		}
	}

	return tokens
}
