package shlex

import "testing"

func TestJoin_Posix(t *testing.T) {
	tests := map[string][]string{
		``:                              {},
		`echo hello`:                    {"echo", "hello"},
		`echo "hello world"`:            {"echo", "hello world"},
		`echo "\$(ls)"`:                 {"echo", "$(ls)"},
		`echo "\"ls\""`:                 {"echo", `"ls"`},
		"echo \"\\`ls\\`\"":             {"echo", "`ls`"},
		`echo "with\"doubleQuote"`:      {"echo", `with"doubleQuote`},
		`echo "with'singleQuote"`:       {"echo", "with'singleQuote"},
		`echo "with\$dollar"`:           {"echo", "with$dollar"},
		"echo \"with\\\nlinefeed\"":     {"echo", "with\nlinefeed"},
		"echo \"with\rcarriageReturn\"": {"echo", "with\rcarriageReturn"},
		"echo \"with\ttab\"":            {"echo", "with\ttab"},
		`ls /tmp | xargs -n 1 echo`:     {"ls", "/tmp", "|", "xargs", "-n", "1", "echo"},
	}
	for expected, words := range tests {
		if actual := Join(words, Bash); actual != expected {
			t.Errorf("Join(bash)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoin_Fish(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:           {"echo", "hello"},
		`echo "hello world"`:   {"echo", "hello world"},
		`echo "say \"hello\""`: {"echo", `say "hello"`},
		`echo "cost \$5"`:      {"echo", "cost $5"},
	}
	for expected, words := range tests {
		if actual := Join(words, Fish); actual != expected {
			t.Errorf("Join(fish)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoin_Elvish(t *testing.T) {
	tests := map[string][]string{
		`echo ''`:            {"echo", ""},
		`echo hello`:         {"echo", "hello"},
		`echo 'hello world'`: {"echo", "hello world"},
		`echo 'it''s'`:       {"echo", "it's"},
		`echo C:\path`:       {"echo", `C:\path`},
		`echo 'say "hi"'`:    {"echo", `say "hi"`},
		`echo '$var'`:        {"echo", "$var"},
		`echo '*glob'`:       {"echo", "*glob"},
		`echo '|pipe'`:       {"echo", "|pipe"},
		`echo ';semi'`:       {"echo", ";semi"},
		`echo '(cap'`:        {"echo", "(cap"},
		`echo '[list'`:       {"echo", "[list"},
		`echo '{brace'`:      {"echo", "{brace"},
		`echo '&amp'`:        {"echo", "&amp"},
		`echo '#comment'`:    {"echo", "#comment"},
		`echo 'a=b'`:         {"echo", "a=b"},
		`echo 'a,b'`:         {"echo", "a,b"},
		`echo a~b`:           {"echo", "a~b"},
	}
	for expected, words := range tests {
		if actual := Join(words, Elvish); actual != expected {
			t.Errorf("Join(elvish)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoin_PowerShell(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:         {"echo", "hello"},
		`echo 'hello world'`: {"echo", "hello world"},
		`echo 'don''t'`:      {"echo", "don't"},
	}
	for expected, words := range tests {
		if actual := Join(words, Powershell); actual != expected {
			t.Errorf("Join(powershell)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoin_Nushell(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:           {"echo", "hello"},
		`echo "hello world"`:   {"echo", "hello world"},
		`echo "say \"hello\""`: {"echo", `say "hello"`},
	}
	for expected, words := range tests {
		if actual := Join(words, Nushell); actual != expected {
			t.Errorf("Join(nushell)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoin_Cmd(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:             {"echo", "hello"},
		`echo "hello world"`:     {"echo", "hello world"},
		`echo "say "^"hello"^""`: {"echo", `say "hello"`},
	}
	for expected, words := range tests {
		if actual := Join(words, Cmd); actual != expected {
			t.Errorf("Join(cmd)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}

func TestJoin_Xonsh(t *testing.T) {
	tests := map[string][]string{
		`echo hello`:         {"echo", "hello"},
		`echo 'hello world'`: {"echo", "hello world"},
		`echo 'it\'s'`:       {"echo", "it's"},
	}
	for expected, words := range tests {
		if actual := Join(words, Xonsh); actual != expected {
			t.Errorf("Join(xonsh)\nactual  : %#v\nexpected: %#v", actual, expected)
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
		if actual := Join(words, Default); actual != expected {
			t.Errorf("Join(words, Default)\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}
