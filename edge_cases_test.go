package shlex

import "testing"

// Edge case tests derived from the format reference docs.

// --- Bash edge cases ---

func TestBashEdge_CloseQuoteEscapeReopen(t *testing.T) {
	// The POSIX idiom for embedding a single quote: 'it'\''s
	tokens, err := SplitWith("echo 'it'\\''s", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's" {
		t.Errorf("bash '\\'': Words = %v, want [echo it's]", words)
	}
}

func TestBashEdge_EscapedSpace(t *testing.T) {
	tokens, err := SplitWith(`echo a\ b`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("bash escaped space: Words = %v, want [echo a b]", words)
	}
}

func TestBashEdge_BackslashNInDoubleQuotes(t *testing.T) {
	// In bash, \n inside "..." is literal (backslash not special before n)
	// The state machine consumes \ + next char, so Value = "hellonworld"
	// This is a known limitation — the lexer is not a full expander.
	tokens, err := SplitWith(`echo "hello\nworld"`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != IN_WORD_STATE {
		t.Errorf("bash \\n in double: State = %v, want IN_WORD_STATE", last.State)
	}
}

func TestBashEdge_AdjacentQuotedSegments(t *testing.T) {
	tokens, err := SplitWith(`echo a"b"'c'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "abc" {
		t.Errorf("bash adjacent: Words = %v, want [echo abc]", words)
	}
}

func TestBashEdge_SingleQuoteLiteral(t *testing.T) {
	tokens, err := SplitWith(`echo '$HOME \n \t'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `$HOME \n \t` {
		t.Errorf("bash single literal: Words = %v, want [echo $HOME \\n \\t]", words)
	}
}

func TestBashEdge_AtWordbreakPrefix(t *testing.T) {
	// @ is a wordbreak but WordbreakPrefix skips it
	ctx := SplitForCompletion("echo foo@bar", BashFormat())
	// @ is a wordbreak, but Words() merges adjoining tokens, so CurrentWord is the full word
	if ctx.CurrentWord != "foo@bar" {
		t.Errorf("bash @: CurrentWord = %q, want %q", ctx.CurrentWord, "foo@bar")
	}
	// @ is skipped as a wordbreak boundary, so prefix should be "foo"
	if ctx.Prefix != "foo" {
		t.Errorf("bash @: Prefix = %q, want %q", ctx.Prefix, "foo")
	}
}

func TestBashEdge_EscapeAtEOF(t *testing.T) {
	tokens, err := SplitWith(`echo foo\`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != ESCAPING_STATE {
		t.Errorf("bash escape EOF: State = %v, want ESCAPING_STATE", last.State)
	}
	if last.Value != "foo" {
		t.Errorf("bash escape EOF: Value = %q, want %q", last.Value, "foo")
	}
}

func TestBashEdge_Comment(t *testing.T) {
	tokens, err := SplitWith("echo hello # comment", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	// Lexer skips comments, so only "echo" and "hello" are returned
	words := tokens.Words().Strings()
	if len(words) != 2 {
		t.Errorf("bash comment: Words = %v, want 2 words", words)
	}
}

// --- Zsh edge cases ---

func TestZshEdge_DoubleQuoteEscape(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("zsh double escape: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestZshEdge_RCQuotesLonger(t *testing.T) {
	tokens, err := SplitWith("echo 'it''s a test'", ZshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's a test" {
		t.Errorf("zsh RC_QUOTES longer: Words = %v, want [echo it's a test]", words)
	}
}

// --- Fish edge cases ---

func TestFishEdge_DollarNotEscapeInSingleQuotes(t *testing.T) {
	// \$ is NOT an escape in fish single quotes — only \' and \\ are
	tokens, err := SplitWith(`echo 'cost: \$5'`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: \$5` {
		t.Errorf("fish \\$ in single: Words = %v, want [echo cost: \\$5]", words)
	}
}

func TestFishEdge_EscapedSpace(t *testing.T) {
	tokens, err := SplitWith(`echo a\ b`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("fish escaped space: Words = %v, want [echo a b]", words)
	}
}

func TestFishEdge_ParensNotWordbreak(t *testing.T) {
	// Fish: () are command substitution, not word breaks.
	// Spaces still split words, but parens are part of the words.
	tokens, err := SplitWith("echo (echo test)", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "(echo" || words[2] != "test)" {
		t.Errorf("fish parens: Words = %v, want [echo (echo test)]", words)
	}
}

// --- Elvish edge cases ---

func TestElvishEdge_DoubledQuoteSplit(t *testing.T) {
	tokens, err := SplitWith("echo 'it''s a test'", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's a test" {
		t.Errorf("elvish '' split: Words = %v, want [echo it's a test]", words)
	}
}

func TestElvishEdge_DoubleQuoteValue(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("elvish double value: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestElvishEdge_AmpNotListOperator(t *testing.T) {
	// & is for map literals in elvish, not a list operator
	tokens, err := SplitWith("echo foo & echo bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 1 {
		t.Errorf("elvish &: %d pipelines, want 1 (& is not a separator)", len(pipelines))
	}
}

func TestElvishEdge_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("elvish pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestElvishEdge_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("elvish semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestElvishEdge_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("elvish open single: State = %v, want QUOTING_STATE", last.State)
	}
}

// --- Nushell edge cases ---

func TestNushellEdge_InterpolatedDouble(t *testing.T) {
	tokens, err := SplitWith(`echo $"hello"`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$hello" {
		t.Errorf("nushell $\": Words = %v, want [echo $hello]", words)
	}
}

func TestNushellEdge_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("nushell semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestNushellEdge_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("nushell open single: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestNushellEdge_OpenDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hel`, NushellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("nushell open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}

// --- PowerShell edge cases ---

func TestPowershellEdge_DoubledSingleQuoteSplit(t *testing.T) {
	tokens, err := SplitWith("echo 'don''t'", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "don't" {
		t.Errorf("powershell '' split: Words = %v, want [echo don't]", words)
	}
}

func TestPowershellEdge_BacktickEscapeOutside(t *testing.T) {
	tokens, err := SplitWith("echo `$HOME", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("powershell backtick outside: Words = %v, want [echo $HOME]", words)
	}
}

func TestPowershellEdge_BacktickInDoubleQuotes(t *testing.T) {
	tokens, err := SplitWith("echo \"say `\"hello`\"\"", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("powershell backtick in double: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestPowershellEdge_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("powershell pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestPowershellEdge_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("powershell semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestPowershellEdge_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("powershell &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestPowershellEdge_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("powershell open single: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestPowershellEdge_OpenDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hel`, PowershellFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("powershell open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}

// --- Xonsh edge cases ---

func TestXonshEdge_RawDoubleQuoted(t *testing.T) {
	// r"..." — r prefix merges with double-quoted segment.
	// Note: \ inside double quotes is consumed as escape (ESCAPING_QUOTED_STATE).
	// The r prefix is a word char and doesn't change quote behavior in the lexer.
	tokens, err := SplitWith(`echo r"C:\path"`, XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	// \p is consumed as escape → "p" emitted, backslash dropped
	if len(words) != 2 || words[1] != "rC:path" {
		t.Errorf("xonsh r\"\": Words = %v, want [echo rC:path]", words)
	}
}

func TestXonshEdge_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshEdge_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshEdge_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("xonsh semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestXonshEdge_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", XonshFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("xonsh open single: State = %v, want QUOTING_STATE", last.State)
	}
}

// --- Tcsh edge cases ---

func TestTcshEdge_SingleQuoteLiteral(t *testing.T) {
	tokens, err := SplitWith("echo '$HOME'", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "$HOME" {
		t.Errorf("tcsh single literal: Words = %v, want [echo $HOME]", words)
	}
}

func TestTcshEdge_BacktickLiteralInSingleQuotes(t *testing.T) {
	tokens, err := SplitWith("echo '`cmd`'", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "`cmd`" {
		t.Errorf("tcsh backtick in single: Words = %v, want [echo `cmd`]", words)
	}
}

func TestTcshEdge_EscapedDoubleQuoteOutside(t *testing.T) {
	tokens, err := SplitWith(`echo \"hello\"`, TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `"hello"` {
		t.Errorf("tcsh escaped double: Words = %v, want [echo \"hello\"]", words)
	}
}

func TestTcshEdge_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshEdge_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshEdge_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("tcsh semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestTcshEdge_Background(t *testing.T) {
	tokens, err := SplitWith("echo foo &", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	var amp Token
	for _, tok := range tokens {
		if tok.RawValue == "&" {
			amp = tok
		}
	}
	if amp.Type != WORDBREAK_TOKEN || amp.WordbreakType != WORDBREAK_LIST_ASYNC {
		t.Errorf("tcsh &: Type=%v WT=%v, want WORDBREAK_TOKEN/LIST_ASYNC", amp.Type, amp.WordbreakType)
	}
}

func TestTcshEdge_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", TcshFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("tcsh open single: State = %v, want QUOTING_STATE", last.State)
	}
}

// --- Oil edge cases ---

func TestOilEdge_SingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hello world'", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello world" {
		t.Errorf("oil single: Words = %v, want [echo hello world]", words)
	}
}

func TestOilEdge_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilEdge_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestOilEdge_DoubleAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", OilFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("oil &&: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

// --- Cmd edge cases ---

func TestCmdEdge_DoubleOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("cmd ||: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestCmdEdge_Redirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo > bar", CmdFormat())
	if !ctx.IsRedirect {
		t.Errorf("cmd redirect: IsRedirect = false, want true")
	}
	if ctx.CurrentWord != "bar" {
		t.Errorf("cmd redirect: CurrentWord = %q, want %q", ctx.CurrentWord, "bar")
	}
}

func TestCmdEdge_OpenDoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hel`, CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_ESCAPING_STATE {
		t.Errorf("cmd open double: State = %v, want QUOTING_ESCAPING_STATE", last.State)
	}
}

func TestCmdEdge_CaretAtEOF(t *testing.T) {
	tokens, err := SplitWith("echo foo^", CmdFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != ESCAPING_STATE {
		t.Errorf("cmd caret EOF: State = %v, want ESCAPING_STATE", last.State)
	}
	if last.Value != "foo" {
		t.Errorf("cmd caret EOF: Value = %q, want %q", last.Value, "foo")
	}
}
