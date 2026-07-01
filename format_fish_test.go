package shlex

import "testing"

func TestFishFormat_SingleQuoteEscape(t *testing.T) {
	// Fish: \' inside single quotes → literal '
	tokens, err := SplitWith("echo 'it\\'s'", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != "it's" {
		t.Errorf("fish \\' in single quotes: Value = %q, want %q", last.Value, "it's")
	}
	if last.State != IN_WORD_STATE {
		t.Errorf("fish \\' in single quotes: State = %v, want IN_WORD_STATE", last.State)
	}
}

func TestFishFormat_SingleQuoteBackslashEscape(t *testing.T) {
	// Fish: \\ inside single quotes → literal \
	tokens, err := SplitWith("echo 'C:\\\\path'", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.Value != `C:\path` {
		t.Errorf("fish \\\\ in single quotes: Value = %q, want %q", last.Value, `C:\path`)
	}
}

func TestFishFormat_KeywordOperators(t *testing.T) {
	// Fish: "and" and "or" are keyword operators that split pipelines
	tokens, err := SplitWith("echo foo and echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish keyword operators: %d pipelines, want 2 (and splits)", len(pipelines))
	}
}

func TestFishFormat_KeywordOperatorAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo and", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	// "and" should be a WORDBREAK_TOKEN
	last := tokens[len(tokens)-1]
	if last.Type != WORDBREAK_TOKEN {
		t.Errorf("fish 'and': Type = %v, want WORDBREAK_TOKEN", last.Type)
	}
	if last.WordbreakType != WORDBREAK_LIST_AND {
		t.Errorf("fish 'and': WordbreakType = %v, want WORDBREAK_LIST_AND", last.WordbreakType)
	}
}

func TestFishFormat_KeywordOperatorOr(t *testing.T) {
	tokens, err := SplitWith("echo foo or", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens[len(tokens)-1]
	if last.Type != WORDBREAK_TOKEN {
		t.Errorf("fish 'or': Type = %v, want WORDBREAK_TOKEN", last.Type)
	}
	if last.WordbreakType != WORDBREAK_LIST_OR {
		t.Errorf("fish 'or': WordbreakType = %v, want WORDBREAK_LIST_OR", last.WordbreakType)
	}
}

func TestFishFormat_NotNotKeywordOperator(t *testing.T) {
	// "not" is a prefix keyword but NOT a pipeline delimiter
	tokens, err := SplitWith("echo foo not", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens[len(tokens)-1]
	if last.Type != WORD_TOKEN {
		t.Errorf("fish 'not': Type = %v, want WORD_TOKEN (not a delimiter)", last.Type)
	}
}

func TestFishFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != QUOTING_STATE {
		t.Errorf("fish open quote: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestFishFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish pipe: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish semicolon: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_DoubleQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "hello world"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != "hello world" {
		t.Errorf("fish double: Words = %v, want [echo hello world]", words)
	}
}

func TestFishFormat_CompletionContext(t *testing.T) {
	ctx := SplitForCompletion("echo foo and grep hel", FishFormat())
	if ctx.CurrentWord != "hel" {
		t.Errorf("fish completion: CurrentWord = %q, want %q", ctx.CurrentWord, "hel")
	}
	if len(ctx.Words) != 2 {
		t.Errorf("fish completion: Words = %v, want 2 words (grep hel)", ctx.Words)
	}
}

func TestFishFormat_DollarNotEscapeInSingleQuotes(t *testing.T) {
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

func TestFishFormat_EscapedSpace(t *testing.T) {
	tokens, err := SplitWith(`echo a\ b`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "a b" {
		t.Errorf("fish escaped space: Words = %v, want [echo a b]", words)
	}
}

func TestFishFormat_ParensNotWordbreak(t *testing.T) {
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

func TestFishFormat_AndAnd(t *testing.T) {
	tokens, err := SplitWith("echo foo && echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish &&: %d pipelines, want 2", len(pipelines))
	}
	last := tokens[len(tokens)-3]
	if last.Type != WORDBREAK_TOKEN || last.WordbreakType != WORDBREAK_LIST_AND {
		t.Errorf("fish &&: Type=%v WordbreakType=%v, want WORDBREAK_TOKEN/LIST_AND", last.Type, last.WordbreakType)
	}
}

func TestFishFormat_OrOr(t *testing.T) {
	tokens, err := SplitWith("echo foo || echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish ||: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_Background(t *testing.T) {
	tokens, err := SplitWith("echo foo & echo bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish &: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_PipeWithStderrMerge(t *testing.T) {
	tokens, err := SplitWith("echo foo |& cat bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish |&: %d pipelines, want 2", len(pipelines))
	}
	wb := tokens[len(tokens)-3]
	if wb.Type != WORDBREAK_TOKEN || wb.WordbreakType != WORDBREAK_PIPE_WITH_STDERR {
		t.Errorf("fish |&: Type=%v WordbreakType=%v, want WORDBREAK_TOKEN/PIPE_WITH_STDERR", wb.Type, wb.WordbreakType)
	}
}

func TestFishFormat_AmpPipe(t *testing.T) {
	tokens, err := SplitWith("echo foo &| cat bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish &|: %d pipelines, want 2", len(pipelines))
	}
}

func TestFishFormat_ExplicitFdPipe(t *testing.T) {
	// >| is a pipe with explicit fd in fish (e.g. echo foo >| bar)
	tokens, err := SplitWith("echo foo >| cat bar", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("fish >|: %d pipelines, want 2", len(pipelines))
	}
	wb := tokens[len(tokens)-3]
	if wb.Type != WORDBREAK_TOKEN || wb.WordbreakType != WORDBREAK_PIPE {
		t.Errorf("fish >|: Type=%v WordbreakType=%v, want WORDBREAK_TOKEN/PIPE", wb.Type, wb.WordbreakType)
	}
}

func TestFishFormat_AmpRedirect(t *testing.T) {
	_, err := SplitWith("echo foo &> file.txt", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	ctx := SplitForCompletion("echo foo &> file.txt", FishFormat())
	if !ctx.IsRedirect {
		t.Errorf("fish &>: IsRedirect = false, want true")
	}
}

func TestFishFormat_AmpRedirectAppend(t *testing.T) {
	_, err := SplitWith("echo foo &>> file.txt", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	ctx := SplitForCompletion("echo foo &>> file.txt", FishFormat())
	if !ctx.IsRedirect {
		t.Errorf("fish &>>: IsRedirect = false, want true")
	}
}

func TestFishFormat_FdRedirect(t *testing.T) {
	// >&2 is a fd redirect: >& is the operator, 2 is the fd number.
	// After >&2, the cursor is at a new word (not a redirect target).
	// Test that >& is classified as a redirect operator.
	tokens, err := SplitWith("echo foo >&2", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_REDIRECT_INPUT_DUPLICATE {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("fish >&2: no WORDBREAK_REDIRECT_INPUT_DUPLICATE token found")
	}
}

func TestFishFormat_InputOutputRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo <> ", FishFormat())
	if !ctx.IsRedirect {
		t.Errorf("fish <>: IsRedirect = false, want true")
	}
}

func TestFishFormat_NoclobberRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo >? ", FishFormat())
	if !ctx.IsRedirect {
		t.Errorf("fish >?: IsRedirect = false, want true")
	}
}

func TestFishFormat_NoclobberAppendRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo >>? ", FishFormat())
	if !ctx.IsRedirect {
		t.Errorf("fish >>?: IsRedirect = false, want true")
	}
}

func TestFishFormat_TryInputRedirect(t *testing.T) {
	ctx := SplitForCompletion("echo foo <? ", FishFormat())
	if !ctx.IsRedirect {
		t.Errorf("fish <?: IsRedirect = false, want true")
	}
}

func TestFishFormat_DoubleQuoteEscapedQuote(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("fish \\\" in double: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestFishFormat_DoubleQuoteEscapedDollar(t *testing.T) {
	tokens, err := SplitWith(`echo "cost: \$5"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `cost: $5` {
		t.Errorf("fish \\$ in double: Words = %v, want [echo cost: $5]", words)
	}
}

func TestFishFormat_DoubleQuoteEscapedBackslash(t *testing.T) {
	tokens, err := SplitWith(`echo "C:\\path"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `C:\path` {
		t.Errorf("fish \\\\ in double: Words = %v, want [echo C:\\path]", words)
	}
}

func TestFishFormat_DoubleQuoteNonEscapeBackslash(t *testing.T) {
	// \n inside fish double quotes is NOT an escape — both \ and n are literal
	tokens, err := SplitWith(`echo "hello\nworld"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `hello\nworld` {
		t.Errorf("fish \\n in double: Words = %v, want [echo hello\\nworld]", words)
	}
}

func TestFishFormat_DoubleQuoteNonEscapeBackslashOther(t *testing.T) {
	// \t inside fish double quotes is NOT an escape — both \ and t are literal
	tokens, err := SplitWith(`echo "a\tb"`, FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `a\tb` {
		t.Errorf("fish \\t in double: Words = %v, want [echo a\\tb]", words)
	}
}

func TestFishFormat_DoubleQuoteEscapedNewline(t *testing.T) {
	// \<newline> is a line continuation escape inside fish double quotes
	tokens, err := SplitWith("echo \"hello\\\nworld\"", FishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "hello\nworld" {
		t.Errorf("fish \\<newline> in double: Words = %v, want [echo hello\\nworld]", words)
	}
}

func TestFishFormat_QuoteWordBacktick(t *testing.T) {
	// Backtick is a regular character in fish — should not trigger quoting
	q := fishQuoteWord("hello`world")
	if q != "hello`world" {
		t.Errorf("fishQuoteWord backtick: got %q, want %q", q, "hello`world")
	}
}

func TestFishFormat_QuoteWordDollar(t *testing.T) {
	q := fishQuoteWord("hello$world")
	if q != `"hello\$world"` {
		t.Errorf("fishQuoteWord dollar: got %q, want %q", q, `"hello\$world"`)
	}
}

func TestFishFormat_QuoteWordSafe(t *testing.T) {
	q := fishQuoteWord("hello-world")
	if q != "hello-world" {
		t.Errorf("fishQuoteWord safe: got %q, want %q", q, "hello-world")
	}
}

func TestFishFormat_CompletionAndAnd(t *testing.T) {
	ctx := SplitForCompletion("echo foo && echo bar hel", FishFormat())
	if ctx.CurrentWord != "hel" {
		t.Errorf("fish && completion: CurrentWord = %q, want %q", ctx.CurrentWord, "hel")
	}
	if len(ctx.Words) != 3 {
		t.Errorf("fish && completion: Words = %v, want 3 (echo bar hel)", ctx.Words)
	}
}

func TestFishFormat_CompletionBackground(t *testing.T) {
	ctx := SplitForCompletion("echo foo & echo bar hel", FishFormat())
	if ctx.CurrentWord != "hel" {
		t.Errorf("fish & completion: CurrentWord = %q, want %q", ctx.CurrentWord, "hel")
	}
	if len(ctx.Words) != 3 {
		t.Errorf("fish & completion: Words = %v, want 3 (echo bar hel)", ctx.Words)
	}
}
