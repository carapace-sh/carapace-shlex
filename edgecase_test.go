package shlex

import (
	"testing"
)

// Stop-parsing mode edge cases (PowerShell --%).

func TestEdgeCase_StopParsingEOF(t *testing.T) {
	// EOF immediately after --% with no content — empty word merges with --%
	tokens, err := Split("echo --%", Powershell)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "--%" {
		t.Errorf("stop-parsing EOF: Words = %v, want [echo --%%]", words)
	}
}

func TestEdgeCase_StopParsingNewlineOnly(t *testing.T) {
	// Newline after --% — newline is whitespace, next line continues in same pipeline
	tokens, err := Split("echo --%\nfoo", Powershell)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "--%" || words[2] != "foo" {
		t.Errorf("stop-parsing newline: Words = %v, want [echo --%% foo]", words)
	}
	if len(tokens.pipelines()) != 1 {
		t.Errorf("stop-parsing newline: %d pipelines, want 1", len(tokens.pipelines()))
	}
}

func TestEdgeCase_StopParsingPipeInDoubleQuotes(t *testing.T) {
	// After --%, | inside double quotes should NOT split the pipeline
	tokens, err := Split(`echo --% "foo | bar" baz`, Powershell)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.pipelines()) != 1 {
		t.Errorf("stop-parsing pipe in quotes: %d pipelines, want 1", len(tokens.pipelines()))
	}
}

func TestEdgeCase_StopParsingDoubleQuoteToggling(t *testing.T) {
	// After --%, double quotes toggle in-quotes state — raw content is one word
	tokens, err := Split(`echo --% "hello" world`, Powershell)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 3 || words[0] != "echo" || words[1] != "--%" {
		t.Errorf("stop-parsing quote toggling: Words = %v, want 3 words", words)
	}
	if words[2] != `"hello" world` {
		t.Errorf("stop-parsing quote toggling: raw = %q, want %q", words[2], `"hello" world`)
	}
}

// Block comment edge cases (PowerShell <# ... #>).

func TestEdgeCase_BlockCommentUnclosedEOF(t *testing.T) {
	tokens, err := Split("echo <# unclosed comment", Powershell)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 1 || words[0] != "echo" {
		t.Errorf("unclosed block comment: Words = %v, want [echo]", words)
	}
}

func TestEdgeCase_BlockCommentHashInside(t *testing.T) {
	// A lone # inside a block comment does NOT close it — only #> closes
	tokens, err := Split("echo <# has # and # inside #> foo", Powershell)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("block comment # inside: Words = %v, want [echo foo]", words)
	}
}

func TestEdgeCase_BlockCommentCloserRestart(t *testing.T) {
	// The closer #> starts with # — a lone # restarts the match but doesn't close
	tokens, err := Split("echo <# # not closer #> foo", Powershell)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("block comment closer restart: Words = %v, want [echo foo]", words)
	}
}

func TestEdgeCase_BlockCommentAfterWordbreak(t *testing.T) {
	// Block comment opener after a pipe — pipe splits, comment consumed
	tokens, err := Split("echo foo |<# comment #> bar", Powershell)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.pipelines()) != 2 {
		t.Errorf("block comment after wordbreak: %d pipelines, want 2", len(tokens.pipelines()))
	}
}

// Unicode / multi-byte character tests.

func TestEdgeCase_UnicodeCJK(t *testing.T) {
	// CJK characters — rune offsets must be correct (3 runes, not 9 bytes)
	tokens, err := Split("echo 日本語", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "日本語" {
		t.Errorf("unicode CJK: Words = %v, want [echo 日本語]", words)
	}
	// Span should be rune-based: "echo " is 5 runes, "日本語" is 3 runes
	if len(tokens) < 2 {
		t.Fatal("expected at least 2 tokens")
	}
	if tokens[1].Span.Start != 5 || tokens[1].Span.End != 8 {
		t.Errorf("unicode CJK: Span = %v, want {5 8}", tokens[1].Span)
	}
}

func TestEdgeCase_UnicodeAccented(t *testing.T) {
	// Accented character (é) is a single rune
	tokens, err := Split("echo café", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "café" {
		t.Errorf("unicode accented: Words = %v, want [echo café]", words)
	}
	if len(tokens) < 2 {
		t.Fatal("expected at least 2 tokens")
	}
	if tokens[1].Span.Start != 5 || tokens[1].Span.End != 9 {
		t.Errorf("unicode accented: Span = %v, want {5 9}", tokens[1].Span)
	}
}

func TestEdgeCase_UnicodeInQuotes(t *testing.T) {
	tokens, err := Split("echo \"日本語\"", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "日本語" {
		t.Errorf("unicode in quotes: Words = %v, want [echo 日本語]", words)
	}
}

func TestEdgeCase_UnicodeEmoji(t *testing.T) {
	// Emoji (4-byte UTF-8) — single rune
	tokens, err := Split("echo 🎉", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "🎉" {
		t.Errorf("unicode emoji: Words = %v, want [echo 🎉]", words)
	}
}

func TestEdgeCase_UnicodeElvishBareword(t *testing.T) {
	// Elvish allows non-ASCII printable in barewords
	tokens, err := Split("echo naïve", Elvish)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "naïve" {
		t.Errorf("unicode elvish bareword: Words = %v, want [echo naïve]", words)
	}
}

// Triple-quote opening "consumed" path (xonsh).
// When checkTripleQuote peeks two runes and the first matches but the
// second doesn't (or EOF after the first), the first is returned as
// consumedRune. This is distinct from checkTripleClose which IS tested.

func TestEdgeCase_TripleQuoteOpeningTwoQuotesThenWord(t *testing.T) {
	// Two double quotes followed by a word — not a triple quote
	tokens, err := Split(`echo ""hello`, Xonsh)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello" {
		t.Errorf("triple quote two quotes then word: Words = %v, want [echo hello]", words)
	}
}

func TestEdgeCase_TripleQuoteOpeningTwoQuotesEOF(t *testing.T) {
	// Two double quotes at EOF — not a triple quote, empty string
	tokens, err := Split(`echo ""`, Xonsh)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "" {
		t.Errorf("triple quote two quotes EOF: Words = %v, want [echo ]", words)
	}
}

func TestEdgeCase_TripleQuoteOpeningTwoSingleQuotesThenWord(t *testing.T) {
	// Two single quotes followed by a word — not a triple quote
	tokens, err := Split(`echo ''hello`, Xonsh)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello" {
		t.Errorf("triple quote two single then word: Words = %v, want [echo hello]", words)
	}
}

// FilterRedirects fd prefix edge cases.

func TestEdgeCase_FilterRedirectsFd3(t *testing.T) {
	tokens, err := Split("echo 3> file", Bash)
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 1 || words[0] != "echo" {
		t.Errorf("fd 3> filter: Words = %v, want [echo]", words)
	}
}

func TestEdgeCase_FilterRedirectsFd10(t *testing.T) {
	// Multi-digit fd prefix should also be filtered
	tokens, err := Split("echo 10> file", Bash)
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 1 || words[0] != "echo" {
		t.Errorf("fd 10> filter: Words = %v, want [echo]", words)
	}
}

func TestEdgeCase_FilterRedirectsNonNumericNotFiltered(t *testing.T) {
	// Non-numeric word before redirect should NOT be filtered as fd prefix
	tokens, err := Split("echo foo> file", Bash)
	if err != nil {
		t.Fatal(err)
	}
	filtered := tokens.CurrentPipeline().FilterRedirects()
	words := filtered.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("non-numeric before redirect: Words = %v, want [echo foo]", words)
	}
}

// WordbreakPrefix with multiple wordbreaks.

func TestEdgeCase_WordbreakPrefixMultipleEquals(t *testing.T) {
	ctx := Complete("echo foo=bar=baz", Bash)
	if ctx.Prefix != "foo=bar=" {
		t.Errorf("multiple = prefix: Prefix = %q, want %q", ctx.Prefix, "foo=bar=")
	}
}

func TestEdgeCase_WordbreakPrefixMultipleAt(t *testing.T) {
	ctx := Complete("echo foo@bar@baz", Bash)
	if ctx.Prefix != "foo@bar" {
		t.Errorf("multiple @ prefix: Prefix = %q, want %q", ctx.Prefix, "foo@bar")
	}
}

func TestEdgeCase_WordbreakPrefixMixedEqualsAt(t *testing.T) {
	ctx := Complete("echo foo=bar@baz", Bash)
	if ctx.Prefix != "foo=bar" {
		t.Errorf("mixed = @ prefix: Prefix = %q, want %q", ctx.Prefix, "foo=bar")
	}
}

// Whitespace-only input.

func TestEdgeCase_WhitespaceOnly(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"spaces only", "   "},
		{"tabs only", "\t\t"},
		{"mixed whitespace", " \t \n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Split(tt.input, Bash)
			if err != nil {
				t.Fatalf("Split error: %v", err)
			}
			// Should produce exactly one empty token
			if len(tokens) != 1 {
				t.Fatalf("tokens count = %d, want 1", len(tokens))
			}
			if tokens[0].Value != "" {
				t.Errorf("token Value = %q, want empty", tokens[0].Value)
			}
			// Completion context: one empty word (the current cursor position)
			ctx := Complete(tt.input, Bash)
			if ctx.QuotingState != START_STATE {
				t.Errorf("QuotingState = %v, want START_STATE", ctx.QuotingState)
			}
			if len(ctx.Words) != 1 || ctx.Words[0] != "" {
				t.Errorf("Words = %v, want [\"]", ctx.Words)
			}
		})
	}
}

// Pipeline with no space after delimiter.

func TestEdgeCase_PipeNoSpaceAfter(t *testing.T) {
	tokens, err := Split("echo foo|grep", Bash)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.pipelines()) != 2 {
		t.Errorf("pipe no space: %d pipelines, want 2", len(tokens.pipelines()))
	}
}

func TestEdgeCase_PipeNoSpaceCompletion(t *testing.T) {
	ctx := Complete("echo foo|grep bar", Bash)
	if len(ctx.Words) != 2 || ctx.Words[0] != "grep" || ctx.Words[1] != "bar" {
		t.Errorf("pipe no space completion: Words = %v, want [grep bar]", ctx.Words)
	}
}

func TestEdgeCase_PipeEOFCompletion(t *testing.T) {
	// After a pipe at EOF, the cursor is in a new (empty) pipeline
	ctx := Complete("echo foo |", Bash)
	if ctx.CurrentWord != "" {
		t.Errorf("pipe EOF completion: CurrentWord = %q, want empty", ctx.CurrentWord)
	}
	if ctx.QuotingState != START_STATE {
		t.Errorf("pipe EOF completion: QuotingState = %v, want START_STATE", ctx.QuotingState)
	}
}

// Comment edge cases.

func TestEdgeCase_CommentAtEOF(t *testing.T) {
	tokens, err := Split("echo foo # comment", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("comment at EOF: Words = %v, want [echo foo]", words)
	}
}

func TestEdgeCase_CommentThenNewline(t *testing.T) {
	tokens, err := Split("echo foo # comment\necho bar", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 4 || words[0] != "echo" || words[1] != "foo" || words[2] != "echo" || words[3] != "bar" {
		t.Errorf("comment then newline: Words = %v, want [echo foo echo bar]", words)
	}
}

func TestEdgeCase_CommentAtStart(t *testing.T) {
	tokens, err := Split("# comment\necho foo", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "foo" {
		t.Errorf("comment at start: Words = %v, want [echo foo]", words)
	}
}

func TestEdgeCase_HashInDoubleQuotesNotComment(t *testing.T) {
	tokens, err := Split(`echo "foo#bar"`, Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo#bar" {
		t.Errorf("hash in quotes: Words = %v, want [echo foo#bar]", words)
	}
}

// Line continuation edge cases.

func TestEdgeCase_LineContinuationAtEOF(t *testing.T) {
	// Dangling escape at EOF — escape char is dropped, word returned without it
	tokens, err := Split("echo foo\\", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foo" {
		t.Errorf("line continuation at EOF: Words = %v, want [echo foo]", words)
	}
}

func TestEdgeCase_LineContinuationJoins(t *testing.T) {
	tokens, err := Split("echo foo\\\nbar", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "foobar" {
		t.Errorf("line continuation joins: Words = %v, want [echo foobar]", words)
	}
}

func TestEdgeCase_LineContinuationInSingleQuotesLiteral(t *testing.T) {
	// Backslash-newline inside single quotes is literal (no line continuation)
	tokens, err := Split("echo 'foo\\\nbar'", Bash)
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 {
		t.Errorf("line continuation in single quotes: Words = %v, want 2 words", words)
	}
	if words[1] != "foo\\\nbar" {
		t.Errorf("line continuation in single quotes: word = %q, want %q", words[1], "foo\\\nbar")
	}
}

// Empty token after wordbreak at EOF.

func TestEdgeCase_EmptyTokenAfterPipeEOF(t *testing.T) {
	tokens, err := Split("echo foo |", Bash)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d", len(tokens))
	}
	last := tokens[len(tokens)-1]
	if last.Value != "" {
		t.Errorf("last token Value = %q, want empty", last.Value)
	}
}

func TestEdgeCase_EmptyTokenAfterRedirectEOF(t *testing.T) {
	tokens, err := Split("echo foo >", Bash)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d", len(tokens))
	}
	last := tokens[len(tokens)-1]
	if last.Value != "" {
		t.Errorf("last token Value = %q, want empty", last.Value)
	}
}

func TestEdgeCase_EmptyTokenAfterCaseTerminatorEOF(t *testing.T) {
	tokens, err := Split("echo foo ;;", Bash)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d", len(tokens))
	}
	last := tokens[len(tokens)-1]
	if last.Value != "" {
		t.Errorf("last token Value = %q, want empty", last.Value)
	}
}

// Nushell wordbreak type verification for ( and ).

func TestEdgeCase_NushellSubstitutionWordbreakType(t *testing.T) {
	tokens, err := Split("echo (ls)", Nushell)
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == "(" {
			if tok.WordbreakType != WORDBREAK_SUBSTITUTION_OPEN {
				t.Errorf("nushell ( : WordbreakType = %v, want WORDBREAK_SUBSTITUTION_OPEN", tok.WordbreakType)
			}
		}
		if tok.Type == WORDBREAK_TOKEN && tok.RawValue == ")" {
			if tok.WordbreakType != WORDBREAK_SUBSTITUTION_CLOSE {
				t.Errorf("nushell ) : WordbreakType = %v, want WORDBREAK_SUBSTITUTION_CLOSE", tok.WordbreakType)
			}
		}
	}
}

// posixQuoteWord edge cases.

func TestEdgeCase_PosixQuoteWordTab(t *testing.T) {
	// Tab is emitted literally inside double quotes
	got := posixQuoteWord("hello\tworld")
	want := `"hello	world"`
	if got != want {
		t.Errorf("posixQuoteWord tab: got %q, want %q", got, want)
	}
}

func TestEdgeCase_PosixQuoteWordCR(t *testing.T) {
	// CR is emitted literally inside double quotes
	got := "echo " + posixQuoteWord("hello\rworld")
	want := `echo "hello` + "\r" + `world"`
	if got != want {
		t.Errorf("posixQuoteWord CR: got %q, want %q", got, want)
	}
}

func TestEdgeCase_PosixQuoteWordBacktick(t *testing.T) {
	got := posixQuoteWord("hello`world")
	expected := `"hello\` + "`" + `world"`
	if got != expected {
		t.Errorf("posixQuoteWord backtick: got %q, want %q", got, expected)
	}
}

// cmdQuoteWord edge cases.

func TestEdgeCase_CmdQuoteWordCaret(t *testing.T) {
	// Caret triggers quoting but is literal inside double quotes
	got := cmdQuoteWord("hello^world")
	want := `"hello^world"`
	if got != want {
		t.Errorf("cmdQuoteWord caret: got %q, want %q", got, want)
	}
}

func TestEdgeCase_CmdQuoteWordMultipleQuotes(t *testing.T) {
	got := cmdQuoteWord(`a"b"c`)
	want := `"a"^"b"^"c"`
	if got != want {
		t.Errorf("cmdQuoteWord multiple quotes: got %q, want %q", got, want)
	}
}
