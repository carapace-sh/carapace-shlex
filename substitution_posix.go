package shlex

// posixSubstitutionPostProcess reclassifies ( and ) as substitution
// delimiters and merges $ + ( adjacency into a single opener token.
//
// In POSIX shells (bash, zsh, oil, tcsh), ( and ) are in BASH_WORDBREAKS
// (or TCSH_WORDBREAKS) and classified as WORDBREAK_UNKNOWN by the operator
// grammar. This post-pass:
//
//  1. Merges a $ WORD_TOKEN immediately followed by an adjacent ( WORDBREAK
//     into a single WORDBREAK_SUBSTITUTION_OPEN token with RawValue "$(".
//  2. Detects $(( for arithmetic by checking if the next token after $(
//     is another ( — if so, merges both into a single opener "$((".
//  3. Reclassifies standalone ( as WORDBREAK_SUBSTITUTION_OPEN (for
//     process substitution <(, >, and bare () in csh).
//  4. Reclassifies ) as WORDBREAK_SUBSTITUTION_CLOSE.
//     For arithmetic $((...)), the first ) closes arithmetic, the second
//     closes the command substitution — detected by tracking depth and
//     the opener's RawValue.
func posixSubstitutionPostProcess(tokens TokenSlice) TokenSlice {
	result := make(TokenSlice, 0, len(tokens))
	depth := 0
	arithmeticDepth := 0 // tracks how many openers were arithmetic ($(( )

	i := 0
	for i < len(tokens) {
		t := tokens[i]

		// Detect $ + ( adjacency → merge into WORDBREAK_SUBSTITUTION_OPEN
		if t.Type == WORD_TOKEN && t.Value == "$" && i+1 < len(tokens) {
			next := tokens[i+1]
			if next.Type == WORDBREAK_TOKEN && next.Value == "(" && t.adjoins(next) {
				// Check for $(( arithmetic
				if i+2 < len(tokens) && tokens[i+2].Type == WORDBREAK_TOKEN &&
					tokens[i+2].Value == "(" && next.adjoins(tokens[i+2]) {
					merged := Token{
						Type:          WORDBREAK_TOKEN,
						Value:         "$(",
						RawValue:      t.RawValue + next.RawValue + tokens[i+2].RawValue,
						Span:          Span{Start: t.Span.Start, End: tokens[i+2].Span.End},
						State:         tokens[i+2].State,
						WordbreakType: WORDBREAK_SUBSTITUTION_OPEN,
					}
					result = append(result, merged)
					depth++
					arithmeticDepth++
					i += 3
					continue
				}
				// Regular $( command substitution
				merged := Token{
					Type:          WORDBREAK_TOKEN,
					Value:         "$(",
					RawValue:      t.RawValue + next.RawValue,
					Span:          Span{Start: t.Span.Start, End: next.Span.End},
					State:         next.State,
					WordbreakType: WORDBREAK_SUBSTITUTION_OPEN,
				}
				result = append(result, merged)
				depth++
				i += 2
				continue
			}
		}

		// Detect process substitution: <( or >( — merge redirect operator + ( into opener
		if t.Type == WORDBREAK_TOKEN && t.WordbreakType.IsRedirect() &&
			(t.Value == "<" || t.Value == ">") && i+1 < len(tokens) {
			next := tokens[i+1]
			if next.Type == WORDBREAK_TOKEN && next.Value == "(" && t.adjoins(next) {
				merged := Token{
					Type:          WORDBREAK_TOKEN,
					Value:         t.Value + "(",
					RawValue:      t.RawValue + next.RawValue,
					Span:          Span{Start: t.Span.Start, End: next.Span.End},
					State:         next.State,
					WordbreakType: WORDBREAK_SUBSTITUTION_OPEN,
				}
				result = append(result, merged)
				depth++
				i += 2
				continue
			}
		}

		// Reclassify standalone ( as WORDBREAK_SUBSTITUTION_OPEN
		// (for bare () in csh)
		if t.Type == WORDBREAK_TOKEN && t.Value == "(" {
			merged := Token{
				Type:          t.Type,
				Value:         t.Value,
				RawValue:      t.RawValue,
				Span:          t.Span,
				State:         t.State,
				WordbreakType: WORDBREAK_SUBSTITUTION_OPEN,
			}
			result = append(result, merged)
			depth++
			i++
			continue
		}

		// Reclassify ) as WORDBREAK_SUBSTITUTION_CLOSE
		if t.Type == WORDBREAK_TOKEN && t.Value == ")" {
			if arithmeticDepth > 0 && depth == arithmeticDepth {
				// This ) closes the arithmetic part of $((
				// but we need two ) to fully close $((...))
				// Check if the next token is also )
				if i+1 < len(tokens) && tokens[i+1].Type == WORDBREAK_TOKEN &&
					tokens[i+1].Value == ")" && t.adjoins(tokens[i+1]) {
					// Merge both ) into a single closer
					merged := Token{
						Type:          WORDBREAK_TOKEN,
						Value:         "))",
						RawValue:      t.RawValue + tokens[i+1].RawValue,
						Span:          Span{Start: t.Span.Start, End: tokens[i+1].Span.End},
						State:         tokens[i+1].State,
						WordbreakType: WORDBREAK_SUBSTITUTION_CLOSE,
					}
					result = append(result, merged)
					depth--
					arithmeticDepth--
					i += 2
					continue
				}
				// Single ) when arithmetic expected — treat as close
				merged := Token{
					Type:          t.Type,
					Value:         t.Value,
					RawValue:      t.RawValue,
					Span:          t.Span,
					State:         t.State,
					WordbreakType: WORDBREAK_SUBSTITUTION_CLOSE,
				}
				result = append(result, merged)
				depth--
				arithmeticDepth--
				i++
				continue
			}
			merged := Token{
				Type:          t.Type,
				Value:         t.Value,
				RawValue:      t.RawValue,
				Span:          t.Span,
				State:         t.State,
				WordbreakType: WORDBREAK_SUBSTITUTION_CLOSE,
			}
			result = append(result, merged)
			depth--
			i++
			continue
		}

		// Pass through everything else
		result = append(result, t)
		i++
	}

	return result
}
