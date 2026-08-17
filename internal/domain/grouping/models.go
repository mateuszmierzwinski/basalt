// Package grouping arranges matched targets into a tree-like list: one header
// per Bazel package, with that package's targets indented beneath it.
//
// It is a pure transform over match results — no I/O, no terminal — so the
// whole tree shape is testable by comparing structs.
package grouping

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Kind distinguishes a package header from a target beneath it.
type Kind int

const (
	// KindGroup is a package path. It is a label, not a choice: the cursor
	// steps over it.
	KindGroup Kind = iota
	// KindTarget is a selectable target.
	KindTarget
)

// Row is one line of the target list.
//
// Text is what the renderer draws, already composed and indented, and
// MatchStart/MatchEnd index into that text rather than into the label — so the
// renderer highlights without knowing anything about labels.
type Row struct {
	Kind       Kind
	Text       string
	MatchStart int
	MatchEnd   int
	// Status is how this target's last bazel command went, which is what the
	// renderer colours the row by. Always StatusUnknown for a heading.
	Status runlog.Status
	// Languages are the source languages built in this package, alphabetical.
	// Set on a heading, which is the row that speaks for the whole package.
	Languages []target.Language
	// IsPackaging marks a target that produces a distributable artifact.
	IsPackaging bool
	// Target is meaningful only for KindTarget.
	Target target.Target
}

// IsSelectable reports whether the cursor may rest on this row.
func (r Row) IsSelectable() bool {
	return r.Kind == KindTarget
}
