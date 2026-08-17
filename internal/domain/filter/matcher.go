package filter

import (
	"regexp"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

type matcher struct{}

// NewMatcher builds the target matcher, returning the interface (Rule 7.2).
func NewMatcher() Matcher {
	return &matcher{}
}

// Filter selects the targets matching query. An empty query matches everything,
// which is the initial state: the operator sees the whole monorepo and narrows
// from there.
//
// A failed compile returns the error with no matches, leaving the caller to
// keep showing the previous result while the operator finishes typing.
func (m *matcher) Filter(targets []target.Target, query string, mode Mode) Result {
	if query == "" {
		return Result{Matches: matchAll(targets)}
	}
	pattern, err := CompileQuery(query, mode)
	if err != nil {
		return Result{Err: err}
	}
	return Result{Matches: m.collect(targets, query, pattern)}
}

func (m *matcher) collect(targets []target.Target, query string, pattern *regexp.Regexp) []Match {
	matches := make([]Match, 0, len(targets))
	for _, candidate := range targets {
		start, end := span(candidate.Label, query, pattern)
		if start < 0 {
			continue
		}
		matches = append(matches, Match{Target: candidate, Start: start, End: end})
		if len(matches) >= MaxMatches {
			break
		}
	}
	return matches
}

// span dispatches to the literal or regex search depending on whether a
// pattern was compiled.
func span(label, query string, pattern *regexp.Regexp) (int, int) {
	if pattern == nil {
		return LiteralSpan(label, query)
	}
	return RegexSpan(label, pattern)
}

func matchAll(targets []target.Target) []Match {
	matches := make([]Match, 0, len(targets))
	for _, candidate := range targets {
		matches = append(matches, Match{Target: candidate, Start: -1, End: -1})
	}
	return matches
}
