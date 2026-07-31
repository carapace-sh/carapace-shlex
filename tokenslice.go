package shlex

import (
	"strconv"
)

type TokenSlice []Token

func (t TokenSlice) Strings() []string {
	s := make([]string, 0, len(t))
	for _, token := range t {
		s = append(s, token.Value)
	}
	return s
}

func (t TokenSlice) Pipelines() []TokenSlice {
	pipelines := make([]TokenSlice, 0)

	pipeline := make(TokenSlice, 0)
	depth := 0
	for _, token := range t {
		switch {
		case token.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN:
			depth++
			pipeline = append(pipeline, token)
		case token.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE:
			depth--
			pipeline = append(pipeline, token)
		case depth > 0:
			pipeline = append(pipeline, token)
		case token.Type == WORDBREAK_TOKEN && token.WordbreakType.IsPipelineDelimiter():
			pipelines = append(pipelines, pipeline)
			pipeline = make(TokenSlice, 0)
		default:
			pipeline = append(pipeline, token)
		}
	}
	return append(pipelines, pipeline)
}

func (t TokenSlice) CurrentPipeline() TokenSlice {
	pipelines := t.Pipelines()
	return pipelines[len(pipelines)-1]
}

func (t TokenSlice) Words() TokenSlice {
	words := make(TokenSlice, 0)
	for index, token := range t {
		switch {
		case index == 0:
			words = append(words, token)
		case t[index-1].adjoins(token):
			words[len(words)-1].Value += token.Value
			words[len(words)-1].RawValue += token.RawValue
			words[len(words)-1].Span.End = token.Span.End
			words[len(words)-1].State = token.State
		default:
			words = append(words, token)
		}
	}
	return words
}

func (t TokenSlice) FilterRedirects() TokenSlice {
	filtered := make(TokenSlice, 0)
	for index, token := range t {
		switch token.Type {
		case WORDBREAK_TOKEN:
			if token.WordbreakType.IsRedirect() {
				continue
			}
		}

		if index > 0 {
			if t[index-1].WordbreakType.IsRedirect() {
				continue
			}
		}

		if index < len(t)-1 {
			next := t[index+1]
			if token.adjoins(next) {
				if _, err := strconv.Atoi(token.RawValue); err == nil {
					if t[index+1].WordbreakType.IsRedirect() {
						continue
					}
				}
			}
		}

		filtered = append(filtered, token)
	}
	return filtered
}

// WordsWithSubstitutions merges tokens into words, treating closed
// substitution scopes as single words. When a WORDBREAK_SUBSTITUTION_OPEN
// is encountered, all tokens until the matching WORDBREAK_SUBSTITUTION_CLOSE
// are merged into one word. Unclosed substitution scopes (cursor inside)
// are left as separate tokens — the caller should use the inner tokens
// to build a separate completion context.
func (t TokenSlice) WordsWithSubstitutions() TokenSlice {
	words := make(TokenSlice, 0)
	depth := 0
	var sub *Token

	for index, token := range t {
		switch {
		case token.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN:
			if depth == 0 {
				sub = &Token{
					Type: WORD_TOKEN,
					Span: token.Span,
				}
			}
			depth++
			if sub != nil {
				sub.RawValue += token.RawValue
				sub.Value += token.RawValue
				sub.Span.End = token.Span.End
				sub.State = token.State
			}

		case token.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE:
			if depth > 0 {
				if sub != nil {
					sub.RawValue += token.RawValue
					sub.Value += token.RawValue
					sub.Span.End = token.Span.End
					sub.State = token.State
				}
				depth--
				if depth == 0 {
					words = append(words, *sub)
					sub = nil
				}
			}

		case depth > 0:
			if sub != nil {
				// Insert a space when tokens don't adjoin (gap = space
				// tokens that the lexer skipped). The substitution word
				// uses RawValue for both Value and RawValue since the
				// inner content is opaque to the outer command.
				if sub.Span.End != token.Span.Start && sub.RawValue != "" {
					sub.RawValue += " "
					sub.Value += " "
				}
				sub.RawValue += token.RawValue
				sub.Value += token.RawValue
				sub.Span.End = token.Span.End
				sub.State = token.State
			}

		default:
			if index == 0 {
				words = append(words, token)
			} else if t[index-1].adjoins(token) {
				words[len(words)-1].Value += token.Value
				words[len(words)-1].RawValue += token.RawValue
				words[len(words)-1].Span.End = token.Span.End
				words[len(words)-1].State = token.State
			} else {
				words = append(words, token)
			}
		}
	}

	if depth > 0 {
		// Unclosed substitution — don't append the partial word;
		// the inner context will be built separately.
		return words
	}
	return words
}

func (t TokenSlice) CurrentToken() (token Token) {
	if len(t) > 0 {
		token = t[len(t)-1]
	}
	return
}

func (t TokenSlice) WordbreakPrefix() string {
	if len(t) == 0 {
		return ""
	}
	found := false
	prefix := ""

	last := t[len(t)-1]
	switch last.State {
	case QUOTING_STATE, QUOTING_ESCAPING_STATE, ESCAPING_QUOTED_STATE,
		QUOTING_TRIPLE_STATE, QUOTING_TRIPLE_ESCAPING_STATE:
		// Seems bash handles the last opening quote as wordbreak when in quoting state.
		// So add value up to last opening quote to prefix.
		found = true
		prefix = last.Value[:last.WordbreakIndex]
	}

	for i := len(t) - 2; i >= 0; i-- {
		token := t[i]
		if !token.adjoins(t[i+1]) {
			break
		}

		if !found && token.Type == WORDBREAK_TOKEN {
			found = true
			if token.Value == "@" {
				// Seems although `@` is a wordbreak, it weirdly is not part of the prefix.
				continue
			}
		}

		if found {
			prefix = token.Value + prefix
		}
	}
	return prefix
}
