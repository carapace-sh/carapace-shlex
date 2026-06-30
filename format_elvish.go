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
	t.addWordbreaks("|><;")
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
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (elvishFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (elvishFormat) NonEscapingQuoteEscapes() bool          { return true } // '' → '
func (elvishFormat) NonEscapingQuoteBackslashEscapes() bool { return false }
func (elvishFormat) EscapeNotBareword() bool                { return false }
func (elvishFormat) QuoteWord(s string) string              { return elvishQuoteWord(s) }

// braceState tracks the parser context inside braces.
type braceState int

const (
	braceOutside     braceState = iota // not inside braces
	braceLambdaOpen                    // saw '{' followed by |/space/newline → lambda, no params yet
	braceParams                        // inside {|...| parameter list (between first | and second |)
	braceLambdaBody                    // after closing | of params, in lambda body
	braceBraced                        // inside braced list {a,b}
)

// PostProcess reclassifies WORDBREAK_PIPE tokens that are inside elvish
// lambda parameter lists as WORDBREAK_LAMBDA_PIPE. This is needed because
// the flat tokenizer cannot distinguish | as pipe vs | as lambda parameter
// delimiter — that distinction is contextual in elvish's grammar.
func (elvishFormat) PostProcess(tokens TokenSlice) TokenSlice {
	var stack []braceState

	for i := range tokens {
		t := &tokens[i]

		if t.Type == WORD_TOKEN && t.Value == "{" {
			isLambda := false
			if i+1 < len(tokens) {
				next := tokens[i+1]
				if next.Type == WORDBREAK_TOKEN && next.Value == "|" {
					isLambda = true
				} else if !t.adjoins(next) {
					isLambda = true
				}
			}
			if isLambda {
				stack = append(stack, braceLambdaOpen)
			} else {
				stack = append(stack, braceBraced)
			}
			continue
		}

		if t.Type == WORD_TOKEN && t.Value == "}" {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
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
