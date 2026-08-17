package filter

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// MaxQueryLength bounds the search field. Go's regexp is RE2, so there is
	// no catastrophic backtracking to fear, but compiling an unbounded pattern
	// still costs time and memory proportional to its length (Rule 25).
	MaxQueryLength = 512

	// MaxMatches caps the result set. The monorepo has ~1300 targets today;
	// the cap keeps a pathological pattern from making the renderer walk a
	// list far larger than any terminal could show.
	MaxMatches = 5000
)

// CompileQuery turns a query into a compiled pattern, rejecting anything
// oversized before regexp ever sees it.
func CompileQuery(query string, mode Mode) (*regexp.Regexp, error) {
	if len(query) > MaxQueryLength {
		return nil, fmt.Errorf("%w: %d bytes, limit %d", ErrQueryTooLong, len(query), MaxQueryLength)
	}
	if mode == ModeLiteral {
		return nil, nil
	}
	compiled, err := regexp.Compile("(?i)" + query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRegex, err)
	}
	return compiled, nil
}

// LiteralSpan reports where query occurs in subject, case-insensitively, or
// (-1, -1) when it does not.
func LiteralSpan(subject, query string) (int, int) {
	index := strings.Index(strings.ToLower(subject), strings.ToLower(query))
	if index < 0 {
		return -1, -1
	}
	return index, index + len(query)
}

// RegexSpan reports where pattern first matches subject, or (-1, -1).
func RegexSpan(subject string, pattern *regexp.Regexp) (int, int) {
	location := pattern.FindStringIndex(subject)
	if location == nil {
		return -1, -1
	}
	return location[0], location[1]
}
