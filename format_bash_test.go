package shlex

import "testing"

func TestBashFormat_Classifier(t *testing.T) {
	classifier := BashFormat().Classifier()
	tests := map[rune]runeTokenClass{
		' ':  spaceRuneClass,
		'"':  escapingQuoteRuneClass,
		'\'': nonEscapingQuoteRuneClass,
		'#':  commentRuneClass,
	}
	for runeChar, want := range tests {
		got := classifier.ClassifyRune(runeChar)
		if got != want {
			t.Errorf("ClassifyRune(%v) -> %v. Want: %v", runeChar, got, want)
		}
	}
}

func TestBashFormat_CloseQuoteEscapeReopen(t *testing.T) {
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

func TestBashFormat_EscapedSpace(t *testing.T) {
	tokens, err := SplitWith(`echo a\ b`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("bash escaped space: Words = %v, want [echo a b]", words)
	}
}

func TestBashFormat_BackslashNInDoubleQuotes(t *testing.T) {
	// In bash, \n inside "..." is literal (backslash not special before n).
	// Backslash is only special before $, `, ", \, and newline.
	// Both backslash and n should be preserved in the value.
	tokens, err := SplitWith(`echo "hello\nworld"`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != IN_WORD_STATE {
		t.Errorf("bash \\n in double: State = %v, want IN_WORD_STATE", last.State)
	}
	if last.Value != `hello\nworld` {
		t.Errorf("bash \\n in double: Value = %q, want %q", last.Value, `hello\nworld`)
	}
}

func TestBashFormat_AdjacentQuotedSegments(t *testing.T) {
	tokens, err := SplitWith(`echo a"b"'c'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "abc" {
		t.Errorf("bash adjacent: Words = %v, want [echo abc]", words)
	}
}

func TestBashFormat_SingleQuoteLiteral(t *testing.T) {
	tokens, err := SplitWith(`echo '$HOME \n \t'`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `$HOME \n \t` {
		t.Errorf("bash single literal: Words = %v, want [echo $HOME \\n \\t]", words)
	}
}

func TestBashFormat_AtWordbreakPrefix(t *testing.T) {
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

func TestBashFormat_EscapeAtEOF(t *testing.T) {
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

func TestBashFormat_Comment(t *testing.T) {
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

func TestBashFormat_ForceOutputRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo >| bar", BashFormat())
	if !ctx.IsRedirect {
		t.Errorf("bash >|: IsRedirect = false, want true")
	}
}

func TestBashFormat_CaseTerminator(t *testing.T) {
	tokens, err := SplitWith("echo foo ;; bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("bash ;;: Pipelines = %d, want 2", len(pipelines))
	}
	var found *Token
	for i := range tokens {
		if tokens[i].Type == WORDBREAK_TOKEN {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		t.Fatal("bash ;;: no wordbreak token found")
	}
	if found.WordbreakType != WORDBREAK_LIST_SEQUENTIAL_DOUBLE {
		t.Errorf("bash ;;: WordbreakType = %v, want WORDBREAK_LIST_SEQUENTIAL_DOUBLE", found.WordbreakType)
	}
}

func TestBashFormat_BackslashDollarInDoubleQuotes(t *testing.T) {
	// \$ inside "..." should drop the backslash — $ is a CBSDQUOTE char
	tokens, err := SplitWith(`echo "hello\$world"`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello$world" {
		t.Errorf("bash \\$ in double: Words = %v, want [echo hello$world]", words)
	}
}

func TestBashFormat_BackslashQuoteInDoubleQuotes(t *testing.T) {
	// \" inside "..." should drop the backslash — " is a CBSDQUOTE char
	tokens, err := SplitWith(`echo "say \"hello\""`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("bash \\\" in double: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestBashFormat_BackslashBacktickInDoubleQuotes(t *testing.T) {
	// \` inside "..." should drop the backslash — ` is a CBSDQUOTE char
	tokens, err := SplitWith("echo \"hello\\`world\"", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello`world" {
		t.Errorf("bash \\` in double: Words = %v, want [echo hello`world]", words)
	}
}

func TestBashFormat_BackslashBackslashInDoubleQuotes(t *testing.T) {
	// \\ inside "..." should produce a single \ — \ is a CBSDQUOTE char
	tokens, err := SplitWith(`echo "hello\\world"`, BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\world` {
		t.Errorf("bash \\\\ in double: Words = %v, want [echo hello\\world]", words)
	}
}

func TestBashFormat_OperatorBoundary(t *testing.T) {
	// >; should be two separate tokens: > and ;
	// not a single unknown operator >;
	tokens, err := SplitWith("echo foo >; bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var wordbreaks []Token
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN {
			wordbreaks = append(wordbreaks, tok)
		}
	}
	if len(wordbreaks) != 2 {
		t.Fatalf("bash >;: got %d wordbreak tokens, want 2", len(wordbreaks))
	}
	if wordbreaks[0].RawValue != ">" {
		t.Errorf("bash >;: first wordbreak = %q, want >", wordbreaks[0].RawValue)
	}
	if wordbreaks[1].RawValue != ";" {
		t.Errorf("bash >;: second wordbreak = %q, want ;", wordbreaks[1].RawValue)
	}
}

func TestBashFormat_HereDocOperator(t *testing.T) {
	tokens, err := SplitWith("echo foo << bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found *Token
	for i := range tokens {
		if tokens[i].Type == WORDBREAK_TOKEN {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		t.Fatal("bash <<: no wordbreak token found")
	}
	if found.WordbreakType != WORDBREAK_REDIRECT_HERE_DOC {
		t.Errorf("bash <<: WordbreakType = %v, want WORDBREAK_REDIRECT_HERE_DOC", found.WordbreakType)
	}
	if !found.WordbreakType.IsRedirect() {
		t.Errorf("bash <<: IsRedirect() = false, want true")
	}
}

func TestBashFormat_HereStringOperator(t *testing.T) {
	tokens, err := SplitWith("echo foo <<< bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found *Token
	for i := range tokens {
		if tokens[i].Type == WORDBREAK_TOKEN {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		t.Fatal("bash <<<: no wordbreak token found")
	}
	if found.WordbreakType != WORDBREAK_REDIRECT_INPUT_STRING {
		t.Errorf("bash <<<: WordbreakType = %v, want WORDBREAK_REDIRECT_INPUT_STRING", found.WordbreakType)
	}
}

func TestBashFormat_CaseFallthrough(t *testing.T) {
	// ;& is bash 4+ case fall-through
	tokens, err := SplitWith("echo foo ;& bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found *Token
	for i := range tokens {
		if tokens[i].Type == WORDBREAK_TOKEN {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		t.Fatal("bash ;&: no wordbreak token found")
	}
	if found.WordbreakType != WORDBREAK_LIST_FALLTHROUGH {
		t.Errorf("bash ;&: WordbreakType = %v, want WORDBREAK_LIST_FALLTHROUGH", found.WordbreakType)
	}
}

func TestBashFormat_CaseNextPattern(t *testing.T) {
	// ;;& is bash 4+ case next pattern
	tokens, err := SplitWith("echo foo ;;& bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found *Token
	for i := range tokens {
		if tokens[i].Type == WORDBREAK_TOKEN {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		t.Fatal("bash ;;&: no wordbreak token found")
	}
	if found.WordbreakType != WORDBREAK_LIST_CASE_NEXT {
		t.Errorf("bash ;;&: WordbreakType = %v, want WORDBREAK_LIST_CASE_NEXT", found.WordbreakType)
	}
	if !found.WordbreakType.IsPipelineDelimiter() {
		t.Errorf("bash ;;&: IsPipelineDelimiter() = false, want true")
	}
}

func TestBashFormat_CloseParenAsWordbreak(t *testing.T) {
	// ) is a shell break character in bash (shell_break_chars includes it)
	tokens, err := SplitWith("echo (foo) bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var wordbreaks []string
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN {
			wordbreaks = append(wordbreaks, tok.RawValue)
		}
	}
	if len(wordbreaks) < 2 {
		t.Fatalf("bash (foo): got %d wordbreak tokens, want at least 2", len(wordbreaks))
	}
	hasOpen := false
	hasClose := false
	for _, wb := range wordbreaks {
		if wb == "(" {
			hasOpen = true
		}
		if wb == ")" {
			hasClose = true
		}
	}
	if !hasOpen {
		t.Errorf("bash (foo): no ( wordbreak found in %v", wordbreaks)
	}
	if !hasClose {
		t.Errorf("bash (foo): no ) wordbreak found in %v", wordbreaks)
	}
}

func TestBashFormat_OperatorSequencePipeSemicolon(t *testing.T) {
	// |; should be two separate tokens: | and ;
	tokens, err := SplitWith("echo foo |; bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var wordbreaks []Token
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN {
			wordbreaks = append(wordbreaks, tok)
		}
	}
	if len(wordbreaks) != 2 {
		t.Fatalf("bash |;: got %d wordbreak tokens, want 2", len(wordbreaks))
	}
	if wordbreaks[0].RawValue != "|" {
		t.Errorf("bash |;: first wordbreak = %q, want |", wordbreaks[0].RawValue)
	}
	if wordbreaks[1].RawValue != ";" {
		t.Errorf("bash |;: second wordbreak = %q, want ;", wordbreaks[1].RawValue)
	}
}
