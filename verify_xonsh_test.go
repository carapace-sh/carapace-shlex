package shlex

import (
	"testing"
)

// Verification tests cross-checking shlex against xonsh source code.
//
// Authoritative sources verified:
//   - tokenize.py: Special, Bracket, Whitespace, Operator, StringPrefix,
//     endpats, Single/Double/Single3/Double3 regexes, Triple, ContStr,
//     Ignore (line continuation), Funny
//   - lexer.py: _op_map, NEED_WHITESPACE, handle_name, handle_double_amps,
//     handle_double_pipe, handle_redirect, handle_error_linecont,
//     _redir_names, _redir_map, single_quoted, triple_quoted

// XONSH_SPECIAL = ":;.,@" — from tokenize.py: Special = group(r"\r?\n", r"\.\.\.", r"[:;.,@]")
// In subprocess mode, these are punctuation that separates tokens.
// shlex classifies ; and ( as wordbreaks; : . , @ are word characters
// in the shlex classifier (they don't form shell operators in xonsh
// subprocess mode the way | < > & do).
//
// Whitespace = r"[ \f\t]*" — space, form-feed, tab are whitespace.
// shlex uses spaceRunes which includes \r\n as spaces too.
func TestVerify_XonshWordBreakChars(t *testing.T) {
	// Operators that should be wordbreaks: | < > & ; ( )
	const xonshWordBreakOps = "|<>&;()"
	const xonshSpaceChars = " \t\r\n"

	classifier := XonshFormat().Classifier()

	for _, r := range xonshWordBreakOps {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("operator %q: classifier=%v, want wordbreakRuneClass", r, got)
		}
	}

	for _, r := range xonshSpaceChars {
		got := classifier.ClassifyRune(r)
		if got != spaceRuneClass {
			t.Errorf("whitespace %q: classifier=%v, want spaceRuneClass", r, got)
		}
	}
}

// XONSH_STRING_CHARS — characters that are NOT word breaks.
// From tokenize.py: regular characters that are not in Special, Bracket,
// or Operator should be classified as word characters.
// Notable: : . , @ are Special in xonsh's Python tokenizer but in
// shlex's subprocess-mode classifier they are word characters because
// shlex treats them as part of the word (e.g. file.txt, user@host).
func TestVerify_XonshStringCharacters(t *testing.T) {
	classifier := XonshFormat().Classifier()

	// Regular string characters should NOT be wordbreaks
	regularChars := "abcXYZ012_-.@/%+"
	for _, r := range regularChars {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("regular char %q: classifier=wordbreakRuneClass, want non-wordbreak", r)
		}
	}
}

// XONSH_QUOTE_CHARS — from tokenize.py endpats:
// ' and " are the quote characters (Python string literals).
// " is the escaping quote (double quote) — backslash escapes work inside.
// ' is the non-escaping quote (single quote) — limited escapes via
// NonEscapingQuoteBackslashEscapes.
func TestVerify_XonshQuoteChars(t *testing.T) {
	classifier := XonshFormat().Classifier()

	// " is the escaping quote (double quote) — variable expansion, escapes
	if got := classifier.ClassifyRune('"'); got != escapingQuoteRuneClass {
		t.Errorf("\" classifier=%v, want escapingQuoteRuneClass", got)
	}
	// ' is the non-escaping quote (single quote)
	if got := classifier.ClassifyRune('\''); got != nonEscapingQuoteRuneClass {
		t.Errorf("' classifier=%v, want nonEscapingQuoteRuneClass", got)
	}
}

// XONSH_ESCAPE_CHAR — from tokenize.py: Single/Double regexes use \\.
// Backslash is the escape character in xonsh (Python rules).
func TestVerify_XonshEscapeChar(t *testing.T) {
	classifier := XonshFormat().Classifier()
	if got := classifier.ClassifyRune('\\'); got != escapeRuneClass {
		t.Errorf("\\ classifier=%v, want escapeRuneClass", got)
	}
}

// XONSH_COMMENT_CHAR — xonsh uses # for comments (Python and shell).
// In subprocess mode, # at token start begins a comment.
func TestVerify_XonshCommentChar(t *testing.T) {
	classifier := XonshFormat().Classifier()
	if got := classifier.ClassifyRune('#'); got != commentRuneClass {
		t.Errorf("# classifier=%v, want commentRuneClass", got)
	}
}

// Operator grammar — from tokenize.py Operator and lexer.py _op_map.
// Xonsh recognizes these subprocess operators:
//
//	|        → pipe (PIPE)
//	||       → logical OR (mapped to OR via handle_double_pipe)
//	&        → background/ampersand (AMPERSAND)
//	&&       → logical AND (mapped to AND via handle_double_amps)
//	;        → sequential separator (SEMI)
//	>        → output redirect (GT)
//	>>       → append output redirect (RSHIFT)
//	<        → input redirect (LT)
//
// shlex should classify these via bashWordbreakType (xonsh uses POSIX operators).
func TestVerify_XonshOperatorGrammar(t *testing.T) {
	tests := []struct {
		input   string
		opType  WordbreakType
		isPipe  bool // IsPipelineDelimiter
		isRedir bool // IsRedirect
	}{
		// Pipes (pipeline delimiters)
		{"|", WORDBREAK_PIPE, true, false},
		{"||", WORDBREAK_LIST_OR, true, false},

		// List operators (pipeline delimiters)
		{"&&", WORDBREAK_LIST_AND, true, false},
		{"&", WORDBREAK_LIST_ASYNC, true, false},
		{";", WORDBREAK_LIST_SEQUENTIAL, true, false},

		// Redirect operators
		{">", WORDBREAK_REDIRECT_OUTPUT, false, true},
		{">>", WORDBREAK_REDIRECT_OUTPUT_APPEND, false, true},
		{"<", WORDBREAK_REDIRECT_INPUT, false, true},
	}

	for _, tc := range tests {
		got := XonshFormat().ClassifyOperator(tc.input)
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

// Pipeline delimiters from xonsh lexer: |, ||, &, &&, ;
// These should all return IsPipelineDelimiter() = true
func TestVerify_XonshPipelineDelimiters(t *testing.T) {
	pipelineOps := []string{"|", "||", "&", "&&", ";"}
	for _, op := range pipelineOps {
		wbType := XonshFormat().ClassifyOperator(op)
		if !wbType.IsPipelineDelimiter() {
			t.Errorf("ClassifyOperator(%q).IsPipelineDelimiter() = false, want true", op)
		}
	}
}

// Redirect operators from xonsh: >, >>, <
// These should all return IsRedirect() = true
func TestVerify_XonshRedirectOperators(t *testing.T) {
	redirectOps := []string{">", ">>", "<"}
	for _, op := range redirectOps {
		wbType := XonshFormat().ClassifyOperator(op)
		if !wbType.IsRedirect() {
			t.Errorf("ClassifyOperator(%q).IsRedirect() = false, want true", op)
		}
	}
}

// Keyword operators — from lexer.py: NEED_WHITESPACE = frozenset(["and", "or"])
// In subprocess mode, "and" and "or" with surrounding whitespace are
// yielded as AND/OR tokens (pipeline delimiters).
// Other words (not, etc.) are NOT keyword operators.
func TestVerify_XonshKeywordOperators(t *testing.T) {
	kwOps := XonshFormat().KeywordOperators()
	if kwOps == nil {
		t.Fatal("KeywordOperators() = nil, want non-nil for xonsh")
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
}

// Verify keyword operators split pipelines end-to-end
func TestVerify_XonshKeywordOperatorPipelineSplit(t *testing.T) {
	// "and" splits pipelines
	tokens, err := SplitWith("echo foo and echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("xonsh 'and': %d pipelines, want 2", len(pipelines))
	}

	// "or" splits pipelines
	tokens, err = SplitWith("echo foo or echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines = tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("xonsh 'or': %d pipelines, want 2", len(pipelines))
	}

	// "not" is NOT a keyword operator in xonsh (it's a Python keyword,
	// not a subprocess separator) — should NOT split
	tokens, err = SplitWith("echo foo not echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines = tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("xonsh 'not': %d pipelines, want 1 (not is not a subprocess separator)", len(pipelines))
	}
}

// Keyword operators require surrounding whitespace — from lexer.py:
// NEED_WHITESPACE checks RE_NEED_WHITESPACE pattern.
// "fooand" or "echo fooand bar" should NOT trigger keyword splitting.
func TestVerify_XonshKeywordOperatorNoSplitWithoutWhitespace(t *testing.T) {
	tokens, err := SplitWith("echo fooand bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 1 {
		t.Errorf("xonsh fooand: %d pipelines, want 1 (keyword needs whitespace)", len(tokens.Pipelines()))
	}
}

// Double-quote escaping — from tokenize.py Double regex:
// Inside double quotes, \. matches backslash + any character.
// Python semantics: \ escapes the next character (any character).
// This differs from bash's CBSDQUOTE restricted set.
// shlex xonsh: EscapingQuoteEscapeChars() returns nil, meaning
// backslash escapes ANY character inside double quotes.
func TestVerify_XonshDoubleQuoteEscapeChars(t *testing.T) {
	// nil means backslash escapes any character (Python semantics)
	got := XonshFormat().EscapingQuoteEscapeChars()
	if got != nil {
		t.Errorf("EscapingQuoteEscapeChars() = %v, want nil (Python: backslash escapes any char)", got)
	}
}

// Verify double quote escape behavior end-to-end — Python rules.
// \X inside double quotes: backslash dropped, X emitted (for any X).
func TestVerify_XonshDoubleQuoteEscaping(t *testing.T) {
	// \" inside double quotes → literal "
	tokens, err := SplitWith(`echo "say \"hello\""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("xonsh \\\" in double: Words = %v, want [echo say \"hello\"]", words)
	}

	// \\ inside double quotes → literal \
	tokens, err = SplitWith(`echo "C:\\path"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path` {
		t.Errorf("xonsh \\\\ in double: Words = %v, want [echo C:\\path]", words)
	}

	// \n inside double quotes → literal \n (Python: backslash dropped, n emitted)
	tokens, err = SplitWith(`echo "hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hellonworld` {
		t.Errorf("xonsh \\n in double: Words = %v, want [echo hellonworld]", words)
	}

	// \t inside double quotes → literal t (backslash dropped, t emitted)
	tokens, err = SplitWith(`echo "a\tb"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `atb` {
		t.Errorf("xonsh \\t in double: Words = %v, want [echo atb]", words)
	}

	// \$ inside double quotes → literal $ (backslash dropped, $ emitted)
	tokens, err = SplitWith(`echo "cost: \$5"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: $5` {
		t.Errorf("xonsh \\$ in double: Words = %v, want [echo cost: $5]", words)
	}
}

// Single quote escapes — from tokenize.py Single regex: \\.
// Inside single quotes, \. matches backslash + any character.
// Python: \' and \\ are escapes. Other \X sequences keep both chars.
// shlex xonsh: NonEscapingQuoteBackslashEscapes() = true (\' and \\ work).
func TestVerify_XonshSingleQuoteEscapes(t *testing.T) {
	// \' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it\\'s'", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's" {
		t.Errorf("xonsh \\' in single: Words = %v, want [echo it's]", words)
	}

	// \\ inside single quotes → literal \
	tokens, err = SplitWith("echo 'C:\\\\path'", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path` {
		t.Errorf("xonsh \\\\ in single: Words = %v, want [echo C:\\path]", words)
	}
}

// Non-escape \X inside single quotes: both \ and X are literal.
// Python's tokenizer consumes \X (via \\. in the regex), but the
// unescape logic only converts \' and \\ — other \X sequences
// keep both characters.
func TestVerify_XonshSingleQuoteNonEscape(t *testing.T) {
	// \$ inside single quotes → literal \$ (NOT an escape)
	tokens, err := SplitWith(`echo 'cost: \$5'`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: \$5` {
		t.Errorf("xonsh \\$ in single: Words = %v, want [echo cost: \\$5]", words)
	}

	// \n inside single quotes → literal \n (NOT a newline)
	tokens, err = SplitWith(`echo 'hello\nworld'`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("xonsh \\n in single: Words = %v, want [echo hello\\nworld]", words)
	}
}

// Line continuation — from tokenize.py Ignore pattern:
// Whitespace + r"\\\r?\n" + Whitespace
// Backslash followed by \n or \r\n is a line continuation — consumed and removed.
// This applies both outside quotes and inside double quotes (Python semantics).
func TestVerify_XonshLineContinuation(t *testing.T) {
	// Outside quotes: \ + \n → removed (word continues)
	tokens, err := SplitWith("echo foo\\\nbar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation outside: words=%v, want [echo foobar]", words)
	}

	// Inside double quotes: \ + \n → removed (line continuation)
	tokens, err = SplitWith("echo \"foo\\\nbar\"", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation in double: words=%v, want [echo foobar]", words)
	}

	// CRLF variant: \ + \r\n → removed
	tokens, err = SplitWith("echo foo\\\r\nbar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation CRLF: words=%v, want [echo foobar]", words)
	}
}

// Comment: # starts a comment (Python and shell).
// From xonsh: # at token start begins a comment to end of line.
func TestVerify_XonshComment(t *testing.T) {
	tokens, err := SplitWith("echo hello # this is a comment", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello" {
		t.Errorf("xonsh comment: words=%v, want [echo hello]", words)
	}
}

// Triple quotes — from tokenize.py: Triple, Single3, Double3 regexes.
// Triple-single and triple-double quoted strings (Python syntax).
// shlex: TripleQuoteSupport() = true.
func TestVerify_XonshTripleQuoteSupport(t *testing.T) {
	if !XonshFormat().TripleQuoteSupport() {
		t.Error("TripleQuoteSupport() = false, want true (xonsh supports triple quotes)")
	}

	// Triple single quotes
	tokens, err := SplitWith(`echo '''hello world'''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("xonsh triple single: Words = %v, want [echo hello world]", words)
	}

	// Triple double quotes
	tokens, err = SplitWith(`echo """hello world"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("xonsh triple double: Words = %v, want [echo hello world]", words)
	}
}

// Triple quote: two quotes inside triple don't close the string.
// From Single3 regex: a quote not followed by two more is OK.
// From Double3 regex: same for double quotes.
func TestVerify_XonshTripleQuoteEmbeddedQuotes(t *testing.T) {
	// '' inside '''...''' should NOT close
	tokens, err := SplitWith(`echo '''hello''there'''`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello''there` {
		t.Errorf("xonsh '' in triple: Words = %v, want [echo hello''there]", words)
	}

	// "" inside """...""" should NOT close
	tokens, err = SplitWith(`echo """hello""there"""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello""there` {
		t.Errorf("xonsh \"\" in triple: Words = %v, want [echo hello\"\"there]", words)
	}
}

// Triple quote escape behavior — from Single3/Double3 regexes: \\.
// Inside triple double quotes, \" is an escape (backslash dropped, " emitted).
func TestVerify_XonshTripleDoubleQuoteEscape(t *testing.T) {
	tokens, err := SplitWith(`echo """say \"hello\""""`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `say "hello"` {
		t.Errorf("xonsh triple double escape: Value = %q, want %q", last.Value, `say "hello"`)
	}
}

// Raw string prefixes — from tokenize.py StringPrefix:
// r, R, b, B, br, bR, Br, BR, rb, rB, Rb, RB, etc.
// Raw strings suppress escape processing inside double quotes.
// shlex: RawPrefixSupport() = true.
func TestVerify_XonshRawPrefixSupport(t *testing.T) {
	if !XonshFormat().RawPrefixSupport() {
		t.Error("RawPrefixSupport() = false, want true (xonsh supports raw strings)")
	}

	// r'...' — raw single-quoted string, \ is literal
	tokens, err := SplitWith(`echo r'C:\path'`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `rC:\path` {
		t.Errorf("xonsh r'': Words = %v, want [echo rC:\\path]", words)
	}

	// r"..." — raw double-quoted string, \ is literal (no escape processing)
	tokens, err = SplitWith(`echo r"C:\path"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `rC:\path` {
		t.Errorf("xonsh r\"\": Words = %v, want [echo rC:\\path]", words)
	}

	// br"..." — valid raw prefix (b + r), \ is literal
	tokens, err = SplitWith(`echo br"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `brhello\nworld` {
		t.Errorf("xonsh br prefix: Words = %v, want [echo brhello\\nworld]", words)
	}
}

// Invalid raw prefix should NOT suppress escapes.
// 'xr' is not a valid Python string prefix, so r"C:\path" after xr
// should process escapes normally.
func TestVerify_XonshInvalidPrefixDoesNotSuppress(t *testing.T) {
	// xr"..." — 'xr' is not a valid prefix, so escapes are processed
	tokens, err := SplitWith(`echo xr"hello\nworld"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// \n processed as escape (backslash dropped, n emitted) since raw prefix is NOT active
	if words[1] != "xrhellonworld" {
		t.Errorf("xonsh xr false positive: Words = %v, want [echo xrhellonworld]", words)
	}
}

// Stream redirects — from lexer.py _redir_names and _redir_map:
// _redir_names = ("out", "all", "err", "e", "2", "a", "&", "1", "o")
// Redirect operators: e>, o>, a>, err>, out>, e>>, o>>, a>>, etc.
// Also pipe-channel variants: e>p, o>p, a>p
func TestVerify_XonshStreamRedirects(t *testing.T) {
	tests := []struct {
		input   string
		rawOp   string
		wbType  WordbreakType
		isRedir bool
	}{
		// Stderr redirects
		{"echo foo e> bar", "e>", WORDBREAK_REDIRECT_OUTPUT, true},
		{"echo foo err> bar", "err>", WORDBREAK_REDIRECT_OUTPUT, true},
		{"echo foo e>> bar", "e>>", WORDBREAK_REDIRECT_OUTPUT, true},
		// Stdout redirects
		{"echo foo o> bar", "o>", WORDBREAK_REDIRECT_OUTPUT, true},
		{"echo foo out> bar", "out>", WORDBREAK_REDIRECT_OUTPUT, true},
		// All redirects
		{"echo foo a> bar", "a>", WORDBREAK_REDIRECT_OUTPUT_BOTH, true},
		{"echo foo all> bar", "all>", WORDBREAK_REDIRECT_OUTPUT_BOTH, true},
	}

	for _, tc := range tests {
		tokens, err := SplitWith(tc.input, XonshFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", tc.input, err)
		}
		found := false
		for _, tok := range tokens {
			if tok.Type == WORDBREAK_TOKEN && tok.RawValue == tc.rawOp {
				if tok.WordbreakType != tc.wbType {
					t.Errorf("stream redirect %q: WordbreakType = %v, want %v", tc.rawOp, tok.WordbreakType, tc.wbType)
				}
				if tok.WordbreakType.IsRedirect() != tc.isRedir {
					t.Errorf("stream redirect %q: IsRedirect() = %v, want %v", tc.rawOp, tok.WordbreakType.IsRedirect(), tc.isRedir)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("stream redirect %q: no %s wordbreak token found", tc.rawOp, tc.rawOp)
		}
	}
}

// Pipe-channel redirects — from _redir_map:
// a>p, all>p → all streams to pipe
// e>p, err>p → stderr to pipe
// o>p, out>p → stdout to pipe (implied)
func TestVerify_XonshPipeChannelRedirects(t *testing.T) {
	pipeChannelOps := []string{"e>p", "o>p", "a>p"}
	for _, op := range pipeChannelOps {
		input := "echo foo " + op + " bar"
		tokens, err := SplitWith(input, XonshFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", input, err)
		}
		found := false
		for _, tok := range tokens {
			if tok.Type == WORDBREAK_TOKEN && tok.RawValue == op {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pipe-channel redirect %q: no %s wordbreak token found", op, op)
		}
	}
}

// Quoted stream redirect should NOT be merged — from PostProcess:
// Only bare words (Value == RawValue) are merged with > as stream redirects.
// Quoted 'e' is a string literal, not an operator.
func TestVerify_XonshQuotedStreamRedirectNotMerged(t *testing.T) {
	tokens, err := SplitWith("echo foo 'e'> bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "e>" {
			t.Errorf("quoted 'e' should not be merged with > as stream redirect")
		}
	}
}

// Command substitution — from xonsh lexer: $() and @() are operators.
// shlex PostProcess merges $ + ( and @ + ( into WORDBREAK_SUBSTITUTION_OPEN.
func TestVerify_XonshCommandSubstitution(t *testing.T) {
	// $(...) — command substitution
	tokens, err := SplitWith("echo $(ls)", XonshFormat())
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
		t.Error("xonsh command substitution: $( not merged into WORDBREAK_SUBSTITUTION_OPEN")
	}
	if !foundClose {
		t.Error("xonsh command substitution: ) not classified as WORDBREAK_SUBSTITUTION_CLOSE")
	}
}

// @() — Python expression substitution
func TestVerify_XonshAtSubstitution(t *testing.T) {
	tokens, err := SplitWith("echo @(1+2)", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN && tok.Value == "@(" {
			found = true
		}
	}
	if !found {
		t.Error("xonsh @() substitution: @ ( not merged into WORDBREAK_SUBSTITUTION_OPEN")
	}
}

// Bare () — xonsh uses () for subprocess grouping.
// PostProcess reclassifies standalone ( as WORDBREAK_SUBSTITUTION_OPEN
// and ) as WORDBREAK_SUBSTITUTION_CLOSE.
func TestVerify_XonshBareParens(t *testing.T) {
	tokens, err := SplitWith("echo (ls)", XonshFormat())
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
		t.Error("xonsh bare (): ( not classified as WORDBREAK_SUBSTITUTION_OPEN")
	}
	if !foundClose {
		t.Error("xonsh bare (): ) not classified as WORDBREAK_SUBSTITUTION_CLOSE")
	}
}

// Substitution delimiters should not split pipelines.
// Pipe inside () should not split the outer pipeline.
func TestVerify_XonshSubstitutionDoesntSplitPipeline(t *testing.T) {
	tokens, err := SplitWith("echo $(ls | grep foo)", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("xonsh subst pipeline: %d pipelines, want 1 (pipe inside () shouldn't split)", len(pipelines))
	}
}

// QuoteWord for JoinWith — xonsh uses Python single-quote wrapping
// with \' and \\ escapes. Double quotes are not escaped (Python style).
func TestVerify_XonshQuoteWord(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Safe words: no quoting needed
		{"hello", "hello"},
		{"abc123", "abc123"},
		{"hello-world", "hello-world"},
		// Special chars need quoting
		{"hello world", `'hello world'`},
		{`with'quote`, `'with\'quote'`},
		{`with\backslash`, `'with\\backslash'`},
		// Double quote is safe inside single-quoted string
		{`with"quote`, `'with"quote'`},
		// Empty string
		{"", `""`},
	}

	for _, tc := range tests {
		got := xonshQuoteWord(tc.input)
		if got != tc.want {
			t.Errorf("xonshQuoteWord(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// No word splitting on variable expansion — xonsh (like fish) doesn't
// split on $VAR expansion. Spaces inside quotes don't split words.
func TestVerify_XonshNoWordSplitting(t *testing.T) {
	// Inside double quotes, spaces don't split
	tokens, err := SplitWith(`echo "hello world"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("xonsh no word split: Words = %v, want [echo hello world]", words)
	}

	// Outside quotes, spaces DO split (literal source whitespace)
	tokens, err = SplitWith("echo hello world", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 3 {
		t.Errorf("xonsh word split: Words = %v, want 3 words", words)
	}
}

// Escape outside quotes: backslash escapes the next character.
// From tokenize.py: escape mode consumes next char literally.
func TestVerify_XonshEscapeOutsideQuotes(t *testing.T) {
	// Escaped space → one word
	tokens, err := SplitWith(`echo a\ b`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("xonsh escaped space: Words = %v, want [echo a b]", words)
	}

	// Escaped pipe → literal | (not a pipeline delimiter)
	tokens, err = SplitWith(`echo foo\|bar`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo|bar" {
		t.Errorf("xonsh escaped |: Words = %v, want [echo foo|bar]", words)
	}
}

// Xonsh has no COMP_WORDBREAKS dependency — the wordbreak set is fixed.
func TestVerify_XonshNoEnvVarDependency(t *testing.T) {
	classifier := XonshFormat().Classifier()
	for _, r := range "|<>&;()" {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("xonsh classifier %q: got %v, want wordbreakRuneClass", r, got)
		}
	}
}

// LineContinuationEscaper — xonsh implements IsLineContinuation.
// From tokenize.py Ignore pattern: \\\r?\n is line continuation.
// \n and \r should both return true.
func TestVerify_XonshLineContinuationInterface(t *testing.T) {
	lc, ok := XonshFormat().(LineContinuationEscaper)
	if !ok {
		t.Fatal("XonshFormat() does not implement LineContinuationEscaper")
	}
	if !lc.IsLineContinuation('\n') {
		t.Error("IsLineContinuation('\\n') = false, want true")
	}
	if !lc.IsLineContinuation('\r') {
		t.Error("IsLineContinuation('\\r') = false, want true")
	}
	if lc.IsLineContinuation('a') {
		t.Error("IsLineContinuation('a') = true, want false")
	}
}
