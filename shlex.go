package shlex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// TokenType is a top-level token classification: A word, space, comment, unknown.
type TokenType int

func (t TokenType) MarshalJSON() ([]byte, error) {
	return json.Marshal(tokenTypes[t])
}

// runeTokenClass is the type of a UTF-8 character classification: A quote, space, escape.
type runeTokenClass int

// the internal state used by the lexer state machine
type LexerState int

func (l LexerState) MarshalJSON() ([]byte, error) {
	return json.Marshal(lexerStates[l])
}

// Token is a (type, value) pair representing a lexicographic token.
type Token struct {
	Type           TokenType
	Value          string
	RawValue       string
	Span           Span
	State          LexerState
	WordbreakType  WordbreakType `json:",omitempty"`
	WordbreakIndex int           // index of last opening quote in Value (only correct when in quoting state)
}

func (t *Token) add(r rune) {
	t.Value += string(r)
}

func (t *Token) removeLastRaw() {
	runes := []rune(t.RawValue)
	t.RawValue = string(runes[:len(runes)-1])
}

func (t Token) adjoins(other Token) bool {
	return t.Span.End == other.Span.Start || t.Span.Start == other.Span.End
}

// Equal reports whether tokens a, and b, are equal.
// Two tokens are equal if both their types and values are equal. A nil token can
// never be equal to another token.
func (t *Token) Equal(other *Token) bool {
	switch {
	case t == nil,
		other == nil,
		t.Type != other.Type,
		t.Value != other.Value,
		t.RawValue != other.RawValue,
		t.Span != other.Span,
		t.State != other.State,
		t.WordbreakType != other.WordbreakType,
		t.WordbreakIndex != other.WordbreakIndex:
		return false
	default:
		return true
	}
}

// Named classes of UTF-8 runes
const (
	spaceRunes            = " \t\r\n"
	escapingQuoteRunes    = `"`
	nonEscapingQuoteRunes = "'"
	escapeRunes           = `\`
	commentRunes          = "#"
)

// Classes of rune token
const (
	unknownRuneClass runeTokenClass = iota
	spaceRuneClass
	escapingQuoteRuneClass
	nonEscapingQuoteRuneClass
	escapeRuneClass
	commentRuneClass
	wordbreakRuneClass
	eofRuneClass
)

// Classes of lexographic token
const (
	UNKNOWN_TOKEN TokenType = iota
	WORD_TOKEN
	SPACE_TOKEN
	COMMENT_TOKEN
	WORDBREAK_TOKEN
)

var tokenTypes = map[TokenType]string{
	UNKNOWN_TOKEN:   "UNKNOWN_TOKEN",
	WORD_TOKEN:      "WORD_TOKEN",
	SPACE_TOKEN:     "SPACE_TOKEN",
	COMMENT_TOKEN:   "COMMENT_TOKEN",
	WORDBREAK_TOKEN: "WORDBREAK_TOKEN",
}

// Lexer state machine states
const (
	START_STATE                   LexerState = iota // no runes have been seen
	IN_WORD_STATE                                   // processing regular runes in a word
	ESCAPING_STATE                                  // we have just consumed an escape rune; the next rune is literal
	ESCAPING_QUOTED_STATE                           // we have just consumed an escape rune within a quoted string
	QUOTING_ESCAPING_STATE                          // we are within a quoted string that supports escaping ("...")
	QUOTING_STATE                                   // we are within a string that does not support escaping ('...')
	QUOTING_TRIPLE_STATE                            // we are within a triple-quoted non-escaping string ('''...''')
	QUOTING_TRIPLE_ESCAPING_STATE                   // we are within a triple-quoted escaping string ("""...""")
	COMMENT_STATE                                   // we are within a comment (everything following an unquoted or unescaped #
	BLOCK_COMMENT_STATE                             // we are within a block comment (e.g. PowerShell <# ... #>)
	STOP_PARSING_STATE                              // we are in raw mode after a stop-parsing token (e.g. PowerShell --%)
	WORDBREAK_STATE                                 // we have just consumed a wordbreak rune
)

var lexerStates = map[LexerState]string{
	START_STATE:                   "START_STATE",
	IN_WORD_STATE:                 "IN_WORD_STATE",
	ESCAPING_STATE:                "ESCAPING_STATE",
	ESCAPING_QUOTED_STATE:         "ESCAPING_QUOTED_STATE",
	QUOTING_ESCAPING_STATE:        "QUOTING_ESCAPING_STATE",
	QUOTING_STATE:                 "QUOTING_STATE",
	QUOTING_TRIPLE_STATE:          "QUOTING_TRIPLE_STATE",
	QUOTING_TRIPLE_ESCAPING_STATE: "QUOTING_TRIPLE_ESCAPING_STATE",
	COMMENT_STATE:                 "COMMENT_STATE",
	BLOCK_COMMENT_STATE:           "BLOCK_COMMENT_STATE",
	STOP_PARSING_STATE:            "STOP_PARSING_STATE",
	WORDBREAK_STATE:               "WORDBREAK_STATE",
}

// tokenClassifier is used for classifying rune characters.
type tokenClassifier map[rune]runeTokenClass

func (typeMap tokenClassifier) addRuneClass(runes string, tokenType runeTokenClass) {
	for _, runeChar := range runes {
		typeMap[runeChar] = tokenType
	}
}

// ClassifyRune classifies a rune
func (t tokenClassifier) ClassifyRune(runeVal rune) runeTokenClass {
	return t[runeVal]
}

// newBaseClassifier creates a classifier with the standard POSIX rune classes
// (space, escaping quote, non-escaping quote, escape, comment) but without
// any wordbreak runes. Formats add their own wordbreaks on top.
func newBaseClassifier(escapeChar string) tokenClassifier {
	t := tokenClassifier{}
	t.addRuneClass(spaceRunes, spaceRuneClass)
	t.addRuneClass(escapingQuoteRunes, escapingQuoteRuneClass)
	t.addRuneClass(nonEscapingQuoteRunes, nonEscapingQuoteRuneClass)
	t.addRuneClass(escapeChar, escapeRuneClass)
	t.addRuneClass(commentRunes, commentRuneClass)
	return t
}

// addWordbreaks adds wordbreak runes to a classifier, filtering out any
// that are already classified as space/quote/escape/comment.
func (t tokenClassifier) addWordbreaks(wordbreakRunes string) {
	filtered := make([]rune, 0)
	for _, r := range wordbreakRunes {
		if t.ClassifyRune(r) == unknownRuneClass {
			filtered = append(filtered, r)
		}
	}
	t.addRuneClass(string(filtered), wordbreakRuneClass)
}

// lexer turns an input stream into a sequence of tokens. Whitespace and comments are skipped.
type lexer tokenizer

// newLexer creates a new lexer from an input stream and format.
func newLexer(r io.Reader, format Format) *lexer {
	return (*lexer)(newTokenizer(r, format))
}

// Next returns the next token, or an error. If there are no more tokens,
// the error will be io.EOF.
func (l *lexer) Next() (*Token, error) {
	for {
		token, err := (*tokenizer)(l).Next()
		if err != nil {
			return token, err
		}
		switch token.Type {
		case WORD_TOKEN, WORDBREAK_TOKEN:
			return token, nil
		case COMMENT_TOKEN:
			// skip comments
		default:
			return nil, fmt.Errorf("unknown token type: %v", token.Type)
		}
	}
}

// tokenizer turns an input stream into a sequence of typed tokens
type tokenizer struct {
	input            bufio.Reader
	classifier       tokenClassifier
	format           Format
	index            int
	state            LexerState
	rawQuote         bool   // true when current quote was opened with a raw prefix (r/R)
	tripleQuoteRune  rune   // the quote char (' or ") that opened a triple-quote
	blockCloser      string // the closer string for the current block comment
	blockCloserIdx   int    // index into blockCloser for match tracking
	stopParsingDelim string // pipeline delimiter set for stop-parsing mode
}

func (t *tokenizer) ReadRune() (r rune, size int, err error) {
	if r, size, err = t.input.ReadRune(); err == nil {
		t.index += 1
	}
	return
}

func (t *tokenizer) UnreadRune() (err error) {
	if err = t.input.UnreadRune(); err == nil {
		t.index -= 1
	}
	return
}

// newTokenizer creates a new tokenizer from an input stream and format.
func newTokenizer(r io.Reader, format Format) *tokenizer {
	input := bufio.NewReader(r)
	classifier := format.Classifier()
	return &tokenizer{
		input:      *input,
		classifier: classifier,
		format:     format}
}

// checkTripleQuote peeks ahead two runes to check if this is a triple-quote.
// Returns true if the next two runes match the quote rune. The two runes are
// consumed (added to index) and returned so the caller can add them to RawValue.
// If not a triple-quote, all peeked runes are unread (at most one unread is
// needed since bufio.Reader only supports one level of UnreadRune; the first
// rune is returned via the consumedRune parameter so the caller can handle it).
func (t *tokenizer) checkTripleQuote(quote rune) (isTriple bool, r1 rune, r2 rune, consumedRune rune) {
	if !t.format.TripleQuoteSupport() {
		return false, 0, 0, 0
	}
	peek1, _, err1 := t.ReadRune()
	if err1 != nil {
		return false, 0, 0, 0
	}
	if peek1 != quote {
		t.UnreadRune()
		return false, 0, 0, 0
	}
	peek2, _, err2 := t.ReadRune()
	if err2 != nil {
		// peek1 matched but EOF after — can't unread peek1, return it as consumed
		return false, 0, 0, peek1
	}
	if peek2 != quote {
		// peek1 matched but peek2 didn't — unread peek2, return peek1 as consumed
		t.UnreadRune()
		return false, 0, 0, peek1
	}
	return true, peek1, peek2, 0
}

// checkRawPrefix examines the current token's Value to see if it ends with
// a raw string prefix (r or R) that qualifies as a Python string prefix.
// The r/R must be at word start or preceded only by other valid prefix
// characters (b, B, p, P, f, F, u, U, r, R). This prevents false positives
// like "abr" where the r is part of a regular word.
func (t *tokenizer) checkRawPrefix(token *Token) bool {
	if !t.format.RawPrefixSupport() {
		return false
	}
	val := token.Value
	if len(val) == 0 {
		return false
	}
	// Check that all chars in Value are valid string prefix chars
	for i := 0; i < len(val); i++ {
		if !isStringPrefixChar(val[i]) {
			return false
		}
	}
	// Must end with r or R
	last := val[len(val)-1]
	return last == 'r' || last == 'R'
}

// isStringPrefixChar returns true for characters valid in Python string
// prefixes: b, B, p, P, r, R, u, U, f, F.
func isStringPrefixChar(c byte) bool {
	switch c {
	case 'b', 'B', 'p', 'P', 'r', 'R', 'u', 'U', 'f', 'F':
		return true
	}
	return false
}

// checkTripleClose checks if the current quote rune (already consumed and added
// to RawValue at the top of the loop) is the start of a closing triple-quote.
// Reads two more runes. Returns:
//   - closed=true: both runes matched, closing triple-quote consumed (added to RawValue by caller)
//   - closed=false, consumedRune!=0: first rune matched but second didn't. The first rune was
//     consumed and cannot be unread; it's returned as consumedRune so the caller can add it
//     to RawValue and emit it as a literal. The second rune was unread.
//   - closed=false, consumedRune=0: first rune didn't match (or EOF), it was unread.
//     The caller should emit the current quote rune as a literal.
func (t *tokenizer) checkTripleClose() (closed bool, r1 rune, r2 rune, consumedRune rune) {
	peek1, _, err1 := t.ReadRune()
	if err1 != nil {
		return false, 0, 0, 0
	}
	if peek1 != t.tripleQuoteRune {
		t.UnreadRune()
		return false, 0, 0, 0
	}
	peek2, _, err2 := t.ReadRune()
	if err2 != nil {
		return false, 0, 0, peek1
	}
	if peek2 != t.tripleQuoteRune {
		t.UnreadRune()
		return false, 0, 0, peek1
	}
	return true, peek1, peek2, 0
}

// checkBlockCommentOpener checks if the current position matches a block
// comment opener (e.g. "<#" for PowerShell). The firstRune has already been
// consumed and added to token.RawValue by the caller. If the remaining runes
// of the opener match, they are consumed and added to token.RawValue, and
// the tokenizer enters BLOCK_COMMENT_STATE. Returns true if the opener was
// matched and consumed. On mismatch, peeked runes are unread.
func (t *tokenizer) checkBlockCommentOpener(firstRune rune, token *Token) bool {
	bc, ok := t.format.(BlockCommenter)
	if !ok {
		return false
	}
	opener := bc.BlockCommentOpener()
	if len(opener) == 0 || rune(opener[0]) != firstRune {
		return false
	}
	// Try to match remaining runes of the opener (index 1 onward)
	for i := 1; i < len(opener); i++ {
		r, _, err := t.ReadRune()
		if err != nil {
			// EOF before full match — unread what we consumed
			for j := 0; j < i-1; j++ {
				t.UnreadRune()
			}
			return false
		}
		if rune(opener[i]) != r {
			// Mismatch — unread this rune and any previously consumed runes
			t.UnreadRune()
			for j := 0; j < i-1; j++ {
				t.UnreadRune()
			}
			return false
		}
	}
	// Full match — add remaining opener runes to RawValue
	for i := 1; i < len(opener); i++ {
		token.RawValue += string(opener[i])
	}
	t.blockCloser = bc.BlockCommentCloser()
	t.blockCloserIdx = 0
	return true
}

// scanStream scans the stream for the next token using the internal state machine.
// It will panic if it encounters a rune which it does not know how to handle.
func (t *tokenizer) scanStream() (*Token, error) {
	previousState := t.state
	t.state = START_STATE
	t.rawQuote = false
	t.tripleQuoteRune = 0
	token := &Token{}
	var nextRune rune
	var nextRuneType runeTokenClass
	var err error
	consumed := 0

	for {
		nextRune, _, err = t.ReadRune()
		nextRuneType = t.classifier.ClassifyRune(nextRune)
		token.RawValue += string(nextRune)
		consumed += 1 // TODO find a nicer solution for this

		switch {
		case err == io.EOF:
			nextRuneType = eofRuneClass
			err = nil
		case err != nil:
			return nil, err
		}

		switch t.state {
		case START_STATE: // no runes read yet
			{
				if nextRuneType != spaceRuneClass {
					token.Span.Start = t.index - 1
				}
				// Check for block comment opener before other classification
				if t.checkBlockCommentOpener(nextRune, token) {
					token.Type = COMMENT_TOKEN
					t.state = BLOCK_COMMENT_STATE
					continue
				}
				switch nextRuneType {
				case eofRuneClass:
					switch {
					case t.index == 0: // tokenizer contains an empty string
						token.removeLastRaw()
						token.Type = WORD_TOKEN
						token.Span.Start = t.index
						token.Span.End = t.index
						t.index += 1
						return token, nil // return an additional empty token for current cursor position
					case previousState == WORDBREAK_STATE, consumed > 1: // consumed is greater than 1 when when there were spaceRunes before
						token.removeLastRaw()
						token.Type = WORD_TOKEN
						token.Span.Start = t.index
						token.Span.End = t.index
						return token, nil // return an additional empty token for current cursor position
					default:
						return nil, io.EOF
					}
				case spaceRuneClass:
					token.removeLastRaw()
				case escapingQuoteRuneClass:
					token.Type = WORD_TOKEN
					token.WordbreakIndex = len(token.Value)
					if isTriple, r1, r2, consumed := t.checkTripleQuote(nextRune); isTriple {
						token.RawValue += string(r1)
						token.RawValue += string(r2)
						t.tripleQuoteRune = nextRune
						if t.checkRawPrefix(token) {
							t.rawQuote = true
							t.state = QUOTING_TRIPLE_STATE
						} else {
							t.state = QUOTING_TRIPLE_ESCAPING_STATE
						}
					} else if consumed != 0 {
						token.RawValue += string(consumed)
						t.state = IN_WORD_STATE
					} else if t.checkRawPrefix(token) {
						t.rawQuote = true
						t.state = QUOTING_ESCAPING_STATE
					} else {
						t.state = QUOTING_ESCAPING_STATE
					}
				case nonEscapingQuoteRuneClass:
					token.Type = WORD_TOKEN
					token.WordbreakIndex = len(token.Value)
					if isTriple, r1, r2, consumed := t.checkTripleQuote(nextRune); isTriple {
						token.RawValue += string(r1)
						token.RawValue += string(r2)
						t.tripleQuoteRune = nextRune
						t.state = QUOTING_TRIPLE_STATE
					} else if consumed != 0 {
						token.RawValue += string(consumed)
						t.state = IN_WORD_STATE
					} else {
						t.state = QUOTING_STATE
					}
				case escapeRuneClass:
					token.Type = WORD_TOKEN
					if t.format.EscapeNotBareword() {
						// Check for line continuation (e.g. PowerShell backtick + newline)
						if lc, ok := t.format.(LineContinuationEscaper); ok {
							peekRune, _, peekErr := t.ReadRune()
							if peekErr != nil {
								// EOF after escape — enter ESCAPING_STATE to handle
								t.UnreadRune() // can't unread EOF, but harmless
								_ = peekRune
								t.state = ESCAPING_STATE
								continue
							}
							if lc.IsLineContinuation(peekRune) {
								// Consume optional \n after \r
								if peekRune == '\r' {
									peek2, _, peek2Err := t.ReadRune()
									if peek2Err == nil && peek2 == '\n' {
										// CRLF consumed — don't add to RawValue
									} else if peek2Err == nil {
										t.UnreadRune()
									}
								}
								// Line continuation: remove escape char from RawValue and Value
								token.removeLastRaw() // remove the escape char
								// Stay in START_STATE (no word content yet)
								continue
							}
							// Not a line continuation — unread and enter ESCAPING_STATE
							t.UnreadRune()
						}
						t.state = ESCAPING_STATE
					} else {
						token.add(nextRune)
						t.state = IN_WORD_STATE
					}
				case commentRuneClass:
					token.Type = COMMENT_TOKEN
					t.state = COMMENT_STATE
				case wordbreakRuneClass:
					token.Type = WORDBREAK_TOKEN
					token.add(nextRune)
					t.state = WORDBREAK_STATE
				default:
					token.Type = WORD_TOKEN
					token.add(nextRune)
					t.state = IN_WORD_STATE
				}
			}
		case WORDBREAK_STATE:
			// Check for block comment opener: the current wordbreak token
			// may be the first rune of the opener (e.g. "<" in "<#").
			// token.RawValue includes nextRune (added at top of loop), so
			// we check if the RawValue without nextRune matches the opener start.
			if bc, ok := t.format.(BlockCommenter); ok {
				opener := bc.BlockCommentOpener()
				if len(opener) > 0 {
					// The wordbreak portion is token.Value (without nextRune).
					// token.RawValue includes nextRune; we need to check if
					// token.Value (the wordbreak so far) is the opener prefix.
					wordbreakPart := token.Value
					if wordbreakPart == string(opener[0]) && len(opener) > 1 {
						// nextRune should be opener[1]
						if rune(opener[1]) == nextRune {
							// Match remaining opener runes from index 2
							matched := true
							for i := 2; i < len(opener); i++ {
								r, _, e := t.ReadRune()
								if e != nil || rune(opener[i]) != r {
									if e == nil {
										t.UnreadRune()
									}
									for j := 0; j < i-2; j++ {
										t.UnreadRune()
									}
									matched = false
									break
								}
							}
							if matched {
								// Add remaining opener runes to RawValue
								for i := 2; i < len(opener); i++ {
									token.RawValue += string(opener[i])
								}
								token.Type = COMMENT_TOKEN
								t.blockCloser = bc.BlockCommentCloser()
								t.blockCloserIdx = 0
								t.state = BLOCK_COMMENT_STATE
								continue
							}
						}
					}
				}
			}
			switch nextRuneType {
			case wordbreakRuneClass:
				// token.RawValue already includes nextRune (added at top of loop).
				// If the extended raw value is not a known operator, the current
				// rune starts a new operator — unread it and return the current token.
				if t.format.ClassifyOperator(token.RawValue) == WORDBREAK_UNKNOWN {
					token.removeLastRaw()
					t.UnreadRune()
					return token, err
				}
				token.add(nextRune)
			default:
				token.removeLastRaw()
				t.UnreadRune()
				return token, err
			}
		case IN_WORD_STATE: // in a regular word
			switch nextRuneType {
			case wordbreakRuneClass:
				token.removeLastRaw()
				t.UnreadRune()
				return token, err
			case eofRuneClass, spaceRuneClass:
				token.removeLastRaw()
				t.UnreadRune()
				return token, err
			case escapingQuoteRuneClass:
				token.WordbreakIndex = len(token.Value)
				if isTriple, r1, r2, consumed := t.checkTripleQuote(nextRune); isTriple {
					token.RawValue += string(r1)
					token.RawValue += string(r2)
					t.tripleQuoteRune = nextRune
					if t.checkRawPrefix(token) {
						t.rawQuote = true
						t.state = QUOTING_TRIPLE_STATE
					} else {
						t.state = QUOTING_TRIPLE_ESCAPING_STATE
					}
				} else if consumed != 0 {
					token.RawValue += string(consumed)
					t.state = IN_WORD_STATE
				} else if t.checkRawPrefix(token) {
					t.rawQuote = true
					t.state = QUOTING_ESCAPING_STATE
				} else {
					t.state = QUOTING_ESCAPING_STATE
				}
			case nonEscapingQuoteRuneClass:
				token.WordbreakIndex = len(token.Value)
				if isTriple, r1, r2, consumed := t.checkTripleQuote(nextRune); isTriple {
					token.RawValue += string(r1)
					token.RawValue += string(r2)
					t.tripleQuoteRune = nextRune
					t.state = QUOTING_TRIPLE_STATE
				} else if consumed != 0 {
					token.RawValue += string(consumed)
					t.state = IN_WORD_STATE
				} else {
					t.state = QUOTING_STATE
				}
			case escapeRuneClass:
				if t.format.EscapeNotBareword() {
					// Check for line continuation (e.g. PowerShell backtick + newline)
					if lc, ok := t.format.(LineContinuationEscaper); ok {
						peekRune, _, peekErr := t.ReadRune()
						if peekErr != nil {
							// EOF after escape — enter ESCAPING_STATE to handle
							t.UnreadRune()
							_ = peekRune
							t.state = ESCAPING_STATE
							continue
						}
						if lc.IsLineContinuation(peekRune) {
							// Consume optional \n after \r
							if peekRune == '\r' {
								peek2, _, peek2Err := t.ReadRune()
								if peek2Err == nil && peek2 == '\n' {
									// CRLF consumed — don't add to RawValue
								} else if peek2Err == nil {
									t.UnreadRune()
								}
							}
							// Line continuation: remove escape char from RawValue
							token.removeLastRaw() // remove the escape char
							// Stay in IN_WORD_STATE (word continues on next line)
							continue
						}
						// Not a line continuation — unread and enter ESCAPING_STATE
						t.UnreadRune()
					}
					t.state = ESCAPING_STATE
				} else {
					token.add(nextRune) // elvish: \ is a bareword char
				}
			default:
				token.add(nextRune)
			}
		case ESCAPING_STATE: // the rune after an escape character
			switch nextRuneType {
			case eofRuneClass: // EOF found after escape character
				token.removeLastRaw()
				return token, err
			default:
				// Check for line continuation (e.g. PowerShell backtick + newline)
				if lc, ok := t.format.(LineContinuationEscaper); ok && lc.IsLineContinuation(nextRune) {
					// Consume optional \n after \r (without adding to RawValue)
					if nextRune == '\r' {
						peek2, _, peek2Err := t.ReadRune()
						if peek2Err == nil && peek2 == '\n' {
							// CRLF consumed
						} else if peek2Err == nil {
							t.UnreadRune()
						}
					}
					// Line continuation: remove newline and escape char from RawValue
					token.removeLastRaw() // remove newline (\n or \r)
					token.removeLastRaw() // remove escape char
					// If we have word content, continue in IN_WORD_STATE;
					// otherwise go back to START_STATE
					if len(token.Value) > 0 {
						t.state = IN_WORD_STATE
					} else {
						t.state = START_STATE
					}
					continue
				}
				t.state = IN_WORD_STATE
				token.add(nextRune)
			}
		case ESCAPING_QUOTED_STATE: // the next rune after an escape character, in double quotes
			switch nextRuneType {
			case eofRuneClass: // EOF found after escape character
				token.removeLastRaw()
				return token, err
			default:
				t.state = QUOTING_ESCAPING_STATE
				if unescaper, ok := t.format.(EscapingQuoteUnescaper); ok {
					if replacement, handled := unescaper.EscapingQuoteUnescape(nextRune); handled {
						token.Value += replacement
					} else {
						token.add('\\')
						token.add(nextRune)
					}
				} else if escapeChars := t.format.EscapingQuoteEscapeChars(); escapeChars != nil {
					if escapeChars[nextRune] {
						token.add(nextRune)
					} else {
						token.add('\\')
						token.add(nextRune)
					}
				} else {
					token.add(nextRune)
				}
			}
		case QUOTING_ESCAPING_STATE: // in escaping double quotes
			switch nextRuneType {
			case eofRuneClass: // EOF found when expecting closing quote
				token.removeLastRaw()
				return token, err
			case escapingQuoteRuneClass:
				if t.format.NonEscapingQuoteEscapes() {
					// PowerShell: "" → literal " (doubled quote), else close
					peekRune, _, peekErr := t.ReadRune()
					if peekErr == nil && t.classifier.ClassifyRune(peekRune) == escapingQuoteRuneClass {
						token.RawValue += string(peekRune)
						token.add(nextRune) // emit one literal "
						// stay in QUOTING_ESCAPING_STATE
					} else {
						if peekErr == nil {
							t.UnreadRune()
						}
						t.rawQuote = false
						t.state = IN_WORD_STATE
					}
				} else {
					t.rawQuote = false
					t.state = IN_WORD_STATE
				}
			case escapeRuneClass:
				if t.rawQuote {
					token.add(nextRune) // raw string: backslash is literal
				} else if t.format.EscapeNotInEscapingQuote() {
					token.add(nextRune) // cmd: caret is literal inside double quotes
				} else {
					t.state = ESCAPING_QUOTED_STATE
				}
			default:
				token.add(nextRune)
			}
		case QUOTING_STATE: // in non-escaping single quotes
			switch nextRuneType {
			case eofRuneClass: // EOF found when expecting closing quote
				token.removeLastRaw()
				return token, err
			case nonEscapingQuoteRuneClass:
				t.rawQuote = false
				if t.format.NonEscapingQuoteEscapes() {
					// Peek: '' → literal ' (stay in quote), else close
					peekRune, _, peekErr := t.ReadRune()
					if peekErr == nil && t.classifier.ClassifyRune(peekRune) == nonEscapingQuoteRuneClass {
						token.RawValue += string(peekRune)
						token.add(nextRune) // emit one literal '
						// stay in QUOTING_STATE
					} else {
						// Not a doubled quote — unread and close
						if peekErr == nil {
							t.UnreadRune()
						}
						t.state = IN_WORD_STATE
					}
				} else {
					t.state = IN_WORD_STATE
				}
			case escapeRuneClass:
				if t.format.NonEscapingQuoteBackslashEscapes() {
					// Fish: only \' and \\ are escapes inside single quotes.
					// Other \X sequences are literal (\ + X).
					peekRune, _, peekErr := t.ReadRune()
					if peekErr == nil {
						token.RawValue += string(peekRune)
						switch peekRune {
						case '\'', '\\':
							token.add(peekRune) // emit just the escaped char
						default:
							// Not an escape — emit both \ and the char
							token.add(nextRune)
							token.add(peekRune)
						}
						// stay in QUOTING_STATE
					} else {
						// EOF after \ — emit \ as literal
						token.add(nextRune)
					}
				} else {
					token.add(nextRune) // literal backslash
				}
			default:
				token.add(nextRune)
			}
		case QUOTING_TRIPLE_STATE: // in triple-quoted non-escaping string ('''...''')
			switch nextRuneType {
			case eofRuneClass: // EOF found when expecting closing triple-quote
				token.removeLastRaw()
				return token, err
			case nonEscapingQuoteRuneClass, escapingQuoteRuneClass:
				if nextRune == t.tripleQuoteRune {
					closed, r1, r2, consumed := t.checkTripleClose()
					if closed {
						token.RawValue += string(r1)
						token.RawValue += string(r2)
						t.rawQuote = false
						t.tripleQuoteRune = 0
						t.state = IN_WORD_STATE
					} else if consumed != 0 {
						token.RawValue += string(consumed)
						token.add(nextRune)
						token.add(consumed)
					} else {
						token.add(nextRune)
					}
				} else {
					token.add(nextRune)
				}
			default:
				token.add(nextRune)
			}
		case QUOTING_TRIPLE_ESCAPING_STATE: // in triple-quoted escaping string ("""...""")
			switch nextRuneType {
			case eofRuneClass: // EOF found when expecting closing triple-quote
				token.removeLastRaw()
				return token, err
			case escapingQuoteRuneClass, nonEscapingQuoteRuneClass:
				if nextRune == t.tripleQuoteRune {
					closed, r1, r2, consumed := t.checkTripleClose()
					if closed {
						token.RawValue += string(r1)
						token.RawValue += string(r2)
						t.rawQuote = false
						t.tripleQuoteRune = 0
						t.state = IN_WORD_STATE
					} else if consumed != 0 {
						token.RawValue += string(consumed)
						token.add(nextRune)
						token.add(consumed)
					} else {
						token.add(nextRune)
					}
				} else {
					token.add(nextRune)
				}
			case escapeRuneClass:
				if t.rawQuote {
					token.add(nextRune) // raw string: backslash is literal
				} else {
					t.state = ESCAPING_QUOTED_STATE
				}
			default:
				token.add(nextRune)
			}
		case COMMENT_STATE: // in a comment
			switch nextRuneType {
			case eofRuneClass:
				return token, err
			case spaceRuneClass:
				if nextRune == '\n' {
					token.removeLastRaw()
					t.state = START_STATE
					return token, err
				} else {
					token.add(nextRune)
				}
			default:
				token.add(nextRune)
			}
		case BLOCK_COMMENT_STATE: // in a block comment (e.g. PowerShell <# ... #>)
			// Match the closer rune-by-rune. nextRune is already in RawValue.
			if rune(t.blockCloser[t.blockCloserIdx]) == nextRune {
				t.blockCloserIdx++
				if t.blockCloserIdx >= len(t.blockCloser) {
					// Full closer matched — return the comment token
					t.blockCloser = ""
					t.blockCloserIdx = 0
					t.state = START_STATE
					return token, err
				}
				// Partial match — keep scanning
			} else {
				// Reset closer match index
				t.blockCloserIdx = 0
				// Check if this rune restarts the closer match
				if rune(t.blockCloser[t.blockCloserIdx]) == nextRune {
					t.blockCloserIdx++
				}
			}
			if nextRuneType == eofRuneClass {
				token.removeLastRaw()
				return token, err
			}
		default:
			return nil, fmt.Errorf("unexpected state: %v", t.state)
		}
	}
}

// scanStopParsing reads tokens in raw mode after a stop-parsing token (e.g.
// PowerShell --%). In this mode, everything is literal until newline or
// a pipeline delimiter (|). Double quotes toggle an "in quotes" state
// where | is not treated as a delimiter, matching PowerShell's
// GetVerbatimCommandArgument behavior. Single & is treated as literal
// (&& is not specially handled due to bufio's single-unread limitation;
// in practice --% mode passes everything literally to native commands).
func (t *tokenizer) scanStopParsing() (*Token, error) {
	token := &Token{}
	token.Type = WORD_TOKEN
	t.state = STOP_PARSING_STATE
	inQuotes := false

	// Skip leading whitespace (matches PowerShell's GetVerbatimCommandArgument
	// which calls SkipWhiteSpace before collecting the raw argument)
	for {
		nextRune, _, err := t.ReadRune()
		if err != nil {
			if err == io.EOF {
				// No content after --% — return empty word at cursor
				token.Span.Start = t.index
				token.Span.End = t.index
				t.state = START_STATE
				return token, nil
			}
			return nil, err
		}
		if nextRune != ' ' && nextRune != '\t' && nextRune != '\r' && nextRune != '\n' {
			t.UnreadRune()
			break
		}
		// If we hit a newline, there's no raw content on this line
		if nextRune == '\r' || nextRune == '\n' {
			t.UnreadRune()
			token.Span.Start = t.index
			token.Span.End = t.index
			t.state = START_STATE
			return token, nil
		}
	}

	for {
		nextRune, _, err := t.ReadRune()
		if err != nil {
			if err == io.EOF {
				if len(token.RawValue) == 0 {
					return nil, err
				}
				token.Span.End = token.Span.Start + len([]rune(token.RawValue))
				token.State = IN_WORD_STATE
				t.state = START_STATE
				return token, nil
			}
			return nil, err
		}

		// Set span start on first rune
		if len(token.RawValue) == 0 {
			token.Span.Start = t.index - 1
		}

		// Check for end conditions before adding to token
		if nextRune == '\r' || nextRune == '\n' {
			t.UnreadRune()
			if len(token.RawValue) == 0 {
				token.Span.Start = t.index
				token.Span.End = t.index
				t.state = START_STATE
				return token, nil
			}
			token.Span.End = token.Span.Start + len([]rune(token.RawValue))
			token.State = IN_WORD_STATE
			t.state = START_STATE
			return token, nil
		}

		token.RawValue += string(nextRune)

		if nextRune == '"' {
			inQuotes = !inQuotes
			token.add(nextRune)
			continue
		}

		if !inQuotes && nextRune == '|' {
			// Pipeline delimiter — return word before it
			token.removeLastRaw()
			t.UnreadRune()
			if len(token.RawValue) == 0 {
				token.Span.Start = t.index
				token.Span.End = t.index
			} else {
				token.Span.End = token.Span.Start + len([]rune(token.RawValue))
			}
			token.State = IN_WORD_STATE
			t.state = START_STATE
			return token, nil
		}

		token.add(nextRune)
	}
}

// Next returns the next token in the stream.
func (t *tokenizer) Next() (*Token, error) {
	// If we're in stop-parsing state, scan in raw mode
	if t.state == STOP_PARSING_STATE {
		token, err := t.scanStopParsing()
		if err == nil {
			// scanStopParsing already sets token.State and t.state
			if token.Span.End == 0 && token.Span.Start >= 0 {
				token.Span.End = token.Span.Start + len([]rune(token.RawValue))
			}
		}
		return token, err
	}

	token, err := t.scanStream()
	if err == nil {
		token.State = t.state // TODO should be done in scanStream
		if token.Span.End == 0 && token.Span.Start >= 0 {
			token.Span.End = token.Span.Start + len([]rune(token.RawValue))
		}
		if token.Type == WORDBREAK_TOKEN {
			token.WordbreakType = t.format.ClassifyOperator(token.RawValue)
		}
		// Keyword operators (fish and/or/not): reclassify WORD_TOKEN as WORDBREAK_TOKEN
		if token.Type == WORD_TOKEN {
			if kwOps := t.format.KeywordOperators(); kwOps != nil {
				if wbType, ok := kwOps[token.RawValue]; ok {
					token.Type = WORDBREAK_TOKEN
					token.WordbreakType = wbType
				}
			}
			// Check for stop-parsing token (e.g. PowerShell --%)
			if sp, ok := t.format.(StopParsingToken); ok {
				if token.Value == sp.StopParsingWord() && token.Value == token.RawValue {
					t.state = STOP_PARSING_STATE
					t.stopParsingDelim = ""
				}
			}
		}
	}
	return token, err
}

// Split partitions a string into tokens using the default (bash) format.
func Split(s string) (TokenSlice, error) {
	return SplitWith(s, BashFormat())
}

// SplitWith partitions a string into tokens using the given format.
func SplitWith(s string, format Format) (TokenSlice, error) {
	l := newLexer(strings.NewReader(s), format)
	tokens := make(TokenSlice, 0)
	for {
		token, err := l.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		tokens = append(tokens, *token)
	}
	if pp, ok := format.(PostProcessor); ok {
		tokens = pp.PostProcess(tokens)
	}
	return tokens, nil
}

// Join concatenates words to create a single string using the default
// (bash) format. It quotes and escapes where appropriate.
func Join(s []string) string {
	return JoinWith(s, BashFormat())
}

// JoinWith concatenates words using the given format's quoting rules.
func JoinWith(s []string, format Format) string {
	formatted := make([]string, 0, len(s))
	for _, arg := range s {
		formatted = append(formatted, format.QuoteWord(arg))
	}
	return strings.Join(formatted, " ")
}
