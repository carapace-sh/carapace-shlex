package shlex

import (
	"os"
	"testing"
)

// Verification tests cross-checking shlex against bash source code.
//
// Authoritative sources verified:
//   - syntax.h: shell_meta_chars, shell_break_chars, slashify_in_quotes,
//     shell_quote_chars, shell_exp_chars, shell_glob_chars
//   - bashline.c: bash_completer_word_break_characters, check_redir()
//   - parse.y: operator tokenization (|, ||, &, &&, ;, ;;, ;&, ;;&, >, >>,
//     >|, <<, <<<, <&, <>, &>, &>>, |&)
//   - lib/sh/shquote.c: sh_backslash_quote (CBSDQUOTE quoting)
//   - subst.c: CBSDQUOTE usage in command substitution

// BASH_META_CHARS = "()<>;&|" — from syntax.h: shell_meta_chars
// These are the characters that separate words at the parser level.
func TestVerify_BashMetaChars(t *testing.T) {
	const bashMetaChars = "()<>;&|"
	classifier := BashFormat().Classifier()
	for _, r := range bashMetaChars {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("meta char %q: classifier=%v, want wordbreakRuneClass", r, got)
		}
	}
}

// BASH_BREAK_CHARS = "()<>;&| \t\n" — from syntax.h: shell_break_chars
// These are the characters readline uses to break words.
func TestVerify_BashBreakChars(t *testing.T) {
	const bashBreakChars = "()<>;&| \t\n"
	classifier := BashFormat().Classifier()
	for _, r := range bashBreakChars {
		got := classifier.ClassifyRune(r)
		// Space chars are classified as space, the rest as wordbreak
		switch r {
		case ' ', '\t', '\n':
			if got != spaceRuneClass {
				t.Errorf("break char %q: classifier=%v, want spaceRuneClass", r, got)
			}
		default:
			if got != wordbreakRuneClass {
				t.Errorf("break char %q: classifier=%v, want wordbreakRuneClass", r, got)
			}
		}
	}
}

// BASH_QUOTE_CHARS = "\"`'" — from syntax.h: shell_quote_chars
// " and ' are quotes; ` is backquote (CBACKQ).
func TestVerify_BashQuoteChars(t *testing.T) {
	classifier := BashFormat().Classifier()

	// " is the escaping quote (double quote)
	if got := classifier.ClassifyRune('"'); got != escapingQuoteRuneClass {
		t.Errorf("\" classifier=%v, want escapingQuoteRuneClass", got)
	}
	// ' is the non-escaping quote (single quote)
	if got := classifier.ClassifyRune('\''); got != nonEscapingQuoteRuneClass {
		t.Errorf("' classifier=%v, want nonEscapingQuoteRuneClass", got)
	}
}

// SLASHIFY_IN_QUOTES = "\\`$\"\n" — from syntax.h: slashify_in_quotes
// These are the characters that backslash can escape inside double quotes (CBSDQUOTE).
// The shlex EscapingQuoteEscapeChars() must match this set.
func TestVerify_SlashifyInQuotes(t *testing.T) {
	// bash source: slashify_in_quotes = "\\`$\"\n"
	expected := map[rune]bool{
		'\\': true,
		'`':  true,
		'$':  true,
		'"':  true,
		'\n': true,
	}
	got := BashFormat().EscapingQuoteEscapeChars()
	for r, want := range expected {
		if got[r] != want {
			t.Errorf("EscapingQuoteEscapeChars[%q]=%v, want %v", r, got[r], want)
		}
	}
	// Characters NOT in slashify_in_quotes should not be escapable
	for _, r := range "abc123 \t" {
		if got[r] {
			t.Errorf("EscapingQuoteEscapeChars[%q]=true, want false (not in CBSDQUOTE)", r)
		}
	}
}

// SLASHIFY_IN_HERE_DOCUMENT = "\\`$" — from syntax.h: slashify_in_here_document
// (Not directly used by shlex, but documents that here-doc escapes are a subset)

// COMP_WORDBREAKS — from bashline.c
// bash_completer_word_break_characters = " \t\n\"'@><=;|&(:"
// bash_nohostname_word_break_characters = " \t\n\"'><=;|&(:"
func TestVerify_CompletionWordBreaks(t *testing.T) {
	// Default (with hostname completion): includes @
	const defaultWB = " \t\n\"'@><=;|&(:"
	// Without hostname completion: @ removed
	const noHostnameWB = " \t\n\"'><=;|&(:"

	// Save and clear COMP_WORDBREAKS to test the default
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	classifier := BashFormat().Classifier()

	// The shlex BASH_WORDBREAKS constant is " \t\r\n\"'@><=;|&():"
	// Note: shlex includes \r (carriage return) and uses () instead of (
	// This is because ( and ) are both wordbreaks in bash, and \r is
	// treated as whitespace (isblank doesn't include \r in C locale,
	// but bash's spaceRunes in shlex includes \r as a space).
	for _, r := range defaultWB {
		got := classifier.ClassifyRune(r)
		switch r {
		case ' ', '\t', '\n':
			if got != spaceRuneClass {
				t.Errorf("defaultWB %q: classifier=%v, want spaceRuneClass", r, got)
			}
		case '"':
			if got != escapingQuoteRuneClass {
				t.Errorf("defaultWB %q: classifier=%v, want escapingQuoteRuneClass", r, got)
			}
		case '\'':
			if got != nonEscapingQuoteRuneClass {
				t.Errorf("defaultWB %q: classifier=%v, want nonEscapingQuoteRuneClass", r, got)
			}
		default:
			if got != wordbreakRuneClass {
				t.Errorf("defaultWB %q: classifier=%v, want wordbreakRuneClass", r, got)
			}
		}
	}

	// Test with hostname completion off (no @)
	os.Setenv("COMP_WORDBREAKS", noHostnameWB)
	classifier2 := BashFormat().Classifier()
	// @ should NOT be a wordbreak when COMP_WORDBREAKS doesn't include it
	if got := classifier2.ClassifyRune('@'); got == wordbreakRuneClass {
		t.Error("@ should not be wordbreak when COMP_WORDBREAKS excludes it")
	}
}

// Operator grammar verification — from parse.y
// bash recognizes these multi-character operators:
//
//	|  → pipe
//	|| → logical OR
//	&  → background/async
//	&& → logical AND
//	;  → sequential
//	;; → case terminator
//	;& → case fallthrough
//	;;& → case next pattern
//	>  → output redirect
//	>> → append output redirect
//	>| → force output redirect (noclobber override)
//	<< → here document
//	<<< → here string
//	<  → input redirect
//	<& → duplicate input fd
//	<> → input-output redirect
//	&> → redirect stdout+stderr
//	&>> → append stdout+stderr
//	|& → pipe with stderr
func TestVerify_OperatorGrammar(t *testing.T) {
	// Save and clear COMP_WORDBREAKS
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tests := []struct {
		input    string
		opType   WordbreakType
		opString string
	}{
		{"|", WORDBREAK_PIPE, "|"},
		{"||", WORDBREAK_LIST_OR, "||"},
		{"&", WORDBREAK_LIST_ASYNC, "&"},
		{"&&", WORDBREAK_LIST_AND, "&&"},
		{";", WORDBREAK_LIST_SEQUENTIAL, ";"},
		{";;", WORDBREAK_LIST_SEQUENTIAL_DOUBLE, ";;"},
		{";&", WORDBREAK_LIST_FALLTHROUGH, ";&"},
		{";;&", WORDBREAK_LIST_CASE_NEXT, ";;&"},
		{">", WORDBREAK_REDIRECT_OUTPUT, ">"},
		{">>", WORDBREAK_REDIRECT_OUTPUT_APPEND, ">>"},
		{">|", WORDBREAK_REDIRECT_OUTPUT_FORCE, ">|"},
		{"<<", WORDBREAK_REDIRECT_HERE_DOC, "<<"},
		{"<<<", WORDBREAK_REDIRECT_INPUT_STRING, "<<<"},
		{"<", WORDBREAK_REDIRECT_INPUT, "<"},
		{"<&", WORDBREAK_REDIRECT_INPUT_DUPLICATE, "<&"},
		{"<>", WORDBREAK_REDIRECT_INPUT_OUTPUT, "<>"},
		{"&>", WORDBREAK_REDIRECT_OUTPUT_BOTH, "&>"},
		{"&>>", WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND, "&>>"},
		{"|&", WORDBREAK_PIPE_WITH_STDERR, "|&"},
	}

	for _, tc := range tests {
		tokens, err := SplitWith("echo "+tc.input+" foo", BashFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", "echo "+tc.input+" foo", err)
		}
		// Find the wordbreak token
		found := false
		for _, tok := range tokens {
			if tok.Type == WORDBREAK_TOKEN {
				if tok.WordbreakType == tc.opType {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("operator %q: expected WordbreakType %v, not found in tokens", tc.input, tc.opString)
		}
	}
}

// check_redir from bashline.c recognizes these two-char combos as
// NOT command separators (i.e., they're redirects):
//
//	>&  (this_char=='&' && prev=='>')
//	<&  (this_char=='&' && prev=='<')
//	>|  (this_char=='|' && prev=='>')
//
// shlex should classify these as redirects, not pipeline delimiters.
func TestVerify_CheckRedirRedirects(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// >& and <& should be redirects (IsRedirect=true), not pipeline delimiters
	redirectOps := []string{">&", "<&", ">|"}
	for _, op := range redirectOps {
		wbType := bashWordbreakType(op)
		if !wbType.IsRedirect() {
			t.Errorf("bashWordbreakType(%q).IsRedirect() = false, want true", op)
		}
		if wbType.IsPipelineDelimiter() {
			t.Errorf("bashWordbreakType(%q).IsPipelineDelimiter() = true, want false", op)
		}
	}
}

// Pipeline delimiters from parse.y:
// |, ||, &, &&, ;, ;;, ;&, ;;&, |&
// These should all return IsPipelineDelimiter() = true
func TestVerify_PipelineDelimiters(t *testing.T) {
	pipelineOps := []string{
		"|", "||", "&", "&&", ";", ";;", ";&", ";;&", "|&",
	}
	for _, op := range pipelineOps {
		wbType := bashWordbreakType(op)
		if !wbType.IsPipelineDelimiter() {
			t.Errorf("bashWordbreakType(%q).IsPipelineDelimiter() = false, want true", op)
		}
	}
}

// Redirect operators from parse.y (Redirections section):
// <, >, >>, >|, <<, <<<, <&, <>, &>, &>>
// These should all return IsRedirect() = true
func TestVerify_RedirectOperators(t *testing.T) {
	redirectOps := []string{
		"<", ">", ">>", ">|", "<<", "<<<", "<&", "<>", "&>", "&>>",
	}
	for _, op := range redirectOps {
		wbType := bashWordbreakType(op)
		if !wbType.IsRedirect() {
			t.Errorf("bashWordbreakType(%q).IsRedirect() = false, want true", op)
		}
	}
}

// Double-quote backslash escaping — from syntax.h slashify_in_quotes and
// lib/sh/shquote.c sh_backslash_quote:
// Inside double quotes, \ only escapes: \, `, $, ", \n
// \ before other chars is preserved literally (both \ and the char).
func TestVerify_DoubleQuoteBackslashEscaping(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Characters that \ CAN escape inside double quotes (CBSDQUOTE set)
	escapable := `\` + "`" + `$"` + "\n"
	for _, r := range escapable {
		input := `echo "\` + string(r) + `"`
		if r == '\n' {
			input = "echo \"\\\n\""
		}
		tokens, err := SplitWith(input, BashFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", input, err)
		}
		words := tokens.Words()
		last := words[len(words)-1]
		// The escaped char should appear in the value (backslash removed)
		if r == '\n' {
			// Line continuation: both \ and \n removed
			if last.Value != "" {
				t.Errorf("backslash-newline in double: Value=%q, want empty (line continuation)", last.Value)
			}
		} else {
			if last.Value != string(r) {
				t.Errorf("escaped %q in double: Value=%q, want %q", r, last.Value, string(r))
			}
		}
	}

	// Characters that \ CANNOT escape inside double quotes (not in CBSDQUOTE)
	// \ before these is preserved literally (both \ and char in value)
	nonEscapable := "abc123 \t"
	for _, r := range nonEscapable {
		if r == ' ' || r == '\t' {
			continue // spaces break the word
		}
		input := `echo "\` + string(r) + `"`
		tokens, err := SplitWith(input, BashFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", input, err)
		}
		words := tokens.Words()
		last := words[len(words)-1]
		// Both \ and the char should be in the value
		expected := `\` + string(r)
		if last.Value != expected {
			t.Errorf("non-escaped %q in double: Value=%q, want %q", r, last.Value, expected)
		}
	}
}

// Single quotes preserve literal value of EVERY character — from bash manual
// No escape is possible inside single quotes, not even \'
func TestVerify_SingleQuoteNoEscape(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// \ is literal inside single quotes
	tokens, err := SplitWith(`echo 'hello\nworld'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `hello\nworld` {
		t.Errorf("single quote literal: Value=%q, want %q", last.Value, `hello\nworld`)
	}

	// $ is literal inside single quotes
	tokens, err = SplitWith(`echo '$HOME'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words()
	last = words[len(words)-1]
	if last.Value != `$HOME` {
		t.Errorf("single quote $ literal: Value=%q, want $HOME", last.Value)
	}

	// ` is literal inside single quotes
	tokens, err = SplitWith(`echo '`+"`"+`cmd`+"`"+`'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words()
	last = words[len(words)-1]
	if last.Value != "`cmd`" {
		t.Errorf("single quote backtick literal: Value=%q, want `cmd`", last.Value)
	}
}

// Line continuation: backslash + newline is removed — from bash manual
// "A non-quoted backslash preserves the literal value of the next character,
// with the exception of newline."
func TestVerify_LineContinuation(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Outside quotes: \ + \n → removed (word continues)
	tokens, err := SplitWith("echo foo\\\nbar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation: words=%v, want [echo foobar]", words)
	}

	// Inside double quotes: \ + \n → removed (line continuation)
	tokens, err = SplitWith("echo \"foo\\\nbar\"", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation in double: words=%v, want [echo foobar]", words)
	}

	// CRLF variant: \ + \r\n → removed
	tokens, err = SplitWith("echo foo\\\r\nbar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation CRLF: words=%v, want [echo foobar]", words)
	}
}

// Comment: # starts a comment (CSHBRK is not set for #, but the tokenizer
// handles it as commentRuneClass). From bash manual: "a word beginning
// with # causes that word and all remaining characters on that line to
// be ignored."
func TestVerify_Comment(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo hello # this is a comment", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello" {
		t.Errorf("comment: words=%v, want [echo hello]", words)
	}
}

// Quoting for JoinWith — from lib/sh/shquote.c sh_backslash_quote:
// Double-quote wrapping with \ escaping for CBSDQUOTE chars (minus \n).
// shlex's posixQuoteWord should match this behavior.
func TestVerify_QuoteWord(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tests := []struct {
		input string
		want  string
	}{
		// Safe words: no quoting needed
		{"hello", "hello"},
		{"abc123", "abc123"},
		// Special chars need quoting
		{"hello world", `"hello world"`},
		{`with"quote`, `"with\"quote"`},
		{`with\backslash`, `"with\\backslash"`},
		{`with$dollar`, `"with\$dollar"`},
		{`with` + "`" + `backtick`, `"with\` + "`" + `backtick"`},
		{`with
newline`, `"with\` + "\n" + `newline"`},
		// Empty string
		{"", `""`},
	}

	for _, tc := range tests {
		got := posixQuoteWord(tc.input)
		if got != tc.want {
			t.Errorf("posixQuoteWord(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// COMMAND_SEPARATORS from bashline.c: ";|&{(`"
// These are the characters that indicate a command boundary.
// The shlex pipeline delimiter set should cover ;, |, & (and their
// multi-char variants). Note: { and ` are handled differently in shlex
// (backtick is CBACKQ, { is not a wordbreak in the default COMP_WORDBREAKS).
func TestVerify_CommandSeparators(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// ; | & should all split pipelines
	separators := []string{";", "|", "&"}
	for _, sep := range separators {
		input := "echo foo" + sep + " bar"
		tokens, err := SplitWith(input, BashFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", input, err)
		}
		pipelines := tokens.Pipelines()
		if len(pipelines) < 2 {
			t.Errorf("command separator %q: got %d pipelines, want >= 2", sep, len(pipelines))
		}
	}
}

// fd prefix in redirects — from parse.y and bashline.c check_redir
// A numeric token immediately before a redirect operator is the fd prefix.
// FilterRedirects should filter it out.
func TestVerify_FdPrefixInRedirects(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// 2> should filter out the "2" as fd prefix
	tokens, err := SplitWith("echo 2> file", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipeline := tokens.CurrentPipeline()
	filtered := pipeline.FilterRedirects()
	words := filtered.Words().Strings()

	// "2" should be filtered out, leaving [echo file]
	found := false
	for _, w := range words {
		if w == "2" {
			found = true
		}
	}
	if found {
		t.Errorf("fd prefix 2> : words=%v, expected 2 to be filtered", words)
	}
}

// Process substitution: <( and >( — from parse.y (PROCESS_SUBSTITUTION)
// These should be recognized as substitution openers, not plain redirects.
func TestVerify_ProcessSubstitution(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	for _, op := range []string{"<(", ">(", ")"} {
		input := "echo " + op
		if op == ")" {
			input = "echo <(cat) " + op
		}
		tokens, err := SplitWith(input, BashFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", input, err)
		}
		found := false
		for _, tok := range tokens {
			if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN ||
				tok.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("process substitution %q: no substitution token found", op)
		}
	}
}

// Command substitution: $(...) — from parse.y
// $ + ( should merge into WORDBREAK_SUBSTITUTION_OPEN
func TestVerify_CommandSubstitution(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo $(ls)", BashFormat())
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
		t.Error("command substitution: $( not merged into WORDBREAK_SUBSTITUTION_OPEN")
	}
	if !foundClose {
		t.Error("command substitution: ) not classified as WORDBREAK_SUBSTITUTION_CLOSE")
	}
}

// Arithmetic expansion: $((...)) — from parse.y
// $(( should merge into a single opener, )) into a single closer
func TestVerify_ArithmeticExpansion(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo $((1+2))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	foundArithOpen := false
	foundArithClose := false
	for _, tok := range tokens {
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN {
			if isArithmeticOpener(tok) {
				foundArithOpen = true
			}
		}
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE {
			if isArithmeticCloser(tok) {
				foundArithClose = true
			}
		}
	}
	if !foundArithOpen {
		t.Error("arithmetic: $(( not detected as arithmetic opener")
	}
	if !foundArithClose {
		t.Error("arithmetic: )) not detected as arithmetic closer")
	}
}

// shell_exp_chars from syntax.h: "$<>" (with PROCESS_SUBSTITUTION)
// $ is the expansion character. shlex doesn't treat $ as a wordbreak
// (it's part of words), which is correct — $ is CEXP, not CSHBRK.
func TestVerify_DollarNotWordbreak(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	classifier := BashFormat().Classifier()
	got := classifier.ClassifyRune('$')
	if got == wordbreakRuneClass {
		t.Error("$ should not be a wordbreak (it's CEXP, not CSHBRK)")
	}
}

// shell_glob_chars from syntax.h: "*?[]^"
// These are glob characters, not wordbreaks. shlex correctly does not
// include them in wordbreaks.
func TestVerify_GlobCharsNotWordbreak(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	classifier := BashFormat().Classifier()
	for _, r := range "*?[]^" {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("glob char %q should not be wordbreak", r)
		}
	}
}

// ext_glob_chars from syntax.h: "@*+?!" (with EXTENDED_GLOB)
// These are extended glob pattern chars, not wordbreaks.
// Note: @ IS in COMP_WORDBREAKS but as a completion wordbreak, not
// a shell break char.
func TestVerify_ExtGlobCharsNotShellBreak(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Without COMP_WORDBREAKS, @ should be in BASH_WORDBREAKS
	classifier := BashFormat().Classifier()
	got := classifier.ClassifyRune('@')
	if got != wordbreakRuneClass {
		t.Error("@ should be wordbreak (it's in BASH_WORDBREAKS)")
	}

	// + and ! are NOT in BASH_WORDBREAKS
	for _, r := range "+!" {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("ext glob char %q should not be wordbreak", r)
		}
	}
}
