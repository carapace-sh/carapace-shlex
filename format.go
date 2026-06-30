package shlex

// Span represents a rune-offset range in the input string.
type Span struct {
	Start int // rune offset of the first character
	End   int // rune offset after the last character
}

// Format describes a shell's lexical rules: which runes are quotes,
// escapes, comments, and word breaks, and how operators are classified.
type Format interface {
	// Classifier returns a rune classifier mapping runes to runeTokenClass.
	// Called once per tokenizer; should be freshly built (may read env vars).
	Classifier() tokenClassifier

	// ClassifyOperator maps a wordbreak token's RawValue to a WordbreakType.
	// Called for WORDBREAK_TOKENs to determine redirect vs pipeline vs other.
	ClassifyOperator(raw string) WordbreakType

	// KeywordOperators returns bare-word operators (e.g. fish "and"/"or")
	// that should be treated as WORDBREAK_TOKEN despite being word characters.
	// Returns nil for shells without keyword operators.
	KeywordOperators() map[string]WordbreakType

	// NonEscapingQuoteEscapes returns true if the non-escaping quote (single
	// quote) supports limited escapes. When true, the QUOTING_STATE handler
	// peeks at the next rune on seeing the quote char or escape char:
	//   - '' (doubled quote) → one literal quote, stay in state
	//   - \' or \\ → escaped char, stay in state
	// Supported by: fish, elvish, zsh (RC_QUOTES), PowerShell.
	NonEscapingQuoteEscapes() bool

	// NonEscapingQuoteBackslashEscapes returns true if backslash (\) is an
	// escape character inside the non-escaping quote (single quotes).
	// When true, \' and \\ inside single quotes produce the escaped char
	// and stay in QUOTING_STATE. Only fish needs this.
	// Requires NonEscapingQuoteEscapes() to also be true.
	NonEscapingQuoteBackslashEscapes() bool

	// EscapeNotBareword returns false if the escape character (backslash)
	// is a literal bareword character outside quotes rather than an escape.
	// When false, the state machine does NOT enter ESCAPING_STATE from
	// IN_WORD_STATE — the escape char is treated as a regular word char.
	// Only elvish needs this (\ is a bareword char in elvish).
	EscapeNotBareword() bool

	// QuoteWord quotes a single word for safe insertion into a command line.
	// Used by JoinWith. The implementation should use the shell's preferred
	// quoting style: backslash-escaping for POSIX shells, double-quote
	// wrapping for nushell/PowerShell, single-quote wrapping for shells
	// that support it, etc.
	QuoteWord(s string) string
}

// PostProcessor is an optional interface for formats that need to reclassify
// tokens after the main tokenization pass. Used by formats that require
// context not available in the flat state machine (e.g. elvish brace/lambda
// context for | disambiguation).
type PostProcessor interface {
	PostProcess(tokens TokenSlice) TokenSlice
}
