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

	// EscapeNotInEscapingQuote returns true if the escape character is
	// literal inside the escaping quote (double quotes) rather than acting
	// as an escape. When true, the QUOTING_ESCAPING_STATE handler treats
	// the escape rune as a regular word character instead of entering
	// ESCAPING_QUOTED_STATE. Only cmd needs this: cmd's caret (^) is
	// completely literal inside double quotes — it does not escape the
	// next character when quoted.
	EscapeNotInEscapingQuote() bool

	// EscapingQuoteEscapeChars returns the set of characters that backslash
	// can escape inside the escaping quote (double quotes). If nil, backslash
	// escapes any character. POSIX shells (bash, zsh, tcsh) return the
	// CBSDQUOTE set: \, `, $, ", and newline. Fish returns only
	// `"`, `$`, `\`, and newline (no backtick).
	EscapingQuoteEscapeChars() map[rune]bool

	// TripleQuoteSupport returns true if the shell supports triple-quoted
	// strings ('''...''' and """..."""). When true, the tokenizer peeks
	// ahead two runes on seeing a quote char to detect triple quotes and
	// enters a dedicated triple-quote state. Only xonsh needs this.
	TripleQuoteSupport() bool

	// RawPrefixSupport returns true if the shell supports raw string prefixes
	// (r'...', r"...") that suppress escape processing inside double quotes.
	// When true, the tokenizer checks whether the current word ends with a
	// raw prefix (r/R) before entering QUOTING_ESCAPING_STATE, and if so
	// treats the double quote as non-escaping instead. Only xonsh needs this.
	RawPrefixSupport() bool

	// QuoteWord quotes a single word for safe insertion into a command line.
	// Used by JoinWith. The implementation should use the shell's preferred
	// quoting style: backslash-escaping for POSIX shells, double-quote
	// wrapping for nushell/PowerShell, single-quote wrapping for shells
	// that support it, etc.
	QuoteWord(s string) string
}

// EscapingQuoteUnescaper is an optional interface for formats that need to
// transform escape sequences inside double quotes beyond simple
// backslash-dropping. When implemented, the ESCAPING_QUOTED_STATE handler
// calls EscapingQuoteUnescape for the rune following a backslash. If the
// rune is a recognized escape, the replacement string is used; otherwise
// both the backslash and the rune are kept literally. Formats implementing
// this interface take priority over EscapingQuoteEscapeChars.
type EscapingQuoteUnescaper interface {
	EscapingQuoteUnescape(r rune) (replacement string, handled bool)
}

// PostProcessor is an optional interface for formats that need to reclassify
// tokens after the main tokenization pass. Used by formats that require
// context not available in the flat state machine (e.g. elvish brace/lambda
// context for | disambiguation, nushell stream-redirect operator merging).
type PostProcessor interface {
	PostProcess(tokens TokenSlice) TokenSlice
}

// LineContinuationEscaper is an optional interface for formats where the
// escape character followed by a newline (or carriage return) acts as a
// line continuation — the escape+newline sequence is consumed and discarded,
// NOT added to the word value. This matches PowerShell's backtick line
// continuation behavior.
//
// Without this interface, the ESCAPING_STATE handler always adds the
// post-escape rune to the word value, which is correct for POSIX shells
// where backslash-newline is handled differently.
type LineContinuationEscaper interface {
	// IsLineContinuation returns true if the rune following the escape
	// character should be treated as a line continuation. The parameter
	// is the rune that follows the escape character (e.g. '\n' or '\r').
	IsLineContinuation(r rune) bool
}

// BlockCommenter is an optional interface for formats that support
// multi-line block comments (e.g. PowerShell's <# ... #>). When the
// tokenizer encounters the blockCommentOpener runes at a word boundary,
// it enters a dedicated BLOCK_COMMENT_STATE that scans until the
// blockCommentCloser runes are found, spanning multiple lines.
type BlockCommenter interface {
	BlockCommentOpener() string // e.g. "<#" for PowerShell
	BlockCommentCloser() string // e.g. "#>" for PowerShell
}

// StopParsingToken is an optional interface for formats that support a
// stop-parsing token (e.g. PowerShell's --%). When the tokenizer encounters
// this token as a bare word, it switches to a raw lexing mode for the
// remainder of the line (until newline or pipeline delimiter), where
// all characters except the pipeline delimiters are treated as literal
// word content.
type StopParsingToken interface {
	// StopParsingWord returns the literal token that triggers raw mode.
	// e.g. "--%" for PowerShell.
	StopParsingWord() string
}
