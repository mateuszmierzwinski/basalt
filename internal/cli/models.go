package cli

import (
	"io"

	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/domain/workspace"
)

// Options carries the parsed command-line configuration inward. Flags are
// defined in main.go and passed as typed values; nothing below the composition
// root reads the flag package (Rule 21.3).
type Options struct {
	NoCache   bool
	IndexOnly bool
}

// Dependencies is the app's constructor argument, grouped so the composition
// root reads as a list of names rather than six positional parameters.
type Dependencies struct {
	Resolver workspace.Resolver
	Detector toolset.Detector
	Indexers IndexerFactory
	Session  SessionFactory
	Stdout   io.Writer
	Stderr   io.Writer
}

// Preflight is the validated startup state: where the monorepo is and which
// external tools are usable. Everything basalt draws is derived from it.
type Preflight struct {
	Workspace workspace.Workspace
	Tools     toolset.Set
}
