package shlex

import "testing"

func TestSubstitution_NestedScopeLastOpenTracking(t *testing.T) {
	// $(( opens (arithmetic, skipped), then ( opens (depth=1, command scope),
	// then ) closes (depth=0). But the $(( arithmetic is still open.
	// Since arithmetic is not a command scope, innermostUnclosedCommandScope
	// should return -1 (no unclosed command scope).
	tokens, err := SplitWith("echo $(( (foo) bar", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope >= 0 {
		t.Errorf("innermostUnclosedCommandScope = %d, want -1 (arithmetic is not a command scope)", scope)
	}
}

func TestSubstitution_NestedCommandSubLastOpenTracking(t *testing.T) {
	// $( (test) foo — $( opens (depth=1), ( opens (depth=2), ) closes
	// (depth=1). $(( is still unclosed. innermostUnclosedCommandScope
	// should return the index of $(, not the already-closed (.
	tokens, err := SplitWith("echo $( (test) foo", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope < 0 {
		t.Fatal("innermostUnclosedCommandScope = -1, want >= 0 (unclosed $()")
	}
	opener := tokens[scope]
	if opener.Value != "$(" {
		t.Errorf("scope opener Value = %q, want $( (not the already-closed ()", opener.Value)
	}
}

func TestSubstitution_StrayCloseDoesNotMaskOpen(t *testing.T) {
	// ) $(git ch — stray ) should not make depth negative,
	// so the unclosed $( should be detected.
	tokens, err := SplitWith("echo ) $(git ch", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope < 0 {
		t.Error("innermostUnclosedCommandScope = -1, want >= 0 (unclosed $( after stray ))")
	}
	depth := countUnclosedCommandScopes(tokens)
	if depth != 1 {
		t.Errorf("countUnclosedCommandScopes = %d, want 1", depth)
	}
}

func TestSubstitution_ArithmeticInsideCommandSub(t *testing.T) {
	// $(echo $((1+2)) — $( opens (depth=1, arith=0), $(( opens
	// (depth=2, arith=1), )) closes arithmetic (depth=1, arith=0).
	// $( is still unclosed. innermostUnclosedCommandScope should
	// detect the unclosed $().
	tokens, err := SplitWith("echo $(echo $((1+2))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope < 0 {
		t.Error("innermostUnclosedCommandScope = -1, want >= 0 (unclosed $() inside $((...))")
	}
	depth := countUnclosedCommandScopes(tokens)
	if depth != 1 {
		t.Errorf("countUnclosedCommandScopes = %d, want 1", depth)
	}
}

func TestSubstitution_ArithmeticClosed(t *testing.T) {
	// $((1+2)) should fully close — depth 0, no unclosed scope.
	tokens, err := SplitWith("echo $((1+2))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope >= 0 {
		t.Errorf("innermostUnclosedCommandScope = %d, want -1 (arithmetic closed)", scope)
	}
	depth := countUnclosedCommandScopes(tokens)
	if depth != 0 {
		t.Errorf("countUnclosedCommandScopes = %d, want 0", depth)
	}
}

func TestSubstitution_NestedArithmetic(t *testing.T) {
	// $(( $((1+2)) + 3 )) — nested arithmetic should fully close.
	tokens, err := SplitWith("echo $(( $((1+2)) + 3 ))", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope >= 0 {
		t.Errorf("innermostUnclosedCommandScope = %d, want -1 (nested arithmetic closed)", scope)
	}
}

func TestSubstitution_EmptyClosed(t *testing.T) {
	// $() — empty but closed.
	tokens, err := SplitWith("echo $()", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[1] != "$()" {
		t.Errorf("Words = %v, want [echo $()]", words)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope >= 0 {
		t.Errorf("innermostUnclosedCommandScope = %d, want -1 ($() is closed)", scope)
	}
}

func TestSubstitution_OpenAtEOF(t *testing.T) {
	// $( at EOF — open with no content.
	tokens, err := SplitWith("echo $(", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope < 0 {
		t.Error("innermostUnclosedCommandScope = -1, want >= 0 (unclosed $( at EOF)")
	}
	ctx := SplitForCompletion("echo $(", BashFormat())
	if ctx.SubstitutionDepth != 1 {
		t.Errorf("SubstitutionDepth = %d, want 1", ctx.SubstitutionDepth)
	}
}

func TestSubstitution_StrayClosePipeline(t *testing.T) {
	// ) echo | test — stray ) should not cause depth to go negative
	// and mask the pipe in Pipelines().
	tokens, err := SplitWith("echo ) | grep foo", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	pipelines := tokens.Pipelines()
	if len(pipelines) != 2 {
		t.Errorf("Pipelines = %d, want 2", len(pipelines))
	}
}

func TestSubstitution_CommandSubInArithmeticAtEOF(t *testing.T) {
	// $(( $(echo test) — arithmetic open, command sub closed.
	// innermostUnclosedCommandScope should return -1 (arithmetic
	// is not a command scope, and the command sub is closed).
	tokens, err := SplitWith("echo $(( $(echo test)", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope >= 0 {
		t.Errorf("innermostUnclosedCommandScope = %d, want -1 (command sub closed, arithmetic is not a command scope)", scope)
	}
}

func TestSubstitution_NestedCommandSubWithArithmetic(t *testing.T) {
	// $( $((1+2)) ) — command sub containing arithmetic, both closed.
	tokens, err := SplitWith("echo $( $((1+2)) )", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	scope := innermostUnclosedCommandScope(tokens)
	if scope >= 0 {
		t.Errorf("innermostUnclosedCommandScope = %d, want -1 (both closed)", scope)
	}
	words := tokens.WordsWithSubstitutions().Strings()
	if len(words) != 2 || words[1] != "$( $((1+2)) )" {
		t.Errorf("Words = %v, want [echo $( $((1+2)) )]", words)
	}
}

func TestSubstitution_AppendProcessSubstitution(t *testing.T) {
	// >>(cmd) — append process substitution. The >> redirect operator
	// followed by ( should be detected as process substitution.
	tokens, err := SplitWith("echo >>(grep foo)", BashFormat())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tok := range tokens {
		if tok.WordbreakType == WORDBREAK_SUBSTITUTION_OPEN && tok.Value == ">>(" {
			found = true
		}
	}
	if !found {
		t.Errorf(">>(cmd): no >>( substitution opener found in %v", tokens)
	}
}
