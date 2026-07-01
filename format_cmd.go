package shlex

// cmdFormat implements Format for cmd.exe (with clink) lexing.
// Key differences from POSIX:
//   - Caret (^) is the escape character outside quotes, but literal inside "..."
//   - Double quotes (") are the only quote — simple toggle, no escapes inside
//   - No single quotes — ' is a literal word character
//   - & is a command separator (like ; in POSIX), not background
//   - ; is NOT a separator (literal character)
//   - ( ) are grouping operators (command blocks)
//   - REM and :: are comments (keyword/two-rune, not rune-based)
//   - % is a word character (variable expansion sigil)
//   - \ is a literal word character (Windows paths)
//   - Caret + newline is line continuation (consumed, not part of word)
//   - Numeric stream redirects: 2>, 2>&1, 1>&2 (merged in PostProcess)
type cmdFormat struct{}

// CmdFormat returns the cmd.exe lexical format.
// REM/:: keyword comments are not yet handled (deferred).
func CmdFormat() Format { return cmdFormat{} }

func (cmdFormat) Classifier() tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	// Cmd: comma is a word delimiter (like space), but not a command separator.
	// Semicolons and equals are also delimiters in cmd, but ; is kept as a
	// literal word char because it's safer for completion (e.g. set VAR=value
	// would break if = were a delimiter). Comma is always safe to split.
	t.addRuneClass(",", spaceRuneClass)
	// Cmd: only " is a quote. ' is a regular word char.
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass) // " is the escaping quote
	// Cmd: ^ is the escape character, not \
	t.addRuneClass("^", escapeRuneClass)
	// Cmd: # is not a comment (that's REM/::). Don't classify it as comment.
	// REM/:: comments need keyword detection (deferred).

	// Cmd operators: |, &, <, >, (, )
	// Note: & is a command separator (like ; in POSIX), not background
	// ; is NOT a separator in cmd (it is a literal character)
	// ( and ) are grouping operators for command blocks
	wordbreakRunes := "|&<>()"
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
	return t
}

func (cmdFormat) ClassifyOperator(raw string) WordbreakType {
	switch raw {
	case "|":
		return WORDBREAK_PIPE
	case "&":
		// Cmd: & is a command separator (like ; in POSIX)
		return WORDBREAK_LIST_SEQUENTIAL
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case ">", ">>":
		return WORDBREAK_REDIRECT_OUTPUT
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	case "(", ")":
		// Cmd: parentheses are grouping operators for command blocks
		return WORDBREAK_UNKNOWN
	default:
		return WORDBREAK_UNKNOWN
	}
}

func (cmdFormat) KeywordOperators() map[string]WordbreakType { return nil }

func (cmdFormat) NonEscapingQuoteEscapes() bool           { return false }
func (cmdFormat) NonEscapingQuoteBackslashEscapes() bool  { return false }
func (cmdFormat) EscapeNotBareword() bool                 { return true }
func (cmdFormat) EscapeNotInEscapingQuote() bool          { return true }
func (cmdFormat) EscapingQuoteEscapeChars() map[rune]bool { return nil }
func (cmdFormat) QuoteWord(s string) string               { return cmdQuoteWord(s) }
func (cmdFormat) TripleQuoteSupport() bool                { return false }
func (cmdFormat) RawPrefixSupport() bool                  { return false }

// IsLineContinuation implements LineContinuationEscaper. cmd.exe's caret
// followed by \n or \r is a line continuation — the sequence is consumed
// and the word continues on the next line.
func (cmdFormat) IsLineContinuation(r rune) bool {
	return r == '\n' || r == '\r'
}

// PostProcess merges cmd.exe numeric stream-redirect operators. The
// tokenizer produces e.g. `2` as a WORD_TOKEN and `>` (or `>>`) as a
// WORDBREAK_TOKEN. This step detects adjacent word+wordbreak sequences
// like `2>`, `2>>`, `2>&1`, `1>&2` and reclassifies them as single
// WORDBREAK_TOKENs with the appropriate WordbreakType.
func (cmdFormat) PostProcess(tokens TokenSlice) TokenSlice {
	result := make(TokenSlice, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		// Look for bare WORD_TOKEN (digit 1-2) immediately followed by
		// WORDBREAK_TOKEN starting with '>' (redirect operator)
		if t.Type == WORD_TOKEN && t.Value == t.RawValue && i+1 < len(tokens) {
			next := tokens[i+1]
			if next.Type == WORDBREAK_TOKEN && next.adjoins(t) &&
				next.WordbreakType.IsRedirect() && len(next.RawValue) > 0 && next.RawValue[0] == '>' {
				if len(t.Value) == 1 && (t.Value[0] == '1' || t.Value[0] == '2') {
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
