package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/domain/workspace"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

const (
	exitOK      = 0
	exitFailure = 1
)

type app struct {
	resolver workspace.Resolver
	detector toolset.Detector
	indexers IndexerFactory
	session  SessionFactory
	stdout   io.Writer
	stderr   io.Writer
}

// NewApp wires the startup path, returning the interface (Rule 7.2).
func NewApp(deps Dependencies) App {
	return &app{
		resolver: deps.Resolver,
		detector: deps.Detector,
		indexers: deps.Indexers,
		session:  deps.Session,
		stdout:   deps.Stdout,
		stderr:   deps.Stderr,
	}
}

// Run performs the startup preflight, builds the target index and hands over to
// the interactive session.
func (a *app) Run(ctx context.Context, options Options) int {
	preflight, err := a.preflight()
	if err != nil {
		return a.fail(err)
	}
	index := a.indexers.For(preflight, options)
	result, err := index.Load(ctx)
	if err != nil {
		return a.fail(err)
	}
	a.reportCacheProblem(result)
	return a.dispatch(ctx, preflight, index, result, options)
}

func (a *app) dispatch(ctx context.Context, preflight Preflight, index indexer.Indexer, result indexer.Result, options Options) int {
	if options.IndexOnly {
		a.reportIndex(preflight, result)
		return exitOK
	}
	if err := a.session.Start(ctx, preflight, index, result); err != nil {
		return a.fail(err)
	}
	return exitOK
}

func (a *app) preflight() (Preflight, error) {
	resolved, err := a.resolver.Resolve()
	if err != nil {
		return Preflight{}, err
	}
	tools, err := a.detector.Detect()
	if err != nil {
		return Preflight{}, err
	}
	return Preflight{Workspace: resolved, Tools: tools}, nil
}

// reportCacheProblem surfaces a rejected or unwritable cache. It is not a
// failure — basalt recovers by re-indexing — but a cache silently discarded on
// every run is a bug nobody would ever notice.
func (a *app) reportCacheProblem(result indexer.Result) {
	if result.CacheError != nil {
		fmt.Fprintf(a.stderr, "basalt: %v\n", result.CacheError)
	}
}

func (a *app) reportIndex(preflight Preflight, result indexer.Result) {
	fmt.Fprintf(a.stdout, "monorepo:  %s\n", preflight.Workspace.Root)
	fmt.Fprintf(a.stdout, "cache:     %s\n", preflight.Workspace.CacheDir)
	fmt.Fprintf(a.stdout, "targets:   %d (from %s)\n", len(result.Index.Targets), result.Source)
	a.reportMissingTools(preflight.Tools)
}

func (a *app) reportMissingTools(tools toolset.Set) {
	for _, tool := range tools.Missing() {
		fmt.Fprintf(a.stdout, "tools:     %s not found — its actions are hidden\n", tool)
	}
}

func (a *app) fail(err error) int {
	fmt.Fprintf(a.stderr, "basalt: %v\n", err)
	return exitFailure
}
