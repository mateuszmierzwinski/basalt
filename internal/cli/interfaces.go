package cli

import (
	"context"

	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

// App is the entry point below main. It returns a process exit code rather
// than calling os.Exit, so the whole startup path stays testable.
type App interface {
	Run(ctx context.Context, options Options) int
}

// IndexerFactory builds an indexer once the workspace is known. The indexer
// needs the resolved root and the bazel path, neither of which exists until the
// preflight has run — hence a factory rather than a constructed value.
type IndexerFactory interface {
	For(preflight Preflight, options Options) indexer.Indexer
}

// SessionFactory starts the interactive session. It is an interface so the
// startup path can be tested without a terminal.
type SessionFactory interface {
	Start(ctx context.Context, preflight Preflight, index indexer.Indexer, result indexer.Result) error
}
