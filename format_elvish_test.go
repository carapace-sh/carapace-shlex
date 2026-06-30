package shlex

import "testing"

func TestElvishFormat_BarewordBackslash(t *testing.T) {
	// Elvish: \ is a bareword character outside quotes (not an escape)
	tokens, err := SplitWith(`echo C:\path`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[0] != "echo" || words[1] != `C:\path` {
		t.Errorf("elvish bareword \\: Words = %v, want [echo C:\\path]", words)
	}
}

func TestElvishFormat_DoubleQuoteEscape(t *testing.T) {
	// Elvish: \ IS an escape inside double quotes
	tokens, err := SplitWith(`echo "hello\nworld"`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words()
	last := words[len(words)-1]
	if last.State != IN_WORD_STATE {
		t.Errorf("elvish double-quote escape: State = %v, want IN_WORD_STATE", last.State)
	}
}

func TestElvishFormat_DoubledQuoteSplit(t *testing.T) {
	tokens, err := SplitWith("echo 'it''s a test'", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != "it's a test" {
		t.Errorf("elvish '' split: Words = %v, want [echo it's a test]", words)
	}
}

func TestElvishFormat_DoubleQuoteValue(t *testing.T) {
	tokens, err := SplitWith(`echo "say \"hello\""`, ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.Words().Strings()
	if len(words) != 2 || words[1] != `say "hello"` {
		t.Errorf("elvish double value: Words = %v, want [echo say \"hello\"]", words)
	}
}

func TestElvishFormat_AmpNotListOperator(t *testing.T) {
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

func TestElvishFormat_Pipe(t *testing.T) {
	tokens, err := SplitWith("echo foo | grep bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("elvish pipe: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestElvishFormat_Semicolon(t *testing.T) {
	tokens, err := SplitWith("echo foo ; echo bar", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens.Pipelines()) != 2 {
		t.Errorf("elvish semicolon: %d pipelines, want 2", len(tokens.Pipelines()))
	}
}

func TestElvishFormat_OpenSingleQuote(t *testing.T) {
	tokens, err := SplitWith("echo 'hel", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	last := tokens.Words().CurrentToken()
	if last.State != QUOTING_STATE {
		t.Errorf("elvish open single: State = %v, want QUOTING_STATE", last.State)
	}
}

func TestElvishFormat_LambdaPipe(t *testing.T) {
	// {| — the | after { is a lambda parameter delimiter, not a pipeline pipe
	tokens, err := SplitWith("bat | {|", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	// Find the second | (after {) — it should be WORDBREAK_LAMBDA_PIPE
	lambdaPipes := 0
	realPipes := 0
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.Value == "|" {
			switch tok.WordbreakType {
			case WORDBREAK_LAMBDA_PIPE:
				lambdaPipes++
			case WORDBREAK_PIPE:
				realPipes++
			}
		}
	}
	if lambdaPipes != 1 {
		t.Errorf("elvish lambda pipe: %d lambda pipes, want 1", lambdaPipes)
	}
	if realPipes != 1 {
		t.Errorf("elvish lambda pipe: %d real pipes, want 1", realPipes)
	}
}

func TestElvishFormat_LambdaPipeParams(t *testing.T) {
	// {|a b| — both | are lambda parameter delimiters
	tokens, err := SplitWith("bat | {|a b|", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	lambdaPipes := 0
	realPipes := 0
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.Value == "|" {
			switch tok.WordbreakType {
			case WORDBREAK_LAMBDA_PIPE:
				lambdaPipes++
			case WORDBREAK_PIPE:
				realPipes++
			}
		}
	}
	if lambdaPipes != 2 {
		t.Errorf("elvish lambda params: %d lambda pipes, want 2", lambdaPipes)
	}
	if realPipes != 1 {
		t.Errorf("elvish lambda params: %d real pipes, want 1 (the bat pipe)", realPipes)
	}
}

func TestElvishFormat_LambdaBodyPipe(t *testing.T) {
	// {|a| cmd1 | cmd2} — first two | are lambda delimiters, third is a real pipe in the body
	tokens, err := SplitWith("{|a| cmd1 | cmd2}", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	lambdaPipes := 0
	realPipes := 0
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.Value == "|" {
			switch tok.WordbreakType {
			case WORDBREAK_LAMBDA_PIPE:
				lambdaPipes++
			case WORDBREAK_PIPE:
				realPipes++
			}
		}
	}
	if lambdaPipes != 2 {
		t.Errorf("elvish lambda body: %d lambda pipes, want 2", lambdaPipes)
	}
	if realPipes != 1 {
		t.Errorf("elvish lambda body: %d real pipes, want 1 (the body pipe)", realPipes)
	}
}

func TestElvishFormat_BracedListNotLambda(t *testing.T) {
	// {a,b} is a braced list, not a lambda — no | to reclassify
	tokens, err := SplitWith("echo {a,b}", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	lambdaPipes := 0
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_LAMBDA_PIPE {
			lambdaPipes++
		}
	}
	if lambdaPipes != 0 {
		t.Errorf("elvish braced list: %d lambda pipes, want 0", lambdaPipes)
	}
}

func TestElvishFormat_LambdaNoParams(t *testing.T) {
	// { body } — lambda with no params (space after {), no | at all
	tokens, err := SplitWith("var f = { echo hi }", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	lambdaPipes := 0
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_LAMBDA_PIPE {
			lambdaPipes++
		}
	}
	if lambdaPipes != 0 {
		t.Errorf("elvish lambda no params: %d lambda pipes, want 0", lambdaPipes)
	}
}

func TestElvishFormat_NestedLambda(t *testing.T) {
	// {|a| {|b| echo $a $b }} — nested lambdas
	tokens, err := SplitWith("{|a| {|b| echo $a $b }}", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	lambdaPipes := 0
	for _, tok := range tokens {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_LAMBDA_PIPE {
			lambdaPipes++
		}
	}
	if lambdaPipes != 4 {
		t.Errorf("elvish nested lambda: %d lambda pipes, want 4 (2 per lambda)", lambdaPipes)
	}
}

func TestElvishFormat_LambdaPipeDoesNotSplitPipeline(t *testing.T) {
	// The lambda | should not split CurrentPipeline — only the real pipe before {|
	tokens, err := SplitWith("bat | {|a", ElvishFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("elvish lambda pipeline split: %d pipelines, want 2 (bat and {|a)", len(pipelines))
	}
	current := tokens.CurrentPipeline()
	// Current pipeline should contain {|a — not split at the lambda |
	// The {, lambda |, and a adjoin so Words() merges them into {|a
	words := current.Words().Strings()
	if len(words) == 0 {
		t.Errorf("elvish lambda pipeline: current pipeline is empty, want non-empty")
	}
	// The pipeline should not be split at the lambda |, so it should contain
	// the lambda tokens (not just an empty slice after a second pipe split)
	hasLambdaPipe := false
	for _, tok := range current {
		if tok.Type == WORDBREAK_TOKEN && tok.WordbreakType == WORDBREAK_LAMBDA_PIPE {
			hasLambdaPipe = true
		}
	}
	if !hasLambdaPipe {
		t.Errorf("elvish lambda pipeline: current pipeline has no lambda pipe, want one")
	}
}

func TestElvishFormat_CompletionInLambdaParams(t *testing.T) {
	// Cursor after {| — should be in lambda parameter position
	ctx := SplitForCompletion("bat | {|", ElvishFormat())
	if !ctx.InLambdaParams {
		t.Errorf("elvish completion in lambda params: InLambdaParams = false, want true")
	}
}

func TestElvishFormat_CompletionInLambdaParamsWithArg(t *testing.T) {
	// Cursor after {|a — still in parameter list
	ctx := SplitForCompletion("bat | {|a", ElvishFormat())
	if !ctx.InLambdaParams {
		t.Errorf("elvish completion in lambda params (with arg): InLambdaParams = false, want true")
	}
}

func TestElvishFormat_CompletionAfterLambdaParams(t *testing.T) {
	// Cursor after {|a| — parameter list closed, in lambda body
	ctx := SplitForCompletion("bat | {|a|", ElvishFormat())
	if ctx.InLambdaParams {
		t.Errorf("elvish completion after lambda params: InLambdaParams = true, want false (in body)")
	}
}

func TestElvishFormat_CompletionNotInLambda(t *testing.T) {
	// Cursor after regular pipe — not in lambda
	ctx := SplitForCompletion("bat | grep ", ElvishFormat())
	if ctx.InLambdaParams {
		t.Errorf("elvish completion not in lambda: InLambdaParams = true, want false")
	}
}
