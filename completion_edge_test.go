package shlex

import "testing"

func TestCompletion_MultipleWordbreaks(t *testing.T) {
	ctx := Complete("echo a=b=c", Bash)
	if ctx.CurrentWord != "a=b=c" {
		t.Errorf("CurrentWord = %q, want a=b=c", ctx.CurrentWord)
	}
	if ctx.Prefix != "a=b=" {
		t.Errorf("Prefix = %q, want a=b=", ctx.Prefix)
	}
}

func TestCompletion_TrailingWordbreak(t *testing.T) {
	ctx := Complete("echo a=b=", Bash)
	if ctx.CurrentWord != "a=b=" {
		t.Errorf("CurrentWord = %q, want a=b=", ctx.CurrentWord)
	}
	if ctx.Prefix != "a=b=" {
		t.Errorf("Prefix = %q, want a=b=", ctx.Prefix)
	}
}

func TestCompletion_WordbreakWithSpace(t *testing.T) {
	ctx := Complete("echo a= b=c", Bash)
	if ctx.CurrentWord != "b=c" {
		t.Errorf("CurrentWord = %q, want b=c", ctx.CurrentWord)
	}
	if ctx.Prefix != "b=" {
		t.Errorf("Prefix = %q, want b=", ctx.Prefix)
	}
}

func TestCompletion_MultipleAtWordbreaks(t *testing.T) {
	ctx := Complete("echo foo@@bar", Bash)
	if ctx.Prefix != "foo" {
		t.Errorf("Prefix = %q, want foo (all @ should be skipped)", ctx.Prefix)
	}
}

func TestCompletion_AtAfterOtherWordbreak(t *testing.T) {
	ctx := Complete("echo foo@:bar", Bash)
	// @ should be skipped, but : is a regular wordbreak and included
	if ctx.Prefix != "foo:" {
		t.Errorf("Prefix = %q, want foo: (@ skipped, : included)", ctx.Prefix)
	}
}

func TestCompletion_ColonWordbreak(t *testing.T) {
	ctx := Complete("echo foo:bar", Bash)
	if ctx.Prefix != "foo:" {
		t.Errorf("Prefix = %q, want foo:", ctx.Prefix)
	}
}

func TestCompletion_FdPrefix(t *testing.T) {
	ctx := Complete("echo 2> file", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
	if ctx.CurrentWord != "file" {
		t.Errorf("CurrentWord = %q, want file", ctx.CurrentWord)
	}
}

func TestCompletion_MultiDigitFd(t *testing.T) {
	ctx := Complete("echo 10> file", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
	if ctx.CurrentWord != "file" {
		t.Errorf("CurrentWord = %q, want file", ctx.CurrentWord)
	}
}

func TestCompletion_FdRedirectNoSpace(t *testing.T) {
	ctx := Complete("echo 2>file", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
}

func TestCompletion_RedirectAtEOF(t *testing.T) {
	ctx := Complete("echo foo >", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
	if ctx.CurrentWord != "" {
		t.Errorf("CurrentWord = %q, want empty", ctx.CurrentWord)
	}
}

func TestCompletion_FdRedirectAtEOF(t *testing.T) {
	ctx := Complete("echo 2>", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
}

func TestCompletion_FdDuplication(t *testing.T) {
	ctx := Complete("echo 2>&1", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
}

func TestCompletion_WordbreakBeforeDoubleQuote(t *testing.T) {
	ctx := Complete(`echo foo="bar`, Bash)
	if ctx.QuotingState != QUOTING_ESCAPING_STATE {
		t.Errorf("QuotingState = %v, want QUOTING_ESCAPING_STATE", ctx.QuotingState)
	}
	if ctx.Prefix != "foo=" {
		t.Errorf("Prefix = %q, want foo=", ctx.Prefix)
	}
}

func TestCompletion_WordbreakBeforeSingleQuote(t *testing.T) {
	ctx := Complete("echo foo='bar", Bash)
	if ctx.QuotingState != QUOTING_STATE {
		t.Errorf("QuotingState = %v, want QUOTING_STATE", ctx.QuotingState)
	}
	if ctx.Prefix != "foo=" {
		t.Errorf("Prefix = %q, want foo=", ctx.Prefix)
	}
}

func TestCompletion_MidWordQuoteOpening(t *testing.T) {
	ctx := Complete("echo foo=bar'baz", Bash)
	if ctx.QuotingState != QUOTING_STATE {
		t.Errorf("QuotingState = %v, want QUOTING_STATE", ctx.QuotingState)
	}
	if ctx.Prefix != "foo=bar" {
		t.Errorf("Prefix = %q, want foo=bar", ctx.Prefix)
	}
}

func TestCompletion_WordbreakCharInsideQuotes(t *testing.T) {
	ctx := Complete(`echo "foo=bar`, Bash)
	if ctx.QuotingState != QUOTING_ESCAPING_STATE {
		t.Errorf("QuotingState = %v, want QUOTING_ESCAPING_STATE", ctx.QuotingState)
	}
	if ctx.Prefix != "" {
		t.Errorf("Prefix = %q, want empty (= inside quotes is not a wordbreak)", ctx.Prefix)
	}
}

func TestCompletion_CommentOnlyInput(t *testing.T) {
	ctx := Complete("# comment", Bash)
	if ctx.QuotingState != START_STATE {
		t.Errorf("QuotingState = %v, want START_STATE", ctx.QuotingState)
	}
	if len(ctx.Words) != 0 {
		t.Errorf("Words = %v, want empty (comment-only input)", ctx.Words)
	}
}

func TestCompletion_WhitespaceOnlyInput(t *testing.T) {
	ctx := Complete("   ", Bash)
	if ctx.QuotingState != START_STATE {
		t.Errorf("QuotingState = %v, want START_STATE", ctx.QuotingState)
	}
	if ctx.CurrentWord != "" {
		t.Errorf("CurrentWord = %q, want empty", ctx.CurrentWord)
	}
}

func TestCompletion_BareRedirect(t *testing.T) {
	ctx := Complete(">", Bash)
	if !ctx.IsRedirect {
		t.Error("IsRedirect = false, want true")
	}
}

func TestCompletion_BarePipe(t *testing.T) {
	ctx := Complete("|", Bash)
	if ctx.CurrentWord != "" {
		t.Errorf("CurrentWord = %q, want empty", ctx.CurrentWord)
	}
}

func TestCompletion_AtBetweenWordsThenWordbreak(t *testing.T) {
	// user@host:bar — @ between two WORD tokens should be in prefix
	ctx := Complete("echo user@host:bar", Bash)
	if ctx.Prefix != "user@host:" {
		t.Errorf("Prefix = %q, want user@host: (@ between words preserved)", ctx.Prefix)
	}
}

func TestCompletion_AtAfterWordbreakThenWord(t *testing.T) {
	// foo:@bar — @ adjacent to : wordbreak should be skipped
	ctx := Complete("echo foo:@bar", Bash)
	if ctx.Prefix != "foo:" {
		t.Errorf("Prefix = %q, want foo: (@ adjacent to : skipped)", ctx.Prefix)
	}
}

func TestCompletion_AtBetweenWordbreakAndWord(t *testing.T) {
	// :@bar — @ adjacent to : (wordbreak side) should be skipped
	ctx := Complete("echo foo:=@bar", Bash)
	if ctx.Prefix != "foo:=" {
		t.Errorf("Prefix = %q, want foo:= (@ adjacent to = skipped)", ctx.Prefix)
	}
}
