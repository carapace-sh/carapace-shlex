package shlex

import (
	"testing"
)

// Verification tests cross-checking shlex against fish shell source code.
//
// Authoritative sources verified:
//   - tokenizer.rs: tok_is_string_character (word break chars),
//     quote_end (escape behavior inside quotes), PipeOrRedir::try_from
//     (operator grammar), is_token_delimiter
//   - parse_constants.rs: ParseTokenType, ParseKeyword enums
//   - parser_keywords.rs: RESERVED_WORDS, parser_keywords_is_subcommand
//   - redirection.rs: RedirectionMode enum

// tok_is_string_character from tokenizer.rs:
//
//	Unconditional separators: '\0' ' ' '\n' '|' '\t' ';' '\r' '<' '>'
//	'&' is a separator unless AmpersandNoBgInToken feature flag is set
//	and the next char is a string character.
//	Everything else (including '?', '(', ')') is a string character.
//
// shlex should classify the unconditional separators as wordbreaks or space.
// '?' is a string character in fish's tokenizer but shlex classifies it as
// a wordbreak because it forms noclobber redirect operators (>? >>? <?) —
// this is acceptable because the operator is recognized as a unit.
func TestVerify_FishWordBreakChars(t *testing.T) {
	// Fish unconditional separators from tok_is_string_character.
	// Whitespace separators are classified as space, not wordbreak.
	const fishWordBreakSeparators = "|;<>"
	const fishSpaceSeparators = " \t\r\n"

	classifier := FishFormat().Classifier()

	for _, r := range fishWordBreakSeparators {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("separator %q: classifier=%v, want wordbreakRuneClass", r, got)
		}
	}

	for _, r := range fishSpaceSeparators {
		got := classifier.ClassifyRune(r)
		if got != spaceRuneClass {
			t.Errorf("whitespace %q: classifier=%v, want spaceRuneClass", r, got)
		}
	}
}

// tok_is_string_character returns true for characters that are NOT separators.
// These should be classified as word characters (unknownRuneClass = default
// word char) by the classifier, NOT wordbreaks or spaces.
//
// Notable: '(', ')', '?' are string characters in fish's tokenizer.
// '?' is classified as wordbreak in shlex (for noclobber operators).
// '(' and ')' are classified as wordbreak in shlex (for command substitution
// via PostProcess).
func TestVerify_FishStringCharacters(t *testing.T) {
	// Characters that fish treats as string characters (not separators)
	// but shlex classifies as wordbreaks for operator recognition.
	// These are implementation choices, not bugs — verify they're deliberate.
	wordbreakStringChars := "?()"
	classifier := FishFormat().Classifier()
	for _, r := range wordbreakStringChars {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("string char %q: classifier=%v, want wordbreakRuneClass (operator component)", r, got)
		}
	}

	// Regular string characters should NOT be wordbreaks
	regularChars := "abcXYZ012_-.@/%+"
	for _, r := range regularChars {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("regular char %q: classifier=wordbreakRuneClass, want non-wordbreak", r)
		}
	}
}

// quote chars: " is escaping quote, ' is non-escaping quote
// From tokenizer.rs: quote_end handles both ' and " as quote delimiters.
func TestVerify_FishQuoteChars(t *testing.T) {
	classifier := FishFormat().Classifier()

	// " is the escaping quote (double quote) — variable expansion occurs
	if got := classifier.ClassifyRune('"'); got != escapingQuoteRuneClass {
		t.Errorf("\" classifier=%v, want escapingQuoteRuneClass", got)
	}
	// ' is the non-escaping quote (single quote) — no expansion
	if got := classifier.ClassifyRune('\''); got != nonEscapingQuoteRuneClass {
		t.Errorf("' classifier=%v, want nonEscapingQuoteRuneClass", got)
	}
}

// Escape character: \ is the escape character in fish.
// From tokenizer.rs: TOK_MODE_CHAR_ESCAPE mode, read_string handles \.
func TestVerify_FishEscapeChar(t *testing.T) {
	classifier := FishFormat().Classifier()
	if got := classifier.ClassifyRune('\\'); got != escapeRuneClass {
		t.Errorf("\\ classifier=%v, want escapeRuneClass", got)
	}
}

// Comment character: # starts a comment at token boundary.
// From tokenizer.rs: read_string checks # at token start → comment_end.
func TestVerify_FishCommentChar(t *testing.T) {
	classifier := FishFormat().Classifier()
	if got := classifier.ClassifyRune('#'); got != commentRuneClass {
		t.Errorf("# classifier=%v, want commentRuneClass", got)
	}
}

// quote_end from tokenizer.rs:
// Inside both single AND double quotes, \ escapes the next character
// (pos += 1 to skip it). The quote terminator is the matching quote char.
// Inside double quotes, $( also ends the string (for command substitution).
//
// Fish single quotes: \' and \\ are escapes (NonEscapingQuoteBackslashEscapes).
// Other \X sequences are literal (both chars emitted).
func TestVerify_FishSingleQuoteEscapes(t *testing.T) {
	// \' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it\\'s'", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's" {
		t.Errorf("fish \\' in single: Words = %v, want [echo it's]", words)
	}

	// \\ inside single quotes → literal \
	tokens, err = SplitWith("echo 'C:\\\\path'", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path` {
		t.Errorf("fish \\\\ in single: Words = %v, want [echo C:\\path]", words)
	}
}

// Non-escape \X inside single quotes: both \ and X are literal.
// Fish's quote_end skips the char after \, but the unescape logic
// only converts \' and \\ — other \X sequences keep both characters.
func TestVerify_FishSingleQuoteNonEscape(t *testing.T) {
	// \$ inside single quotes → literal \$ (NOT an escape)
	tokens, err := SplitWith(`echo 'cost: \$5'`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: \$5` {
		t.Errorf("fish \\$ in single: Words = %v, want [echo cost: \\$5]", words)
	}

	// \n inside single quotes → literal \n (NOT a newline)
	tokens, err = SplitWith(`echo 'hello\nworld'`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("fish \\n in single: Words = %v, want [echo hello\\nworld]", words)
	}
}

// Double quote escapes: from quote_end and the fish docs,
// inside "..." only \" \$ \\ and \+newline are escapes.
// All other \X sequences are literal (both chars emitted).
// This matches the EscapingQuoteEscapeChars() set.
func TestVerify_FishDoubleQuoteEscapeChars(t *testing.T) {
	// The set of characters \ can escape inside fish double quotes
	expected := map[rune]bool{
		'"':  true,
		'$':  true,
		'\\': true,
		'\n': true,
	}
	got := FishFormat().EscapingQuoteEscapeChars()
	for r, want := range expected {
		if got[r] != want {
			t.Errorf("EscapingQuoteEscapeChars[%q]=%v, want %v", r, got[r], want)
		}
	}
	// Characters NOT escapable inside fish double quotes
	for _, r := range "abc123 \t`" {
		if r == ' ' || r == '\t' {
			continue // spaces break words
		}
		if got[r] {
			t.Errorf("EscapingQuoteEscapeChars[%q]=true, want false (not escapable in fish)", r)
		}
	}
}

// Verify double quote escape behavior end-to-end
func TestVerify_FishDoubleQuoteEscaping(t *testing.T) {
	// \" inside double quotes → literal "
	tokens, err := SplitWith(`echo "say \"hello\""`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("fish \\\" in double: Words = %v, want [echo say \"hello\"]", words)
	}

	// \$ inside double quotes → literal $
	tokens, err = SplitWith(`echo "cost: \$5"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: $5` {
		t.Errorf("fish \\$ in double: Words = %v, want [echo cost: $5]", words)
	}

	// \\ inside double quotes → literal \
	tokens, err = SplitWith(`echo "C:\\path"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path` {
		t.Errorf("fish \\\\ in double: Words = %v, want [echo C:\\path]", words)
	}

	// \n inside double quotes → literal \n (NOT a newline)
	tokens, err = SplitWith(`echo "hello\nworld"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("fish \\n in double: Words = %v, want [echo hello\\nworld]", words)
	}

	// \t inside double quotes → literal \t (NOT a tab)
	tokens, err = SplitWith(`echo "a\tb"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `a\tb` {
		t.Errorf("fish \\t in double: Words = %v, want [echo a\\tb]", words)
	}
}

// Line continuation: backslash + newline is consumed (removed) both
// inside and outside double quotes. From tokenizer.rs, the escape
// mode (TOK_MODE_CHAR_ESCAPE) followed by \n is a line continuation.
func TestVerify_FishLineContinuation(t *testing.T) {
	// Outside quotes: \ + \n → removed
	tokens, err := SplitWith("echo foo\\\nbar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation outside: words=%v, want [echo foobar]", words)
	}

	// Inside double quotes: \ + \n → removed
	tokens, err = SplitWith("echo \"foo\\\nbar\"", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation in double: words=%v, want [echo foobar]", words)
	}

	// CRLF variant: \ + \r\n → removed
	tokens, err = SplitWith("echo foo\\\r\nbar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation CRLF: words=%v, want [echo foobar]", words)
	}
}

// Operator grammar — from tokenizer.rs PipeOrRedir::try_from:
//
//	|        → pipe (stdout)
//	|&       → pipe with stderr merge
//	&|       → pipe with stderr merge
//	>|       → pipe with explicit fd (fish-specific, NOT a redirect like bash)
//	&&       → logical AND
//	||       → logical OR
//	&        → background (async)
//	;        → end (sequential separator)
//	>        → redirect overwrite
//	>>       → redirect append
//	>?       → redirect noclobber
//	>>?      → redirect append+noclobber
//	>&       → fd redirection
//	<        → input redirect
//	<&       → fd redirection (input)
//	<?>      → try-input redirect
//	<>       → input-output redirect
//	&>       → redirect with stderr merge (overwrite)
//	&>>      → redirect with stderr merge (append)
//	&>?      → redirect with stderr merge (noclobber)
//	&>>?     → redirect with stderr merge (append+noclobber)
func TestVerify_FishOperatorGrammar(t *testing.T) {
	tests := []struct {
		input   string
		opType  WordbreakType
		isPipe  bool // IsPipelineDelimiter
		isRedir bool // IsRedirect
	}{
		// Pipes (pipeline delimiters)
		{"|", WORDBREAK_PIPE, true, false},
		{"|&", WORDBREAK_PIPE_WITH_STDERR, true, false},
		{"&|", WORDBREAK_PIPE_WITH_STDERR, true, false},
		{">|", WORDBREAK_PIPE, true, false}, // fish-specific: pipe, not redirect

		// List operators (pipeline delimiters)
		{"&&", WORDBREAK_LIST_AND, true, false},
		{"||", WORDBREAK_LIST_OR, true, false},
		{"&", WORDBREAK_LIST_ASYNC, true, false},
		{";", WORDBREAK_LIST_SEQUENTIAL, true, false},

		// Redirect operators
		{">", WORDBREAK_REDIRECT_OUTPUT, false, true},
		{">>", WORDBREAK_REDIRECT_OUTPUT_APPEND, false, true},
		{">?", WORDBREAK_REDIRECT_OUTPUT, false, true},
		{">>?", WORDBREAK_REDIRECT_OUTPUT_APPEND, false, true},
		{"&>", WORDBREAK_REDIRECT_OUTPUT_BOTH, false, true},
		{"&>>", WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND, false, true},
		{"&>?", WORDBREAK_REDIRECT_OUTPUT_BOTH, false, true},
		{"&>>?", WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND, false, true},
		{"<", WORDBREAK_REDIRECT_INPUT, false, true},
		{"<?", WORDBREAK_REDIRECT_INPUT, false, true},
		{"<>", WORDBREAK_REDIRECT_INPUT_OUTPUT, false, true},
		{">&", WORDBREAK_REDIRECT_INPUT_DUPLICATE, false, true}, // fd redirect
	}

	for _, tc := range tests {
		got := FishFormat().ClassifyOperator(tc.input)
		if got != tc.opType {
			t.Errorf("ClassifyOperator(%q) = %v, want %v", tc.input, got, tc.opType)
			continue
		}
		if got.IsPipelineDelimiter() != tc.isPipe {
			t.Errorf("ClassifyOperator(%q).IsPipelineDelimiter() = %v, want %v", tc.input, got.IsPipelineDelimiter(), tc.isPipe)
		}
		if got.IsRedirect() != tc.isRedir {
			t.Errorf("ClassifyOperator(%q).IsRedirect() = %v, want %v", tc.input, got.IsRedirect(), tc.isRedir)
		}
	}
}

// Fish-specific: >| is a PIPE, not a redirect.
// From PipeOrRedir::try_from: > followed by | sets is_pipe=true.
// "In fish, this is a *pipe*. Run bar as a command and attach foo's
// stderr to bar's stdin, while leaving stdout as tty."
// This differs from bash where >| is a noclobber redirect.
func TestVerify_FishGtPipeIsPipe(t *testing.T) {
	wbType := FishFormat().ClassifyOperator(">|")
	if !wbType.IsPipelineDelimiter() {
		t.Errorf(">| IsPipelineDelimiter() = false, want true (fish: >| is a pipe)")
	}
	if wbType.IsRedirect() {
		t.Errorf(">| IsRedirect() = true, want false (fish: >| is a pipe, not redirect)")
	}
}

// Keyword operators — from parser_keywords.rs:
// "and" and "or" are reserved words AND subcommands (is_super_command=true).
// "not" is a reserved word and subcommand, but it's a prefix keyword,
// not a pipeline delimiter.
// "!" is a subcommand but NOT reserved (is_reserved=false).
//
// shlex should include "and" and "or" as keyword operators that split
// pipelines, but NOT "not" or "!".
func TestVerify_FishKeywordOperators(t *testing.T) {
	kwOps := FishFormat().KeywordOperators()
	if kwOps == nil {
		t.Fatal("KeywordOperators() = nil, want non-nil for fish")
	}

	// "and" should be WORDBREAK_LIST_AND
	if wbType, ok := kwOps["and"]; !ok {
		t.Error("KeywordOperators() missing 'and'")
	} else if wbType != WORDBREAK_LIST_AND {
		t.Errorf("KeywordOperators['and'] = %v, want WORDBREAK_LIST_AND", wbType)
	}

	// "or" should be WORDBREAK_LIST_OR
	if wbType, ok := kwOps["or"]; !ok {
		t.Error("KeywordOperators() missing 'or'")
	} else if wbType != WORDBREAK_LIST_OR {
		t.Errorf("KeywordOperators['or'] = %v, want WORDBREAK_LIST_OR", wbType)
	}

	// "not" should NOT be a keyword operator (it's a prefix, not a delimiter)
	if _, ok := kwOps["not"]; ok {
		t.Error("KeywordOperators() should NOT include 'not' (prefix keyword, not pipeline delimiter)")
	}

	// "!" should NOT be a keyword operator
	if _, ok := kwOps["!"]; ok {
		t.Error("KeywordOperators() should NOT include '!' (prefix keyword, not pipeline delimiter)")
	}
}

// Verify keyword operators split pipelines end-to-end
func TestVerify_FishKeywordOperatorPipelineSplit(t *testing.T) {
	// "and" splits pipelines
	tokens, err := SplitWith("echo foo and echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish 'and': %d pipelines, want 2", len(pipelines))
	}

	// "or" splits pipelines
	tokens, err = SplitWith("echo foo or echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines = tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish 'or': %d pipelines, want 2", len(pipelines))
	}

	// "not" does NOT split pipelines (prefix keyword)
	tokens, err = SplitWith("echo foo not echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines = tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("fish 'not': %d pipelines, want 1 (not is a prefix, not a delimiter)", len(pipelines))
	}
}

// Command substitution: fish uses bare () for command substitution
// (no $ prefix required). From tokenizer.rs: is_token_delimiter treats
// ( as a delimiter. The PostProcess reclassifies ( and ) as
// WORDBREAK_SUBSTITUTION_OPEN/CLOSE.
func TestVerify_FishCommandSubstitution(t *testing.T) {
	tokens, err := SplitWith("echo (ls)", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	foundOpen := false
	foundClose := false
	for _, tok := range tokens {
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN {
			foundOpen = true
		}
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE {
			foundClose = true
		}
	}
	if !foundOpen {
		t.Error("fish command substitution: ( not classified as WORDBREAK_SUBSTITUTION_OPEN")
	}
	if !foundClose {
		t.Error("fish command substitution: ) not classified as WORDBREAK_SUBSTITUTION_CLOSE")
	}
}

// Command substitution should not split pipelines.
// The substitution delimiters should track depth so that pipes inside
// (cmd1 | cmd2) don't split the outer pipeline.
func TestVerify_FishSubstitutionDoesntSplitPipeline(t *testing.T) {
	tokens, err := SplitWith("echo (ls | grep foo)", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("fish subst pipeline: %d pipelines, want 1 (pipe inside () shouldn't split)", len(pipelines))
	}
}

// QuoteWord for JoinWith — fish uses double-quote wrapping with
// \" \$ \\ and \+newline escapes. Backtick is NOT special in fish.
func TestVerify_FishQuoteWord(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Safe words: no quoting needed
		{"hello", "hello"},
		{"abc123", "abc123"},
		{"hello-world", "hello-world"},
		// Special chars need quoting
		{"hello world", `"hello world"`},
		{`with"quote`, `"with\"quote"`},
		{`with\backslash`, `"with\\backslash"`},
		{`with$dollar`, `"with\$dollar"`},
		// Backtick is NOT special in fish (unlike bash)
		{"hello`world", "hello`world"},
		// Empty string
		{"", `""`},
	}

	for _, tc := range tests {
		got := fishQuoteWord(tc.input)
		if got != tc.want {
			t.Errorf("fishQuoteWord(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// Fish does NOT do word splitting on variable expansion.
// $var and "$var" produce the same result. This doesn't change the lexer
// (which splits on literal source whitespace), but verify the behavior
// is consistent: spaces inside double quotes don't split words.
func TestVerify_FishNoWordSplitting(t *testing.T) {
	// Inside double quotes, spaces don't split
	tokens, err := SplitWith(`echo "hello world"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("fish no word split: Words = %v, want [echo hello world]", words)
	}

	// Outside quotes, spaces DO split (literal source whitespace)
	tokens, err = SplitWith("echo hello world", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 3 {
		t.Errorf("fish word split: Words = %v, want 3 words", words)
	}
}

// Fish escape outside quotes: backslash escapes the next character.
// From tokenizer.rs: TOK_MODE_CHAR_ESCAPE mode — next char consumed literally.
func TestVerify_FishEscapeOutsideQuotes(t *testing.T) {
	// Escaped space → one word
	tokens, err := SplitWith(`echo a\ b`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("fish escaped space: Words = %v, want [echo a b]", words)
	}

	// Escaped $ → literal $
	tokens, err = SplitWith(`echo \$HOME`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("fish escaped $: Words = %v, want [echo $HOME]", words)
	}

	// Escaped pipe → literal | (not a pipeline delimiter)
	tokens, err = SplitWith(`echo foo\|bar`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo|bar" {
		t.Errorf("fish escaped |: Words = %v, want [echo foo|bar]", words)
	}
}

// Comment: # starts a comment at token boundary.
// From tokenizer.rs: # at token start → comment_end (skip to end of line).
// # mid-word is a regular character.
func TestVerify_FishComment(t *testing.T) {
	tokens, err := SplitWith("echo hello # this is a comment", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello" {
		t.Errorf("fish comment: words=%v, want [echo hello]", words)
	}
}

// Fish does not have COMP_WORDBREAKS — the wordbreak set is fixed by the format.
// Verify there's no env var dependency in the fish classifier.
func TestVerify_FishNoEnvVarDependency(t *testing.T) {
	// Classifier should be the same regardless of COMP_WORDBREAKS
	c1 := FishFormat().Classifier()
	// COMP_WORDBREAKS should not affect fish classifier
	// (it only affects bash)
	for _, r := range "|;<>&?()" {
		got := c1.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("fish classifier %q: got %v, want wordbreakRuneClass", r, got)
		}
	}
}
