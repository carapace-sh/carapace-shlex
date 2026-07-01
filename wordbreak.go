package shlex

import "encoding/json"

const BASH_WORDBREAKS = " \t\r\n" + `"'@><=;|&():`

type WordbreakType int

const (
	WORDBREAK_UNKNOWN WordbreakType = iota
	// https://www.gnu.org/software/bash/manual/html_node/Redirections.html
	WORDBREAK_REDIRECT_INPUT
	WORDBREAK_REDIRECT_OUTPUT
	WORDBREAK_REDIRECT_OUTPUT_APPEND
	WORDBREAK_REDIRECT_OUTPUT_FORCE        // >| (noclobber override)
	WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE // >>| (noclobber override for append)
	WORDBREAK_REDIRECT_OUTPUT_BOTH
	WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND
	WORDBREAK_REDIRECT_INPUT_STRING
	WORDBREAK_REDIRECT_HERE_DOC          // << (here-document)
	WORDBREAK_REDIRECT_INPUT_DUPLICATE
	WORDBREAK_REDIRECT_INPUT_OUTPUT
	// https://www.gnu.org/software/bash/manual/html_node/Pipelines.html
	WORDBREAK_PIPE
	WORDBREAK_PIPE_WITH_STDERR
	// Elvish lambda parameter delimiter: | inside {|params| ...}
	WORDBREAK_LAMBDA_PIPE
	// https://www.gnu.org/software/bash/manual/html_node/Lists.html
	WORDBREAK_LIST_ASYNC
	WORDBREAK_LIST_SEQUENTIAL
	WORDBREAK_LIST_AND
	WORDBREAK_LIST_OR
	WORDBREAK_LIST_SEQUENTIAL_DOUBLE // ;; (case terminator)
	WORDBREAK_LIST_FALLTHROUGH       // ;& (case fall-through)
	WORDBREAK_LIST_FALLTHROUGH_RETRY // ;| (zsh case fall-through with retry)
	WORDBREAK_LIST_CASE_NEXT         // ;;& (case next pattern)
	WORDBREAK_LIST_ASYNC_ERRCHECK    // &| (zsh background with error check)
	// COMP_WORDBREAKS
	WORDBREAK_CUSTOM
	// Elvish-specific: output capture delimiters ( and )
	WORDBREAK_OUTPUT_CAPTURE
	// Elvish-specific: list literal / indexing delimiters [ and ]
	WORDBREAK_BRACKET
)

var wordbreakTypes = map[WordbreakType]string{
	WORDBREAK_UNKNOWN:                      "WORDBREAK_UNKNOWN",
	WORDBREAK_REDIRECT_INPUT:               "WORDBREAK_REDIRECT_INPUT",
	WORDBREAK_REDIRECT_OUTPUT:              "WORDBREAK_REDIRECT_OUTPUT",
	WORDBREAK_REDIRECT_OUTPUT_APPEND:       "WORDBREAK_REDIRECT_OUTPUT_APPEND",
	WORDBREAK_REDIRECT_OUTPUT_FORCE:        "WORDBREAK_REDIRECT_OUTPUT_FORCE",
	WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE: "WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE",
	WORDBREAK_REDIRECT_OUTPUT_BOTH:         "WORDBREAK_REDIRECT_OUTPUT_BOTH",
	WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND:  "WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND",
	WORDBREAK_REDIRECT_INPUT_STRING:        "WORDBREAK_REDIRECT_INPUT_STRING",
	WORDBREAK_REDIRECT_HERE_DOC:            "WORDBREAK_REDIRECT_HERE_DOC",
	WORDBREAK_REDIRECT_INPUT_DUPLICATE:     "WORDBREAK_REDIRECT_INPUT_DUPLICATE",
	WORDBREAK_REDIRECT_INPUT_OUTPUT:        "WORDBREAK_REDIRECT_INPUT_OUTPUT",
	WORDBREAK_PIPE:                         "WORDBREAK_PIPE",
	WORDBREAK_PIPE_WITH_STDERR:             "WORDBREAK_PIPE_WITH_STDERR",
	WORDBREAK_LAMBDA_PIPE:                  "WORDBREAK_LAMBDA_PIPE",
	WORDBREAK_LIST_ASYNC:                   "WORDBREAK_LIST_ASYNC",
	WORDBREAK_LIST_SEQUENTIAL:              "WORDBREAK_LIST_SEQUENTIAL",
	WORDBREAK_LIST_AND:                     "WORDBREAK_LIST_AND",
	WORDBREAK_LIST_OR:                      "WORDBREAK_LIST_OR",
	WORDBREAK_LIST_SEQUENTIAL_DOUBLE:       "WORDBREAK_LIST_SEQUENTIAL_DOUBLE",
	WORDBREAK_LIST_FALLTHROUGH:             "WORDBREAK_LIST_FALLTHROUGH",
	WORDBREAK_LIST_FALLTHROUGH_RETRY:       "WORDBREAK_LIST_FALLTHROUGH_RETRY",
	WORDBREAK_LIST_CASE_NEXT:               "WORDBREAK_LIST_CASE_NEXT",
	WORDBREAK_LIST_ASYNC_ERRCHECK:          "WORDBREAK_LIST_ASYNC_ERRCHECK",
	WORDBREAK_CUSTOM:                       "WORDBREAK_CUSTOM",
	WORDBREAK_OUTPUT_CAPTURE:               "WORDBREAK_OUTPUT_CAPTURE",
	WORDBREAK_BRACKET:                      "WORDBREAK_BRACKET",
}

func (w WordbreakType) MarshalJSON() ([]byte, error) {
	return json.Marshal(wordbreakTypes[w])
}

func (w WordbreakType) IsPipelineDelimiter() bool {
	switch w {
	case
		WORDBREAK_PIPE,
		WORDBREAK_PIPE_WITH_STDERR,
		WORDBREAK_LIST_ASYNC,
		WORDBREAK_LIST_SEQUENTIAL,
		WORDBREAK_LIST_AND,
		WORDBREAK_LIST_OR,
		WORDBREAK_LIST_SEQUENTIAL_DOUBLE,
		WORDBREAK_LIST_FALLTHROUGH,
		WORDBREAK_LIST_FALLTHROUGH_RETRY,
		WORDBREAK_LIST_CASE_NEXT,
		WORDBREAK_LIST_ASYNC_ERRCHECK:
		return true
	default:
		return false
	}
}

func (w WordbreakType) IsRedirect() bool {
	switch w {
	case
		WORDBREAK_REDIRECT_INPUT,
		WORDBREAK_REDIRECT_OUTPUT,
		WORDBREAK_REDIRECT_OUTPUT_APPEND,
		WORDBREAK_REDIRECT_OUTPUT_FORCE,
		WORDBREAK_REDIRECT_OUTPUT_APPEND_FORCE,
		WORDBREAK_REDIRECT_OUTPUT_BOTH,
		WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND,
		WORDBREAK_REDIRECT_INPUT_STRING,
		WORDBREAK_REDIRECT_HERE_DOC,
		WORDBREAK_REDIRECT_INPUT_DUPLICATE,
		WORDBREAK_REDIRECT_INPUT_OUTPUT:
		return true
	default:
		return false
	}
}

// bashWordbreakType maps a wordbreak token's RawValue to a WordbreakType
// using the POSIX/bash operator grammar.
func bashWordbreakType(raw string) WordbreakType {
	switch raw {
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	case ">":
		return WORDBREAK_REDIRECT_OUTPUT
	case ">>":
		return WORDBREAK_REDIRECT_OUTPUT_APPEND
	case ">|":
		return WORDBREAK_REDIRECT_OUTPUT_FORCE
	case "&>", ">&":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH
	case "&>>":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH_APPEND
	case "<<":
		return WORDBREAK_REDIRECT_HERE_DOC
	case "<<<":
		return WORDBREAK_REDIRECT_INPUT_STRING
	case "<&":
		return WORDBREAK_REDIRECT_INPUT_DUPLICATE
	case "<>":
		return WORDBREAK_REDIRECT_INPUT_OUTPUT
	case "|":
		return WORDBREAK_PIPE
	case "|&":
		return WORDBREAK_PIPE_WITH_STDERR
	case "&":
		return WORDBREAK_LIST_ASYNC
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	case ";;":
		return WORDBREAK_LIST_SEQUENTIAL_DOUBLE
	case ";;&":
		return WORDBREAK_LIST_CASE_NEXT
	case ";&":
		return WORDBREAK_LIST_FALLTHROUGH
	default:
		// TODO check COMP_WORDBREAKS -> WORDBREAK_OTHER
		return WORDBREAK_UNKNOWN
	}
}

// tcshWordbreakType maps a wordbreak token's RawValue to a WordbreakType
// using tcsh's operator grammar. Tcsh differs from bash in:
//   - >& for combined stdout+stderr redirect (bash uses &>)
//   - |& for pipe with stderr (same as bash)
//   - no <<< here-string, no ;; case operators, no &> redirect
//   - no >| or >>| (tcsh uses >! and >>! at the parser level, not lexer level)
//
// Note: >! and >>! are NOT single operator tokens in tcsh's lexer. The '!'
// is a regular word character (_PUN, not _META). The parser in syn3()
// recognizes > followed by ! as a separate word as the noclobber override.
func tcshWordbreakType(raw string) WordbreakType {
	switch raw {
	case "<":
		return WORDBREAK_REDIRECT_INPUT
	case ">":
		return WORDBREAK_REDIRECT_OUTPUT
	case ">>":
		return WORDBREAK_REDIRECT_OUTPUT_APPEND
	case ">&":
		return WORDBREAK_REDIRECT_OUTPUT_BOTH
	case "<<":
		return WORDBREAK_REDIRECT_HERE_DOC
	case "<&":
		return WORDBREAK_REDIRECT_INPUT_DUPLICATE
	case "<>":
		return WORDBREAK_REDIRECT_INPUT_OUTPUT
	case "|":
		return WORDBREAK_PIPE
	case "|&":
		return WORDBREAK_PIPE_WITH_STDERR
	case "&":
		return WORDBREAK_LIST_ASYNC
	case ";":
		return WORDBREAK_LIST_SEQUENTIAL
	case "&&":
		return WORDBREAK_LIST_AND
	case "||":
		return WORDBREAK_LIST_OR
	default:
		return WORDBREAK_UNKNOWN
	}
}
