/*
Copyright 2012 Google Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific governing permissions and
limitations under the License.
*/

package shlex

import (
	"os"
	"strings"
	"testing"
)

var (
	// one two "three four" "five \"six\"" seven#eight # nine # ten
	// eleven 'twelve\'
	testString = "one two \"three four\" \"five \\\"six\\\"\" seven#eight # nine # ten\n eleven 'twelve\\' thirteen=13 fourteen/14 | || |after before| & ;"
)

func init() {
	os.Unsetenv("COMP_WORDBREAKS")
}

func TestTokenizer(t *testing.T) {
	testInput := strings.NewReader(testString)
	expectedTokens := []*Token{
		{WORD_TOKEN, "one", "one", Span{Start: 0, End: 3}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "two", "two", Span{Start: 4, End: 7}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "three four", "\"three four\"", Span{Start: 8, End: 20}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "five \"six\"", "\"five \\\"six\\\"\"", Span{Start: 21, End: 35}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "seven#eight", "seven#eight", Span{Start: 36, End: 47}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{COMMENT_TOKEN, " nine # ten", "# nine # ten", Span{Start: 48, End: 60}, START_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "eleven", "eleven", Span{Start: 62, End: 68}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "twelve\\", "'twelve\\'", Span{Start: 69, End: 78}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "thirteen", "thirteen", Span{Start: 79, End: 87}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORDBREAK_TOKEN, "=", "=", Span{Start: 87, End: 88}, WORDBREAK_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "13", "13", Span{Start: 88, End: 90}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "fourteen/14", "fourteen/14", Span{Start: 91, End: 102}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORDBREAK_TOKEN, "|", "|", Span{Start: 103, End: 104}, WORDBREAK_STATE, WORDBREAK_PIPE, 0},
		{WORDBREAK_TOKEN, "||", "||", Span{Start: 105, End: 107}, WORDBREAK_STATE, WORDBREAK_LIST_OR, 0},
		{WORDBREAK_TOKEN, "|", "|", Span{Start: 108, End: 109}, WORDBREAK_STATE, WORDBREAK_PIPE, 0},
		{WORD_TOKEN, "after", "after", Span{Start: 109, End: 114}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORD_TOKEN, "before", "before", Span{Start: 115, End: 121}, IN_WORD_STATE, WORDBREAK_UNKNOWN, 0},
		{WORDBREAK_TOKEN, "|", "|", Span{Start: 121, End: 122}, WORDBREAK_STATE, WORDBREAK_PIPE, 0},
		{WORDBREAK_TOKEN, "&", "&", Span{Start: 123, End: 124}, WORDBREAK_STATE, WORDBREAK_LIST_ASYNC, 0},
		{WORDBREAK_TOKEN, ";", ";", Span{Start: 125, End: 126}, WORDBREAK_STATE, WORDBREAK_LIST_SEQUENTIAL, 0},
		{WORD_TOKEN, "", "", Span{Start: 126, End: 126}, START_STATE, WORDBREAK_UNKNOWN, 0},
	}

	tokenizer := newTokenizer(testInput, BashFormat())
	for i, want := range expectedTokens {
		got, err := tokenizer.Next()
		if err != nil {
			t.Error(err)
		}
		if !got.Equal(want) {
			t.Errorf("Tokenizer.Next()[%v] of %q \nGot : %#v\nWant: %#v", i, testString, got, want)
		}
	}
}

func TestLexer(t *testing.T) {
	testInput := strings.NewReader(testString)
	expectedStrings := []string{"one", "two", "three four", "five \"six\"", "seven#eight", "eleven", "twelve\\", "thirteen", "=", "13", "fourteen/14"}

	lexer := newLexer(testInput, BashFormat())
	for i, want := range expectedStrings {
		got, err := lexer.Next()
		if err != nil {
			t.Error(err)
		}
		if got.Value != want {
			t.Errorf("Lexer.Next()[%v] of %q -> %v. Want: %v", i, testString, got, want)
		}
	}
}

func TestSplit(t *testing.T) {
	want := []string{"one", "two", "three four", "five \"six\"", "seven#eight", "eleven", "twelve\\", "thirteen", "=", "13", "fourteen/14", "|", "||", "|", "after", "before", "|", "&", ";", ""}
	got, err := Split(testString)
	if err != nil {
		t.Error(err)
	}
	if len(want) != len(got) {
		t.Errorf("Split(%q) -> %v. Want: %v", testString, got, want)
	}
	for i, g := range got {
		if g.Value != want[i] {
			t.Errorf("Split(%q)[%v] -> %v. Want: %v", testString, i, g.Value, want[i])
		}
	}
}

func TestJoin(t *testing.T) {
	for expected, words := range map[string][]string{
		``:                          {},
		`echo "\$(ls)"`:             {"echo", "$(ls)"},
		"echo \"'ls'\"":             {"echo", "'ls'"},
		`echo "\"ls\""`:             {"echo", `"ls"`},
		`echo "\$(ls /tmp)"`:        {"echo", "$(ls /tmp)"},
		"echo \"\\`ls\\`\"":         {"echo", "`ls`"},
		`ls /tmp | xargs -n 1 echo`: {"ls", "/tmp", "|", "xargs", "-n", "1", "echo"},

		`echo "with\"doubleQuote"`:    {"echo", "with\"doubleQuote"},
		`echo "with'singleQuote"`:     {"echo", "with'singleQuote"},
		`echo "with space"`:           {"echo", "with space"},
		"echo \"with\\`backtick\"":    {"echo", "with`backtick"},
		`echo "with\$dollar"`:         {"echo", "with$dollar"},
		"echo \"with\\\nlinefeed\"":     {"echo", "with\nlinefeed"},
		"echo \"with\rcarriageReturn\"": {"echo", "with\rcarriageReturn"},
		"echo \"with\ttab\"":            {"echo", "with\ttab"},
	} {
		if actual := Join(words); actual != expected {
			t.Errorf("joined words don't match\nactual  : %#v\nexpected: %#v", actual, expected)
		}
	}
}
