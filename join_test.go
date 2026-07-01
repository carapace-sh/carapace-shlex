package shlex

import "testing"

func TestJoinWith_Posix(t *testing.T) {
	tests := map[string][]string{
		``:                            {},
		`echo hello`:                  {"echo", "hello"},
		`echo "hello world"`:          {"echo", "hello world"},
		`echo "\$(ls)"`:               {"echo", "$(ls)"},
		`echo "\"ls\""`:               {"echo", `"ls"`},
		"echo \"\\`ls\\`\"":           {"echo", "`ls`"},
		`echo "with\"doubleQuote"`:    {"echo", `with"doubleQuote`},
		`echo "with'singleQuote"`:     {"echo", "with'singleQuote"},
		`echo "with\$dollar"`:         {"echo", "with$dollar"},
		"echo \"with\\\nlinefeed\"":   {"echo", "with\nlinefeed"},
		"echo \"with\rcarriageReturn\"": {"echo", "with\rcarriageReturn"},
		"echo \"with\ttab\"":            {"echo", "with\ttab"},
		`ls /tmp | xargs -n 1 echo`:   {"ls", "/tmp", "|", "xargs", "-n", "1", "echo"},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, BashFormat()); actual != expected {
			t.Errorf("JoinWith(bash)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinWith_Fish(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:           {"echo", "hello"},
		`echo "hello world"`:   {"echo", "hello world"},
		`echo "say \"hello\""`: {"echo", `say "hello"`},
		`echo "cost \$5"`:      {"echo", "cost $5"},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, FishFormat()); actual != expected {
			t.Errorf("JoinWith(fish)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinWith_Elvish(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:         {"echo", "hello"},
		`echo 'hello world'`: {"echo", "hello world"},
		`echo 'it''s'`:       {"echo", "it's"},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, ElvishFormat()); actual != expected {
			t.Errorf("JoinWith(elvish)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinWith_PowerShell(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:         {"echo", "hello"},
		`echo 'hello world'`: {"echo", "hello world"},
		`echo 'don''t'`:      {"echo", "don't"},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, PowershellFormat()); actual != expected {
			t.Errorf("JoinWith(powershell)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinWith_Nushell(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:           {"echo", "hello"},
		`echo "hello world"`:   {"echo", "hello world"},
		`echo "say \"hello\""`: {"echo", `say "hello"`},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, NushellFormat()); actual != expected {
			t.Errorf("JoinWith(nushell)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinWith_Cmd(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:             {"echo", "hello"},
		`echo "hello world"`:     {"echo", "hello world"},
		`echo "say "^"hello"^""`: {"echo", `say "hello"`},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, CmdFormat()); actual != expected {
			t.Errorf("JoinWith(cmd)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinWith_Xonsh(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:         {"echo", "hello"},
		`echo 'hello world'`: {"echo", "hello world"},
		`echo 'it\'s'`:       {"echo", "it's"},
	}
	for expected, words := range tests {
		if actual := JoinWith(words, XonshFormat()); actual != expected {
			t.Errorf("JoinWith(xonsh)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoinBackwardCompat(t *testing.T) {
	// Join() with no format defaults to bash and should produce
	// the same results as the v1 Join for the existing test cases
	for expected, words := range map[string][]string{
		``:                          {},
		`echo "\$(ls)"`:             {"echo", "$(ls)"},
		"echo \"'ls'\"":             {"echo", "'ls'"},
		`echo "\"ls\""`:             {"echo", `"ls"`},
		`ls /tmp | xargs -n 1 echo`: {"ls", "/tmp", "|", "xargs", "-n", "1", "echo"},
	} {
		if actual := Join(words); actual != expected {
			t.Errorf("Join() backward compat\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}
