package shlex

// powershellFormat implements Format for PowerShell lexing.
// Key differences from bash:
//   - Backtick (`) is the escape character, not backslash (\)
//   - ” inside single quotes → literal ' (doubled quote)
//   - "" inside double quotes → literal " (doubled quote)
//   - No single-quote-as-quote for outer quote pairs in the POSIX sense;
//     both ' and " are quote chars
//   - Backtick + newline is line continuation (consumed, not part of word)
//   - Block comments <# ... #> (multi-line)
//   - --% stop-parsing token (raw mode for remainder of line)
//   - Stream redirects: 2>, 2>>, 2>&1, 1>&2, *>, *>> (merged in PostProcess)
//   - Here-strings (@'...'@, @"..."@) are deferred
type powershellFormat struct{}

// PowershellFormat returns the PowerShell lexical format.
func PowershellFormat() Format { return powershellFormat{} }

func (powershellFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)       // " is escaping quote
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass) // ' is non-escaping
	// PowerShell: backtick is the escape character, not backslash
	t.addRuneClass("`", escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)

	// PowerShell operators: |, ;, >, >>, &&, ||, &
	// Note: & is the call operator, not a background operator
	wordbreakRunes := "|;&><"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
	return t
}

func (powershellFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case ">", ">>":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case "&":
		return WORDBREAK_UNKNOWN // call operator, not a list operator
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (powershellFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (powershellFormat) NonEscapingQuoteEscapes() bool           { return true } // '' → '
func (powershellFormat) NonEscapingQuoteBackslashEscapes() bool  { return false }
func (powershellFormat) EscapeNotBareword() bool                 { return true }
func (powershellFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (powershellFormat) QuoteWord(s string) string               { return powershellQuoteWord(s) }
func (powershellFormat) TripleQuoteSupport() bool                { return false }
func (powershellFormat) RawPrefixSupport() bool                  { return false }

// IsLineContinuation implements LineContinuationEscaper. PowerShell's
// backtick followed by \n or \r is a line continuation — the sequence is
// consumed and the word continues on the next line.
func (powershellFormat) IsLineContinuation(r rune) bool {
	return r == '\n' || r == '\r'
}

// BlockCommentOpener implements BlockCommenter. PowerShell supports
// multi-line block comments delimited by <# and #>.
func (powershellFormat) BlockCommentOpener() string { return "<#" }

// BlockCommentCloser implements BlockCommenter.
func (powershellFormat) BlockCommentCloser() string { return "#>" }

// StopParsingWord implements StopParsingToken. PowerShell's --% token
// stops PowerShell from interpreting subsequent input.
func (powershellFormat) StopParsingWord() string { return "--%" }

// PostProcess merges PowerShell stream-redirect operators. The tokenizer
// produces e.g. `2` as a WORD_TOKEN and `>` (or `>>`) as a WORDBREAK_TOKEN.
// This step detects adjacent word+wordbreak sequences like `2>`, `2>>`,
// `2>&1`, `1>&2`, `*>`, `*>>` and reclassifies them as single
// WORDBREAK_TOKENs with the appropriate WordbreakType.
func (powershellFormat) PostProcess(tokens TokenSlice) TokenSlice {
	result := make(TokenSlice, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		// Look for bare WORD_TOKEN (digit or *) immediately followed by
		// WORDBREAK_TOKEN starting with '>' (redirect operator)
		if t.Type == WORD_TOKEN && t.Value == t.RawValue && i+1 < len(tokens) {
			next := tokens[i+1]
			if next.Type == WORDBREAK_TOKEN && next.adjoins(t) &&
				next.WordbreakType.IsRedirect() && len(next.RawValue) > 0 && next.RawValue[0] == '>' {
				// Check if the word is a valid stream number or *
				if t.Value == "*" || (len(t.Value) == 1 && t.Value[0] >= '1' && t.Value[0] <= '6') {
					// Check for merging redirect: next token after > is &N
					// e.g. 2>&1 — the & and digit are separate wordbreak/word tokens
					wbType := next.WordbreakType
					mergedRaw := t.RawValue + next.RawValue
					mergedVal := t.Value + next.Value
					mergedSpan := Span{Start: t.Span.Start, End: next.Span.End}

					// Check for &N pattern (stream merge) in the token after next
					if i+2 < len(tokens) && tokens[i+2].Type == WORDBREAK_TOKEN &&
						tokens[i+2].Value == "&" && tokens[i+2].adjoins(next) {
						if i+3 < len(tokens) && tokens[i+3].Type == WORD_TOKEN &&
							tokens[i+3].Value == tokens[i+3].RawValue &&
							tokens[i+3].adjoins(tokens[i+2]) &&
							len(tokens[i+3].Value) == 1 &&
							(tokens[i+3].Value[0] == '1' || tokens[i+3].Value[0] == '2') {
							// 2>&1 pattern — merge all four tokens
							mergedRaw += tokens[i+2].RawValue + tokens[i+3].RawValue
							mergedVal += tokens[i+2].Value + tokens[i+3].Value
							mergedSpan.End = tokens[i+3].Span.End
							wbType = WORDBREAK_REDIRECT_OUTPUT_BOTH
							merged := Token{
								Type:          WORDBREAK_TOKEN,
								Value:         mergedVal,
								RawValue:      mergedRaw,
								Span:          mergedSpan,
								State:         tokens[i+3].State,
								WordbreakType: wbType,
							}
							result = append(result, merged)
							i += 3
							continue
						}
					}

					merged := Token{
						Type:          WORDBREAK_TOKEN,
						Value:         mergedVal,
						RawValue:      mergedRaw,
						Span:          mergedSpan,
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
