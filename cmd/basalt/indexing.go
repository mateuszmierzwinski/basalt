package main

import (
	"github.com/mateuszmierzwinski/basalt/internal/adapter/bazelquery"
	"github.com/mateuszmierzwinski/basalt/internal/cli"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

// indexerFactory builds the indexer once the preflight has resolved the
// monorepo root and the bazel path.
type indexerFactory struct {
	store  port.FileStore
	runner port.CommandRunner
	clock  port.Clock
}

func newIndexerFactory(store port.FileStore, runner port.CommandRunner, clock port.Clock) cli.IndexerFactory {
	return &indexerFactory{store: store, runner: runner, clock: clock}
}

// For wires the bazel query source to the cache.
func (f *indexerFactory) For(preflight cli.Preflight, options cli.Options) indexer.Indexer {
	return indexer.New(indexer.Config{
		Source: bazelquery.NewClient(
			f.runner,
			preflight.Tools.PathOf(toolset.ToolBazel),
			preflight.Workspace.Root,
		),
		Store:    f.store,
		Clock:    f.clock,
		Root:     preflight.Workspace.Root,
		CacheDir: preflight.Workspace.CacheDir,
		UseCache: !options.NoCache,
	})
}
