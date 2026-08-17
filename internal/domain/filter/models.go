package filter

import "github.com/mateuszmierzwinski/basalt/internal/domain/target"

// Mode selects how a query is interpreted.
type Mode int

const (
	// ModeLiteral is case-insensitive substring matching — the default,
	// because most searches are just a project name.
	ModeLiteral Mode = iota
	// ModeRegex interprets the query as a regular expression, which the brief
	// asks for explicitly ("supports regex like (.*)falcon(.*)").
	ModeRegex
)

// Match is one target that satisfied a query, with the span of the label that
// matched so the renderer can highlight it.
type Match struct {
	Target target.Target
	Start  int
	End    int
}

// Result is a completed filter pass. Err is set when a regex query does not
// compile, which is an ordinary state while the operator is still typing it —
// the previous matches stay on screen and the error is shown in the search bar.
type Result struct {
	Matches []Match
	Err     error
}
