package shlex

import (
	"testing"
)

// Verification tests cross-checking shlex against PowerShell source code.
//
// Authoritative sources verified:
//   - CharTraits.cs: ForceStartNewToken, IsSingleQuote, IsDoubleQuote,
//     IsWhitespace, ForceStartNewTokenAfterNumber
//   - tokenizer.cs: Backtick() (escape sequences), ScanStringLiteral
//     (single quote behavior), ScanStringExpandable (double quote
//     escape behavior), ScanBlockComment, ScanLineContinuation,
//     GetVerbatimCommandArgument (stop-parsing), operator dispatch
//     (|, ||, &, &&, ;, >, >>, <, redirects), ScanSubExpression ($(),
//     ScanGenericToken (backtick in barewords)
//   - Parser.cs: VERBATIM_ARGUMENT ("--%"), verbatim argument handling

// ForceStartNewToken from CharTraits.cs:
//
//	Characters that force starting a new token:
//	\0  \t  \n  \v  \f  \r  (space)  &  (  )  ,  ;  {  |  }
//
// shlex classifies the pipeline/redirect operators as wordbreaks and
// whitespace as space. Characters like , { } are ForceStartNewToken in
// PowerShell but are NOT wordbreaks in shlex — they are word characters.
// This is a deliberate shlex choice: shlex tracks only pipeline delimiters
// and redirect operators for completion, not all token boundaries.
func TestVerify_PowershellForceStartNewToken(t *testing.T) {
	// Operators that shlex classifies as wordbreaks (subset of ForceStartNewToken)
	const psWordBreakOps = "|;&><()"
	const psSpaceChars = " \t\r\n"

	classifier := PowershellFormat().Classifier()

	for _, r := range psWordBreakOps {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("ForceStartNewToken %q: classifier=%v, want wordbreakRuneClass", r, got)
		}
	}

	for _, r := range psSpaceChars {
		got := classifier.ClassifyRune(r)
		if got != spaceRuneClass {
			t.Errorf("whitespace %q: classifier=%v, want spaceRuneClass", r, got)
		}
	}
}

// ForceStartNewToken chars NOT classified as wordbreaks by shlex:
// , { } are token separators in PowerShell but shlex treats them as word
// characters. This is intentional — they are not pipeline delimiters or
// redirect operators, so they don't need to break words for completion.
func TestVerify_PowershellNonOperatorTokenSeparators(t *testing.T) {
	classifier := PowershellFormat().Classifier()
	// , { } are ForceStartNewToken in PowerShell but not wordbreaks in shlex
	nonWordbreakSeps := ",{}"
	for _, r := range nonWordbreakSeps {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("char %q: should not be wordbreak (not a pipeline/redirect operator)", r)
		}
	}
}

// Regular characters should not be wordbreaks.
func TestVerify_PowershellStringCharacters(t *testing.T) {
	classifier := PowershellFormat().Classifier()
	regularChars := "abcXYZ012_-.@/%+"
	for _, r := range regularChars {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("regular char %q: classifier=wordbreakRuneClass, want non-wordbreak", r)
		}
	}
}

// Quote chars from CharTraits.cs:
// IsSingleQuote: ' (plus Unicode smart quotes, not tracked by shlex)
// IsDoubleQuote: " (plus Unicode smart quotes, not tracked by shlex)
func TestVerify_PowershellQuoteChars(t *testing.T) {
	classifier := PowershellFormat().Classifier()

	// " is the escaping quote (double quote) — variable expansion, escapes
	if got := classifier.ClassifyRune('"'); got != escapingQuoteRuneClass {
		t.Errorf("\" classifier=%v, want escapingQuoteRuneClass", got)
	}
	// ' is the non-escaping quote (single quote) — literal, no expansion
	if got := classifier.ClassifyRune('\''); got != nonEscapingQuoteRuneClass {
		t.Errorf("' classifier=%v, want nonEscapingQuoteRuneClass", got)
	}
}

// Escape character: backtick (`) is the PowerShell escape character.
// From tokenizer.cs: Backtick() method handles escape sequences.
// From CharTraits.cs: backtick is not ForceStartNewToken (it's part of tokens).
func TestVerify_PowershellEscapeChar(t *testing.T) {
	classifier := PowershellFormat().Classifier()
	if got := classifier.ClassifyRune('`'); got != escapeRuneClass {
		t.Errorf("` classifier=%v, want escapeRuneClass", got)
	}
}

// Backslash is NOT the escape character in PowerShell (backtick is).
// From CharTraits.cs: \ is a regular character (not ForceStartNewToken).
// shlex should classify \ as a regular word character, not escape.
func TestVerify_PowershellBackslashNotEscape(t *testing.T) {
	classifier := PowershellFormat().Classifier()
	if got := classifier.ClassifyRune('\\'); got == escapeRuneClass {
		t.Errorf("\\ classifier=escapeRuneClass, want non-escape (backtick is the escape in PowerShell)")
	}
	if got := classifier.ClassifyRune('\\'); got == wordbreakRuneClass {
		t.Errorf("\\ classifier=wordbreakRuneClass, want non-wordbreak")
	}
}

// Comment character: # starts a line comment.
// From tokenizer.cs: # at token start begins a comment.
func TestVerify_PowershellCommentChar(t *testing.T) {
	classifier := PowershellFormat().Classifier()
	if got := classifier.ClassifyRune('#'); got != commentRuneClass {
		t.Errorf("# classifier=%v, want commentRuneClass", got)
	}
}

// Operator grammar — from tokenizer.cs dispatch loop:
//
//	|   → TokenKind.Pipe        (pipe)
//	||  → TokenKind.OrOr        (logical OR)
//	&&  → TokenKind.AndAnd      (logical AND)
//	;   → TokenKind.Semi        (statement separator)
//	&   → TokenKind.Ampersand   (call operator, NOT background)
//	>   → FileRedirection        (output redirect, stream 1)
//	>>  → FileRedirection        (append output redirect)
//	<   → InputRedirection      (input redirect)
func TestVerify_PowershellOperatorGrammar(t *testing.T) {
	tests := []struct {
		input   string
		opType  WordbreakType
		isPipe  bool // IsPipelineDelimiter
		isRedir bool // IsRedirect
	}{
		// Pipeline delimiters
		{"|", WORDBREAK_PIPE, true, false},
		{"||", WORDBREAK_LIST_OR, true, false},
		{"&&", WORDBREAK_LIST_AND, true, false},
		{";", WORDBREAK_LIST_SEQUENTIAL, true, false},

		// Call operator — NOT a pipeline delimiter in PowerShell
		// (& is the call operator, not background like in bash)
		{"&", WORDBREAK_UNKNOWN, false, false},

		// Redirect operators
		// Note: shlex maps both > and >> to WORDBREAK_REDIRECT_OUTPUT.
		// PowerShell's tokenizer distinguishes append=true/false, but
		// shlex treats them the same — both are redirects.
		{">", WORDBREAK_REDIRECT_OUTPUT, false, true},
		{">>", WORDBREAK_REDIRECT_OUTPUT, false, true},
		{"<", WORDBREAK_REDIRECT_INPUT, false, true},
	}

	for _, tc := range tests {
		got := PowershellFormat().ClassifyOperator(tc.input)
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

// Pipeline delimiters from tokenizer.cs: |, ||, &&, ;
// These should all return IsPipelineDelimiter() = true.
// Note: & (call operator) is NOT a pipeline delimiter in PowerShell
// (unlike bash where & is background/async).
func TestVerify_PowershellPipelineDelimiters(t *testing.T) {
	pipelineOps := []string{"|", "||", "&&", ";"}
	for _, op := range pipelineOps {
		wbType := PowershellFormat().ClassifyOperator(op)
		if !wbType.IsPipelineDelimiter() {
			t.Errorf("ClassifyOperator(%q).IsPipelineDelimiter() = false, want true", op)
		}
	}
}

// Redirect operators from tokenizer.cs: >, >>, <
// These should all return IsRedirect() = true.
func TestVerify_PowershellRedirectOperators(t *testing.T) {
	redirectOps := []string{">", ">>", "<"}
	for _, op := range redirectOps {
		wbType := PowershellFormat().ClassifyOperator(op)
		if !wbType.IsRedirect() {
			t.Errorf("ClassifyOperator(%q).IsRedirect() = false, want true", op)
		}
	}
}

// Call operator & is NOT a pipeline delimiter.
// From tokenizer.cs: & produces TokenKind.Ampersand (call operator).
// In PowerShell, & invokes a command, script, or function — it does NOT
// run in the background. This differs from bash where & is async/background.
func TestVerify_PowershellCallOperatorNotPipelineDelimiter(t *testing.T) {
	wbType := PowershellFormat().ClassifyOperator("&")
	if wbType.IsPipelineDelimiter() {
		t.Error("& IsPipelineDelimiter() = true, want false (call operator, not background)")
	}
	if wbType.IsRedirect() {
		t.Error("& IsRedirect() = true, want false (call operator, not redirect)")
	}
}

// Single quote behavior — from tokenizer.cs ScanStringLiteral:
// Inside single quotes, the ONLY escape is ” (doubled single quote) → literal '.
// No backslash escapes, no backtick escapes — everything is literal.
// NonEscapingQuoteEscapes() = true enables ” handling.
// NonEscapingQuoteBackslashEscapes() = false (no \ escapes in single quotes).
func TestVerify_PowershellSingleQuoteNoEscape(t *testing.T) {
	// \ is literal inside single quotes
	tokens, err := SplitWith(`echo 'hello\nworld'`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("single quote literal \\: Words = %v, want [echo hello\\nworld]", words)
	}

	// $ is literal inside single quotes
	tokens, err = SplitWith(`echo '$HOME'`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("single quote $ literal: Words = %v, want [echo $HOME]", words)
	}

	// backtick is literal inside single quotes
	tokens, err = SplitWith("echo '`hello'", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "`hello" {
		t.Errorf("single quote backtick literal: Words = %v, want [echo `hello]", words)
	}
}

// Single quote doubled quote — from tokenizer.cs ScanStringLiteral:
// ” inside single quotes → literal ' (the first quote "escapes" the second).
func TestVerify_PowershellSingleQuoteDoubled(t *testing.T) {
	tokens, err := SplitWith("echo 'don''t'", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "don't" {
		t.Errorf("single quote '' : Words = %v, want [echo don't]", words)
	}
}

// Double quote doubled quote — from tokenizer.cs ScanStringExpandable:
// "" inside double quotes → literal " (the first quote "escapes" the second).
func TestVerify_PowershellDoubleQuoteDoubled(t *testing.T) {
	tokens, err := SplitWith(`echo "say ""hello"""`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `say "hello"` {
		t.Errorf(`double quote "": Value = %q, want %q`, last.Value, `say "hello"`)
	}
}

// EscapingQuoteEscapeChars — from tokenizer.cs Backtick() method:
// Inside double quotes, backtick can escape ANY character (default case
// returns the character as-is). The Backtick() method handles special
// sequences like `n, `t, `r, but the default case passes any char through.
// shlex returns nil for EscapingQuoteEscapeChars, meaning backtick escapes
// any character inside double quotes — matching PowerShell's behavior.
func TestVerify_PowershellEscapingQuoteEscapeChars(t *testing.T) {
	got := PowershellFormat().EscapingQuoteEscapeChars()
	// nil means backtick escapes any character inside double quotes
	if got != nil {
		t.Errorf("EscapingQuoteEscapeChars() = %v, want nil (backtick escapes any char)", got)
	}
}

// Backtick escape inside double quotes — from tokenizer.cs ScanStringExpandable:
// Backtick followed by any character inside double quotes: the backtick is
// consumed and the character is kept. Special sequences like `n, `t produce
// control characters in PowerShell, but shlex keeps the literal character
// (it drops the backtick and keeps the next char as-is).
func TestVerify_PowershellBacktickInDoubleQuotes(t *testing.T) {
	// `" inside double quotes → literal "
	tokens, err := SplitWith("echo \"say `\"hello`\"\"", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("backtick \" in double: Words = %v, want [echo say \"hello\"]", words)
	}

	// `$ inside double quotes → literal $
	tokens, err = SplitWith("echo \"cost: `$5\"", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "cost: $5" {
		t.Errorf("backtick $ in double: Words = %v, want [echo cost: $5]", words)
	}
}

// Backtick escape outside quotes — from tokenizer.cs ScanGenericToken:
// Backtick in a bareword escapes the next character (backtick consumed,
// next char kept). This matches the EscapeNotBareword() = true behavior.
func TestVerify_PowershellBacktickEscapeOutside(t *testing.T) {
	// `$ outside quotes → literal $ (part of word)
	tokens, err := SplitWith("echo `$HOME", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("backtick $ outside: Words = %v, want [echo $HOME]", words)
	}

	// `| outside quotes → literal | (not a pipeline delimiter)
	tokens, err = SplitWith("echo foo`|bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo|bar" {
		t.Errorf("backtick | outside: Words = %v, want [echo foo|bar]", words)
	}
}

// EscapeNotBareword — from tokenizer.cs ScanGenericToken:
// Backtick IS an escape character in barewords (it escapes the next char).
// EscapeNotBareword() = true means the escape char acts as escape in barewords.
func TestVerify_PowershellEscapeNotBareword(t *testing.T) {
	if !PowershellFormat().EscapeNotBareword() {
		t.Error("EscapeNotBareword() = false, want true (backtick is escape in barewords)")
	}
}

// NonEscapingQuoteEscapes — from tokenizer.cs ScanStringLiteral:
// Single quotes support ” (doubled quote) as an escape for literal '.
func TestVerify_PowershellNonEscapingQuoteEscapes(t *testing.T) {
	if !PowershellFormat().NonEscapingQuoteEscapes() {
		t.Error("NonEscapingQuoteEscapes() = false, want true ('' → literal ')")
	}
}

// NonEscapingQuoteBackslashEscapes — PowerShell single quotes do NOT
// support backslash escapes. Only ” is recognized.
func TestVerify_PowershellNonEscapingQuoteBackslashEscapes(t *testing.T) {
	if PowershellFormat().NonEscapingQuoteBackslashEscapes() {
		t.Error("NonEscapingQuoteBackslashEscapes() = true, want false (no \\ escapes in single quotes)")
	}
}

// EscapeNotInEscapingQuote — from tokenizer.cs ScanStringExpandable:
// Backtick IS an escape character inside double quotes (it triggers
// Backtick() processing). EscapeNotInEscapingQuote() = false means
// the escape char acts as escape inside double quotes.
func TestVerify_PowershellEscapeNotInEscapingQuote(t *testing.T) {
	if PowershellFormat().EscapeNotInEscapingQuote() {
		t.Error("EscapeNotInEscapingQuote() = true, want false (backtick is escape inside double quotes)")
	}
}

// Line continuation — from tokenizer.cs ScanLineContinuation:
// Backtick followed by \n or \r is a line continuation: the sequence is
// consumed and the word continues on the next line.
func TestVerify_PowershellLineContinuation(t *testing.T) {
	// Outside quotes: ` + \n → consumed (word continues)
	tokens, err := SplitWith("echo foo`\nbar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation outside: words=%v, want [echo foobar]", words)
	}

	// CRLF variant: ` + \r\n → consumed
	tokens, err = SplitWith("echo foo`\r\nbar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation CRLF: words=%v, want [echo foobar]", words)
	}

	// At start of word: ` + \n → consumed, word starts on next line
	tokens, err = SplitWith("echo `\nbar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "bar" {
		t.Errorf("line continuation start: words=%v, want [echo bar]", words)
	}
}

// Block comments — from tokenizer.cs ScanBlockComment:
// <# starts a block comment, #> ends it. Can span multiple lines.
func TestVerify_PowershellBlockComment(t *testing.T) {
	tokens, err := SplitWith("echo <# multi\nline\ncomment #> foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("block comment: Words = %v, want [echo foo]", words)
	}
}

// Block comment single line.
func TestVerify_PowershellBlockCommentSingleLine(t *testing.T) {
	tokens, err := SplitWith("echo <# inline comment #> foo", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("block comment inline: Words = %v, want [echo foo]", words)
	}
}

// Block comment delimiters — from tokenizer.cs:
// BlockCommentOpener: <#    BlockCommentCloser: #>
func TestVerify_PowershellBlockCommentDelimiters(t *testing.T) {
	bc, ok := PowershellFormat().(BlockCommenter)
	if !ok {
		t.Fatal("PowershellFormat() does not implement BlockCommenter")
	}
	if bc.BlockCommentOpener() != "<#" {
		t.Errorf("BlockCommentOpener() = %q, want <#", bc.BlockCommentOpener())
	}
	if bc.BlockCommentCloser() != "#>" {
		t.Errorf("BlockCommentCloser() = %q, want #>", bc.BlockCommentCloser())
	}
}

// Stop-parsing token — from Parser.cs: VERBATIM_ARGUMENT = "--%"
// From tokenizer.cs GetVerbatimCommandArgument: after --%, content is
// read literally until newline or | (or &&).
func TestVerify_PowershellStopParsingToken(t *testing.T) {
	sp, ok := PowershellFormat().(StopParsingToken)
	if !ok {
		t.Fatal("PowershellFormat() does not implement StopParsingToken")
	}
	if sp.StopParsingWord() != "--%" {
		t.Errorf("StopParsingWord() = %q, want --%%", sp.StopParsingWord())
	}
}

// Stop-parsing raw content — from tokenizer.cs GetVerbatimCommandArgument:
// After --%, content is read until newline or |. Characters like ( ) are
// literal, not token separators. Double quotes toggle quoting state but
// | and && still act as delimiters.
func TestVerify_PowershellStopParsingRawContent(t *testing.T) {
	tokens, err := SplitWith("icacls X: --% /grant Dom\\HVAdmin:(CI)(OI)F", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 4 {
		t.Errorf("stop-parsing raw: Words = %v, want 4 words", words)
	}
	if words[2] != "--%" {
		t.Errorf("stop-parsing raw: third word = %q, want --%%", words[2])
	}
	rawContent := words[3]
	if rawContent != "/grant Dom\\HVAdmin:(CI)(OI)F" {
		t.Errorf("stop-parsing raw: content = %q, want /grant Dom\\HVAdmin:(CI)(OI)F", rawContent)
	}
}

// Stop-parsing pipe delimiter — from tokenizer.cs GetVerbatimCommandArgument:
// After --%, | is still a pipeline delimiter (raw mode stops at |).
func TestVerify_PowershellStopParsingPipeDelim(t *testing.T) {
	tokens, err := SplitWith("echo --% foo | Select-String bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("stop-parsing pipe: %d pipelines, want 2", len(pipelines))
	}
}

// Stream redirect operators — from tokenizer.cs dispatch (case '0'-'9' and '>'):
//
//	2>   → FileRedirection(stream 2, append: false)
//	2>>  → FileRedirection(stream 2, append: true)
//	2>&1 → MergingRedirection(from 2, to 1)
//	1>&2 → MergingRedirection(from 1, to 2)
//	*>   → FileRedirection(all streams, append: false)
//	*>>  → FileRedirection(all streams, append: true)
//
// shlex merges these in PostProcess into single WORDBREAK_TOKENs.
func TestVerify_PowershellStreamRedirect(t *testing.T) {
	tests := []struct {
		input    string
		rawValue string
		isRedir  bool
	}{
		{"echo foo 2> file", "2>", true},
		{"echo foo 2>> file", "2>>", true},
		{"echo foo 2>&1", "2>&1", true},
		{"echo foo 1>&2", "1>&2", true},
		{"echo foo *> file", "*>", true},
		{"echo foo *>> file", "*>>", true},
	}

	for _, tc := range tests {
		tokens, err := SplitWith(tc.input, PowershellFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", tc.input, err)
		}
		found := false
		for _, tok := range tokens {
			if tok.Type == WORDBREAK_TOKEN && tok.RawValue == tc.rawValue {
				found = true
				if tc.isRedir && !tok.WordbreakType.IsRedirect() {
					t.Errorf("stream redirect %q: IsRedirect() = false, want true", tc.rawValue)
				}
				break
			}
		}
		if !found {
			t.Errorf("stream redirect %q: no merged token found", tc.rawValue)
		}
	}
}

// Subexpression operators — from tokenizer.cs dispatch:
// $( → TokenKind.DollarParen (subexpression open)
// ) → closing paren (subexpression close)
// shlex merges $ + ( into WORDBREAK_SUBSTITUTION_OPEN in PostProcess
// and reclassifies standalone ( and ) as WORDBREAK_SUBSTITUTION_OPEN/CLOSE.
func TestVerify_PowershellSubexpression(t *testing.T) {
	tokens, err := SplitWith("echo $(ls)", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	foundOpen := false
	foundClose := false
	for _, tok := range tokens {
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN {
			if tok.Value == "$(" {
				foundOpen = true
			}
		}
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE {
			foundClose = true
		}
	}
	if !foundOpen {
		t.Error("subexpression: $( not merged into WORDBREAK_SUBSTITUTION_OPEN")
	}
	if !foundClose {
		t.Error("subexpression: ) not classified as WORDBREAK_SUBSTITUTION_CLOSE")
	}
}

// QuoteWord — from PowerShell quoting rules:
// PowerShell prefers single-quote wrapping with ” for literal '.
// Safe words (no special chars) are returned as-is.
func TestVerify_PowershellQuoteWord(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Safe words: no quoting needed
		{"hello", "hello"},
		{"abc123", "abc123"},
		{"C:\\path", "C:\\path"},
		// Special chars need quoting
		{"hello world", "'hello world'"},
		{"with'quote", "'with''quote'"},
		{"with$dollar", "'with$dollar'"},
		{"with\"quote", "'with\"quote'"},
		{"with`backtick", "'with`backtick'"},
		// Empty string
		{"", "''"},
	}

	for _, tc := range tests {
		got := powershellQuoteWord(tc.input)
		if got != tc.want {
			t.Errorf("powershellQuoteWord(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// No keyword operators — PowerShell has no bare-word operators like
// fish's "and"/"or". Pipeline delimiters are all symbolic operators.
func TestVerify_PowershellNoKeywordOperators(t *testing.T) {
	kwOps := PowershellFormat().KeywordOperators()
	if kwOps != nil {
		t.Errorf("KeywordOperators() = %v, want nil (PowerShell has no keyword operators)", kwOps)
	}
}

// No triple quote support — PowerShell does not have triple-quoted strings.
func TestVerify_PowershellNoTripleQuote(t *testing.T) {
	if PowershellFormat().TripleQuoteSupport() {
		t.Error("TripleQuoteSupport() = true, want false (PowerShell has no triple quotes)")
	}
}

// No raw prefix support — PowerShell does not have r'...' raw strings.
func TestVerify_PowershellNoRawPrefix(t *testing.T) {
	if PowershellFormat().RawPrefixSupport() {
		t.Error("RawPrefixSupport() = true, want false (PowerShell has no raw string prefixes)")
	}
}

// Command separation — from tokenizer.cs: |, ;, &&, || split pipelines.
func TestVerify_PowershellCommandSeparators(t *testing.T) {
	separators := []string{"|", ";", "&&", "||"}
	for _, sep := range separators {
		input := "echo foo" + sep + " echo bar"
		tokens, err := SplitWith(input, PowershellFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", input, err)
		}
		pipelines := tokens.Pipelines()
		if len(pipelines) < 2 {
			t.Errorf("command separator %q: got %d pipelines, want >= 2", sep, len(pipelines))
		}
	}
}

// Open quote state detection — from tokenizer.cs:
// An unclosed single quote should leave the tokenizer in QUOTING_STATE.
// An unclosed double quote should leave it in QUOTING_ESCAPING_STATE.
func TestVerify_PowershellOpenQuoteState(t *testing.T) {
	// Open single quote
	tokens, err := SplitWith("echo 'hel", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("open single: State = %v, want QUOTING_STATE", last.State)
	}

	// Open double quote
	tokens, err = SplitWith(`echo "hel`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last = tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}

// No COMP_WORDBREAKS dependency — PowerShell does not use COMP_WORDBREAKS.
// The classifier should be static and not depend on environment variables.
func TestVerify_PowershellNoEnvVarDependency(t *testing.T) {
	classifier := PowershellFormat().Classifier()
	for _, r := range "|;&><()" {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("powershell classifier %q: got %v, want wordbreakRuneClass", r, got)
		}
	}
}

// Backslash is a literal word character in PowerShell.
// From CharTraits.cs: \ is not ForceStartNewToken, not a quote, not an escape.
// shlex should treat \ as a regular word character.
func TestVerify_PowershellBackslashLiteral(t *testing.T) {
	tokens, err := SplitWith(`echo C:\path\to\file`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path\to\file` {
		t.Errorf("backslash literal: Words = %v, want [echo C:\\path\\to\\file]", words)
	}
}
