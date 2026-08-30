package shlex

import (
	"os"
	"testing"
)

// Verification tests cross-checking shlex against zsh source code.
//
// Authoritative sources verified:
//   - zsh.h: SPECCHARS, lextok enum, redirection enum, IS_REDIROP macro,
//     RCQUOTES option
//   - lex.c: initlextabs() (lx1/lx2 character tables), gettok() (operator
//     recognition), gettokstr() LX2_QUOTE case (RC_QUOTES), dquote_parse()
//     (backslash escaping inside double quotes), tokstrings[] (operator strings)
//   - options.c: RCQUOTES option definition (OPT_EMULATE, set by default in zsh)

// SPECCHARS from zsh.h: "#$^*()=|{}[]`<>?~;&\n\t \\'\""
// These are all characters that need quoting if meant literally.
// The subset that are wordbreaks in shlex (via BASH_WORDBREAKS) should
// match the word-breaking characters in zsh's lexer (lx1 + LX2_BREAK).
//
// zsh lx1 (initial-position special chars): \, \n, ;, !, &, |, (, ), {, }, [, ], <, >
// zsh lx2 LX2_BREAK: ;, & (explicitly set), and whitespace (inblank)
// zsh lx2 also has: ), |, $, [, ], ~, (, {, }, >, <, =, \, ', ", `, ,, -, !
//
//	These have their own LX2 actions, not LX2_BREAK — they may or may not
//	break words depending on context.
//
// shlex uses BASH_WORDBREAKS for zsh (since zsh completion uses a similar set).
// The core word-breaking characters (separators + redirects + operators) match.
func TestVerify_ZshWordBreakChars(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Characters that zsh's lexer treats as word-breaking (LX1 special or LX2_BREAK).
	// Whitespace separators are classified as space, not wordbreak.
	const zshWordBreakSeparators = "()<>;&|"
	const zshSpaceSeparators = " \t\r\n"

	classifier := ZshFormat().Classifier()

	for _, r := range zshWordBreakSeparators {
		got := classifier.ClassifyRune(r)
		if got != wordbreakRuneClass {
			t.Errorf("zsh separator %q: classifier=%v, want wordbreakRuneClass", r, got)
		}
	}

	for _, r := range zshSpaceSeparators {
		got := classifier.ClassifyRune(r)
		if got != spaceRuneClass {
			t.Errorf("zsh whitespace %q: classifier=%v, want spaceRuneClass", r, got)
		}
	}
}

// Quote chars from zsh.h + lex.c:
// " is the escaping quote (double quote, LX2_DQUOTE)
// ' is the non-escaping quote (single quote, LX2_QUOTE)
// ` is backtick (LX2_BQUOTE) — command substitution, not a quote in shlex terms
func TestVerify_ZshQuoteChars(t *testing.T) {
	classifier := ZshFormat().Classifier()

	if got := classifier.ClassifyRune('"'); got != escapingQuoteRuneClass {
		t.Errorf("\" classifier=%v, want escapingQuoteRuneClass", got)
	}
	if got := classifier.ClassifyRune('\''); got != nonEscapingQuoteRuneClass {
		t.Errorf("' classifier=%v, want nonEscapingQuoteRuneClass", got)
	}
}

// RC_QUOTES — from options.c (RCQUOTES, OPT_EMULATE) and lex.c LX2_QUOTE case.
// zsh sets RC_QUOTES by default in native zsh emulation mode.
// When RC_QUOTES is set, ” inside single quotes produces a literal '.
//
// From lex.c gettokstr() LX2_QUOTE case:
//
//	e = hgetc();
//	if (e != '\'' || unset(RCQUOTES) || strquote)
//	    break;
//	add(c);
//
// shlex's zshFormat has NonEscapingQuoteEscapes() = true, which enables the
// ” → ' behavior in the QUOTING_STATE handler.
func TestVerify_ZshRCQuotes(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	if !ZshFormat().NonEscapingQuoteEscapes() {
		t.Fatal("zsh NonEscapingQuoteEscapes() = false, want true (RC_QUOTES)")
	}

	// '' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it''s'", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's" {
		t.Errorf("RC_QUOTES: Words = %v, want [echo it's]", words)
	}

	// Multiple '' sequences
	tokens, err = SplitWith("echo '''hello'''", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "'hello'" {
		t.Errorf("RC_QUOTES multiple: Words = %v, want [echo 'hello']", words)
	}

	// Empty single-quoted string '' → empty value (not an escape)
	tokens, err = SplitWith("echo ''", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "" {
		t.Errorf("RC_QUOTES empty: Words = %v, want [echo ]", words)
	}
}

// Single quotes preserve literal value of EVERY character — from zsh manual
// and lex.c LX2_QUOTE case. Unlike fish, zsh does NOT support backslash
// escapes inside single quotes. Only ” is special (RC_QUOTES).
//
// shlex's zshFormat has NonEscapingQuoteBackslashEscapes() = false,
// meaning \ is NOT an escape character inside single quotes.
func TestVerify_ZshSingleQuoteNoBackslashEscape(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	if ZshFormat().NonEscapingQuoteBackslashEscapes() {
		t.Error("zsh NonEscapingQuoteBackslashEscapes() = true, want false (no \\ escapes in single quotes)")
	}

	// \ is literal inside single quotes
	tokens, err := SplitWith(`echo 'hello\nworld'`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("single quote literal \\: Words = %v, want [echo hello\\nworld]", words)
	}

	// $ is literal inside single quotes
	tokens, err = SplitWith(`echo '$HOME'`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `$HOME` {
		t.Errorf("single quote $ literal: Words = %v, want [echo $HOME]", words)
	}

	// ` is literal inside single quotes
	tokens, err = SplitWith("echo '`cmd`'", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "`cmd`" {
		t.Errorf("single quote backtick literal: Words = %v, want [echo `cmd`]", words)
	}
}

// Double-quote backslash escaping — from lex.c dquote_parse():
//
//	case '\\':
//	    c = hgetc();
//	    if (c != '\n') {
//	        if (c == '$' || c == '\\' || (c == '}' && !intick && bct) ||
//	            c == endchar || c == '`' || ...)
//	            add(Bnull);
//	        else {
//	            add('\\');
//	            goto cont;
//	        }
//	    } else if (sub || unset(CSHJUNKIEQUOTES) || endchar != '"')
//	        continue;
//	    break;
//
// Inside double quotes (endchar == '"'), \ escapes: $, \, ", `, and \n.
// \ before other chars is preserved literally (both \ and the char).
// This matches the CBSDQUOTE set, same as bash.
func TestVerify_ZshDoubleQuoteEscapeChars(t *testing.T) {
	expected := map[rune]bool{
		'\\': true,
		'`':  true,
		'$':  true,
		'"':  true,
		'\n': true,
	}
	got := ZshFormat().EscapingQuoteEscapeChars()
	for r, want := range expected {
		if got[r] != want {
			t.Errorf("EscapingQuoteEscapeChars[%q]=%v, want %v", r, got[r], want)
		}
	}
	// Characters NOT escapable inside zsh double quotes
	for _, r := range "abc123 \t" {
		if r == ' ' || r == '\t' {
			continue
		}
		if got[r] {
			t.Errorf("EscapingQuoteEscapeChars[%q]=true, want false (not in CBSDQUOTE)", r)
		}
	}
}

// Verify double quote escape behavior end-to-end
func TestVerify_ZshDoubleQuoteEscaping(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// \" inside double quotes → literal "
	tokens, err := SplitWith(`echo "say \"hello\""`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("zsh \\\" in double: Words = %v, want [echo say \"hello\"]", words)
	}

	// \$ inside double quotes → literal $
	tokens, err = SplitWith(`echo "cost: \$5"`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: $5` {
		t.Errorf("zsh \\$ in double: Words = %v, want [echo cost: $5]", words)
	}

	// \\ inside double quotes → literal \
	tokens, err = SplitWith(`echo "C:\\path"`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path` {
		t.Errorf("zsh \\\\ in double: Words = %v, want [echo C:\\path]", words)
	}

	// \` inside double quotes → literal `
	tokens, err = SplitWith("echo \"cmd \\`whoami\\`\"", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "cmd `whoami`" {
		t.Errorf("zsh \\` in double: Words = %v, want [echo cmd `whoami`]", words)
	}

	// \n inside double quotes → literal \n (NOT a newline)
	tokens, err = SplitWith(`echo "hello\nworld"`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("zsh \\n in double: Words = %v, want [echo hello\\nworld]", words)
	}

	// \t inside double quotes → literal \t (NOT a tab)
	tokens, err = SplitWith(`echo "a\tb"`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != `a\tb` {
		t.Errorf("zsh \\t in double: Words = %v, want [echo a\\tb]", words)
	}
}

// Line continuation: backslash + newline is removed — from lex.c gettok()
// LX1_BKSLASH case (outside quotes) and dquote_parse() (inside double quotes).
//
// Outside quotes: if (d == '\n') goto beginning; — discards both \ and \n.
// Inside double quotes: if (c != '\n') ... else continue; — line continuation.
func TestVerify_ZshLineContinuation(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Outside quotes: \ + \n → removed
	tokens, err := SplitWith("echo foo\\\nbar", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("zsh line continuation outside: words=%v, want [echo foobar]", words)
	}

	// Inside double quotes: \ + \n → removed
	tokens, err = SplitWith("echo \"foo\\\nbar\"", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("zsh line continuation in double: words=%v, want [echo foobar]", words)
	}

	// CRLF variant: \ + \r\n → removed
	tokens, err = SplitWith("echo foo\\\r\nbar", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("zsh line continuation CRLF: words=%v, want [echo foobar]", words)
	}
}

// Comment: # starts a comment at token boundary.
// From lex.c gettok(): hashchar handling before the switch.
// zsh only treats # as comment start when INTERACTIVECOMMENTS is set
// (interactive shell) or when not interactive. For completion context,
// shlex always treats # as a comment char, which matches the non-interactive
// behavior.
func TestVerify_ZshComment(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo hello # this is a comment", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello" {
		t.Errorf("zsh comment: words=%v, want [echo hello]", words)
	}
}

// Operator grammar — from lex.c tokstrings[] and gettok() operator recognition.
//
// zsh recognizes these operators (tokstrings array, lines 171-205):
//
//	;    → SEMI (sequential)
//	;;   → DSEMI (case terminator)
//	;&   → SEMIAMP (case fallthrough)
//	;|   → SEMIBAR (case fallthrough with retry)
//	&    → AMPER (background/async)
//	&&   → DAMPER (logical AND)
//	&|   → AMPERBANG (background with error check / disown)
//	|    → BAR (pipe)
//	||   → DBAR (logical OR)
//	|&   → BARAMP (pipe with stderr)
//	>    → OUTANG (redirect output)
//	>|   → OUTANGBANG (force redirect output, noclobber)
//	>>   → DOUTANG (append output)
//	>>|  → DOUTANGBANG (force append, noclobber)
//	<    → INANG (redirect input)
//	<>   → INOUTANG (input-output redirect)
//	<<   → DINANG (here-document)
//	<<-  → DINANGDASH (here-document with leading tabs)
//	<&   → INANGAMP (duplicate input fd)
//	>&   → OUTANGAMP (duplicate output fd)
//	&>   → AMPOUTANG (redirect stdout+stderr)
//	&>|  → OUTANGAMPBANG (redirect stdout+stderr, noclobber)
//	>>&  → DOUTANGAMP (append stdout+stderr)
//	>>&| → DOUTANGAMPBANG (append stdout+stderr, noclobber)
//	<<<  → TRINANG (here-string)
func TestVerify_ZshOperatorGrammar(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tests := []struct {
		input    string
		opType   WordbreakType
		opString string
	}{
		// List operators (from zsh.h lextok enum)
		{"|", WORDBREAK_PIPE, "|"},
		{"||", WORDBREAK_LIST_OR, "||"},
		{"&", WORDBREAK_LIST_ASYNC, "&"},
		{"&&", WORDBREAK_LIST_AND, "&&"},
		{";", WORDBREAK_LIST_SEQUENTIAL, ";"},
		{";;", WORDBREAK_LIST_SEQUENTIAL_DOUBLE, ";;"},
		{";&", WORDBREAK_LIST_FALLTHROUGH, ";&"},
		{";|", WORDBREAK_LIST_FALLTHROUGH_RETRY, ";|"},
		{"&|", WORDBREAK_LIST_ASYNC_ERRCHECK, "&|"},
		{"|&", WORDBREAK_PIPE_WITH_STDERR, "|&"},

		// Redirect operators (from zsh.h lextok + redirection enum)
		{">", WORDBREAK_REDIRECT_OUTPUT, ">"},
		{">>", WORDBREAK_REDIRECT_OUTPUT_APPEND, ">>"},
		{">|", WORDBREAK_REDIRECT_OUTPUT_FORCE, ">|"},
		{">>|", WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE, ">>|"},
		{"<", WORDBREAK_REDIRECT_INPUT, "<"},
		{"<<", WORDBREAK_REDIRECT_HERE_DOC, "<<"},
		{"<<<", WORDBREAK_REDIRECT_INPUT_STRING, "<<<"},
		{"<&", WORDBREAK_REDIRECT_INPUT_DUPLICATE, "<&"},
		{"<>", WORDBREAK_REDIRECT_INPUT_OUTPUT, "<>"},
		{">&", WORDBREAK_REDIRECT_OUTPUT_BOTH, ">&"},
		{"&>", WORDBREAK_REDIRECT_OUTPUT_BOTH, "&>"},
		{"&>>", WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND, "&>>"},
	}

	for _, tc := range tests {
		tokens, err := SplitWith("echo "+tc.input+" foo", ZshFormat())
		if err != nil {
			t.Fatalf("SplitWith(%q): %v", "echo "+tc.input+" foo", err)
		}
		found := false
		for _, tok := range tokens {
			if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == tc.opType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("operator %q: expected WordbreakType %v, not found in tokens", tc.input, tc.opString)
		}
	}
}

// zsh-specific operators not in bash:
//   - >>|  (DOUTANGBANG) — append with noclobber override
//   - ;&   (SEMIAMP) — case fallthrough
//   - ;|   (SEMIBAR) — case fallthrough with retry
//   - &|   (AMPERBANG) — background with error check (disown)
//
// These should be classified by zshFormat.ClassifyOperator, not bashWordbreakType.
func TestVerify_ZshSpecificOperators(t *testing.T) {
	tests := []struct {
		input  string
		opType WordbreakType
	}{
		{">>|", WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE},
		{";&", WORDBREAK_LIST_FALLTHROUGH},
		{";|", WORDBREAK_LIST_FALLTHROUGH_RETRY},
		{"&|", WORDBREAK_LIST_ASYNC_ERRCHECK},
	}
	for _, tc := range tests {
		got := ZshFormat().ClassifyOperator(tc.input)
		if got != tc.opType {
			t.Errorf("ClassifyOperator(%q) = %v, want %v", tc.input, got, tc.opType)
		}
	}
}

// Pipeline delimiters from zsh.h lextok enum + zsh execution model:
// |, ||, &, &&, ;, ;;, ;&, ;|, &|, |& are pipeline/list delimiters.
// These should all return IsPipelineDelimiter() = true.
//
// IS_REDIROP from zsh.h: (X) >= OUTANG && (X) <= TRINANG
// This covers >, >|, >>, >>|, <, <>, <<, <<-, <&, >&, &>, &>|, >>&, >>&|, <<<
// These should all return IsRedirect() = true (not pipeline delimiters).
func TestVerify_ZshPipelineDelimiters(t *testing.T) {
	pipelineOps := []string{
		"|", "||", "&", "&&", ";", ";;", ";&", ";|", "&|", "|&",
	}
	for _, op := range pipelineOps {
		wbType := ZshFormat().ClassifyOperator(op)
		if !wbType.IsPipelineDelimiter() {
			t.Errorf("zsh ClassifyOperator(%q).IsPipelineDelimiter() = false, want true", op)
		}
	}
}

// Redirect operators from zsh.h (IS_REDIROP macro covers OUTANG through TRINANG):
// >, >|, >>, >>|, <, <>, <<, <<<, <&, >&, &>, &>>
// These should all return IsRedirect() = true
func TestVerify_ZshRedirectOperators(t *testing.T) {
	redirectOps := []string{
		"<", ">", ">>", ">|", ">>|", "<<", "<<<", "<&", "<>",
		">&", "&>", "&>>",
	}
	for _, op := range redirectOps {
		wbType := ZshFormat().ClassifyOperator(op)
		if !wbType.IsRedirect() {
			t.Errorf("zsh ClassifyOperator(%q).IsRedirect() = false, want true", op)
		}
		if wbType.IsPipelineDelimiter() {
			t.Errorf("zsh ClassifyOperator(%q).IsPipelineDelimiter() = true, want false", op)
		}
	}
}

// zsh operators NOT yet handled by shlex — from lex.c tokstrings[].
// These zsh-specific operators are missing from ClassifyOperator:
//   - <<-  (DINANGDASH) — here-document with leading tab stripping
//   - &>|  (OUTANGAMPBANG) — redirect stdout+stderr with noclobber
//   - >>&  (DOUTANGAMP) — append redirect with stderr merge
//   - >>&| (DOUTANGAMPBANG) — append redirect with stderr merge, noclobber
//
// The tokenizer's greedy operator accumulation will split these into shorter
// recognized operators (e.g. &>| → &> + |, >>& → >> + &).
// This is a known limitation — the flat state machine can't distinguish these
// from the two-operator sequence without context.
func TestVerify_ZshMissingOperatorsKnownLimitation(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// These operators return WORDBREAK_UNKNOWN — they are not recognized
	// as single operators by shlex's zsh format.
	missingOps := []string{"<<-", "&>|", ">>&", ">>&|"}
	for _, op := range missingOps {
		got := ZshFormat().ClassifyOperator(op)
		if got != WORDBREAK_UNKNOWN {
			t.Errorf("ClassifyOperator(%q) = %v, want WORDBREAK_UNKNOWN (known limitation)", op, got)
		}
	}
}

// $ is NOT a wordbreak — from zsh.h SPECCHARS and lex.c lx2.
// $ is LX2_STRING (parameter expansion trigger), not LX2_BREAK.
// shlex correctly does not treat $ as a wordbreak.
func TestVerify_ZshDollarNotWordbreak(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	classifier := ZshFormat().Classifier()
	got := classifier.ClassifyRune('$')
	if got == wordbreakRuneClass {
		t.Error("$ should not be a wordbreak (it's LX2_STRING, not LX2_BREAK)")
	}
}

// Glob characters from zsh.h PATCHARS: "#^*()|[]<>?~\\"
// These are pattern matching characters, not word breaks.
// shlex correctly does not include them in wordbreaks.
func TestVerify_ZshGlobCharsNotWordbreak(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	classifier := ZshFormat().Classifier()
	for _, r := range "*?[]^" {
		got := classifier.ClassifyRune(r)
		if got == wordbreakRuneClass {
			t.Errorf("glob char %q should not be wordbreak", r)
		}
	}
}

// Command substitution: $(...) — from lex.c gettok() LX1_INPAR case
// and zsh.h INPAR/OUTPAR tokens.
// $ + ( should merge into WORDBREAK_SUBSTITUTION_OPEN with RawValue "$(".
func TestVerify_ZshCommandSubstitution(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo $(ls)", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	foundOpen := false
	foundClose := false
	for _, tok := range tokens {
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN && tok.Value == "$(" {
			foundOpen = true
		}
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_CLOSE {
			foundClose = true
		}
	}
	if !foundOpen {
		t.Error("zsh command substitution: $( not merged into WORDBREAK_SUBSTITUTION_OPEN")
	}
	if !foundClose {
		t.Error("zsh command substitution: ) not classified as WORDBREAK_SUBSTITUTION_CLOSE")
	}
}

// Arithmetic expansion: $((...)) — from lex.c gettok() LX1_INPAR case
// where ( after ( triggers DINPAR (arithmetic).
// $(( should merge into a single opener, )) into a single closer.
func TestVerify_ZshArithmeticExpansion(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo $((1+2))", ZshFormat())
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
		t.Error("zsh arithmetic: $(( not detected as arithmetic opener")
	}
	if !foundArithClose {
		t.Error("zsh arithmetic: )) not detected as arithmetic closer")
	}
}

// Process substitution: <( and >( — from lex.c gettok() LX1_INANG/LX1_OUTANG
// cases where ( after < or > triggers process substitution.
// These should be recognized as substitution openers, not plain redirects.
func TestVerify_ZshProcessSubstitution(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	for _, op := range []string{"<(", ">(", ")"} {
		input := "echo " + op
		if op == ")" {
			input = "echo <(cat) " + op
		}
		tokens, err := SplitWith(input, ZshFormat())
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

// Substitution should not split pipelines — pipes inside $(...) or <(...)
// should not split the outer pipeline.
func TestVerify_ZshSubstitutionDoesntSplitPipeline(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	tokens, err := SplitWith("echo $(ls | grep foo)", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("zsh subst pipeline: %d pipelines, want 1 (pipe inside $() shouldn't split)", len(pipelines))
	}
}

// fd prefix in redirects — from zsh gettok() idigit() handling.
// A numeric token immediately before a redirect operator is the fd prefix.
// FilterRedirects should filter it out.
func TestVerify_ZshFdPrefixInRedirects(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// 2> should filter out the "2" as fd prefix
	tokens, err := SplitWith("echo 2> file", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipeline := tokens.CurrentPipeline()
	filtered := pipeline.FilterRedirects()
	words := filtered.Words().Strings()

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

// QuoteWord for JoinWith — zsh uses posixQuoteWord (double-quote wrapping
// with \ escapes for CBSDQUOTE chars: \, `, $, ", \n).
// This matches zsh's quotestring() with QT_DOUBLE style.
func TestVerify_ZshQuoteWord(t *testing.T) {
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
		{`with` + "`" + `backtick`, `"with\` + "`" + `backtick"`},
		{`with
newline`, `"with\` + "\n" + `newline"`},
		// Empty string
		{"", `""`},
	}

	for _, tc := range tests {
		got := ZshFormat().QuoteWord(tc.input)
		if got != tc.want {
			t.Errorf("zsh QuoteWord(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// zsh escape outside quotes: backslash escapes the next character.
// From lex.c gettokstr() LX2_BKSLASH case: \ consumes the next char literally.
func TestVerify_ZshEscapeOutsideQuotes(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Escaped space → one word
	tokens, err := SplitWith(`echo a\ b`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("zsh escaped space: Words = %v, want [echo a b]", words)
	}

	// Escaped $ → literal $
	tokens, err = SplitWith(`echo \$HOME`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("zsh escaped $: Words = %v, want [echo $HOME]", words)
	}

	// Escaped pipe → literal | (not a pipeline delimiter)
	tokens, err = SplitWith(`echo foo\|bar`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo|bar" {
		t.Errorf("zsh escaped |: Words = %v, want [echo foo|bar]", words)
	}

	// Escaped > → literal > (not a redirect)
	tokens, err = SplitWith(`echo foo\>bar`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo>bar" {
		t.Errorf("zsh escaped >: Words = %v, want [echo foo>bar]", words)
	}
}

// zsh does NOT do word splitting on unquoted variable expansion by default
// (SHWORDSPLIT is off by default, unlike bash). This doesn't change the lexer
// (which splits on literal source whitespace), but verify the behavior
// is consistent: spaces inside double quotes don't split words.
func TestVerify_ZshNoWordSplitting(t *testing.T) {
	saved := os.Getenv("COMP_WORDBREAKS")
	defer os.Setenv("COMP_WORDBREAKS", saved)
	os.Unsetenv("COMP_WORDBREAKS")

	// Inside double quotes, spaces don't split
	tokens, err := SplitWith(`echo "hello world"`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("zsh no word split: Words = %v, want [echo hello world]", words)
	}

	// Outside quotes, spaces DO split (literal source whitespace)
	tokens, err = SplitWith("echo hello world", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words = tokens.Words().Strings()
	if len(words) != 3 {
		t.Errorf("zsh word split: Words = %v, want 3 words", words)
	}
}
