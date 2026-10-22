package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mateuszmierzwinski/basalt/internal/adapter/ansi"
	"github.com/mateuszmierzwinski/basalt/internal/adapter/fswatch"
	"github.com/mateuszmierzwinski/basalt/internal/adapter/termios"
	"github.com/mateuszmierzwinski/basalt/internal/cli"
	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	idx "github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/tui"
	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/service/history"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
	"github.com/mateuszmierzwinski/basalt/internal/service/session"
	"github.com/mateuszmierzwinski/basalt/internal/service/toolchains"
)

// sessionFactory opens the terminal and assembles the interactive session.
type sessionFactory struct {
	store       port.FileStore
	probe       port.FileProbe
	runner      port.InteractiveRunner
	commands    port.CommandRunner
	environment port.Environment
	clock       port.Clock
}

// SessionDeps groups the adapters the interactive session is built from.
type SessionDeps struct {
	Store       port.FileStore
	Probe       port.FileProbe
	Runner      port.InteractiveRunner
	Commands    port.CommandRunner
	Environment port.Environment
	Clock       port.Clock
}

func newSessionFactory(deps SessionDeps) cli.SessionFactory {
	return &sessionFactory{
		store:       deps.Store,
		probe:       deps.Probe,
		runner:      deps.Runner,
		commands:    deps.Commands,
		environment: deps.Environment,
		clock:       deps.Clock,
	}
}

// Start acquires the terminal, starts the source-tree watcher and runs the
// session. Everything acquired here is released before Start returns.
func (f *sessionFactory) Start(ctx context.Context, preflight cli.Preflight, index indexer.Indexer, result indexer.Result) error {
	terminal, err := termios.Open()
	if err != nil {
		return err
	}
	defer func() { _ = terminal.Close() }()

	watcher := f.watch(preflight)
	defer closeWatcher(watcher)

	prober := toolchains.New(ctx, toolchains.Config{
		Environment: f.environment,
		Runner:      f.commands,
		Store:       f.store,
		Root:        preflight.Workspace.Root,
	})
	return session.New(f.config(preflight, index, result, terminal, watcher, f.runHistory(preflight), prober)).Run(ctx)
}

// runHistory loads what previous runs recorded.
//
// A history that cannot be read is reported and then ignored: it degrades to
// "nothing has been run yet", which is never a reason to refuse to start.
func (f *sessionFactory) runHistory(preflight cli.Preflight) history.Recorder {
	recorder, err := history.New(history.Config{
		Store:    f.store,
		Probe:    f.probe,
		CacheDir: preflight.Workspace.CacheDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "basalt: %v\n", err)
	}
	return recorder
}

func (f *sessionFactory) config(
	preflight cli.Preflight,
	index indexer.Indexer,
	result indexer.Result,
	terminal port.Terminal,
	watcher port.TreeWatcher,
	recorder history.Recorder,
	prober toolchains.Prober,
) session.Config {
	tests := idx.NewTestIndex()
	provider := f.provider(preflight, recorder, tests)
	return session.Config{
		Tests:    tests,
		Terminal: terminal,
		Screen:   ansi.New(terminal),
		Runner:   f.runner,
		Watcher:  watcher,
		Indexer:  index,
		Machine:  tui.NewMachine(filter.NewMatcher(), provider, prober),
		Renderer: layout.NewRenderer(),
		Provider: provider,
		History:  recorder,
		Clock:    f.clock,
		Initial:  result,
	}
}

// provider decides what can be done with a target: its rule kind, the files in
// its directory, the tools installed, and whether a previous run left a log.
func (f *sessionFactory) provider(preflight cli.Preflight, recorder history.Recorder, tests idx.TestIndex) session.Actions {
	return session.NewProvider(session.ProviderConfig{
		Registry: action.NewRegistry(),
		Store:    f.store,
		Tools:    preflight.Tools,
		Root:     preflight.Workspace.Root,
		History:  recorder,
		Tests:    tests,
		LocalBin: localBin(),
	})
}

// localBin is the operator's ~/.local/bin, the conventional per-user directory
// on $PATH. Without a home directory there is nowhere to copy to, and the
// action that would is hidden.
func localBin() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "bin")
}

// watch starts the source-tree watcher, tolerating failure: without it basalt
// simply will not notice a BUILD file change until the next manual re-index,
// which is far better than refusing to start.
func (f *sessionFactory) watch(preflight cli.Preflight) port.TreeWatcher {
	watcher, err := fswatch.New(preflight.Workspace.SourceRoot, fswatch.DefaultDebounce)
	if err != nil {
		return nil
	}
	return watcher
}

func closeWatcher(watcher port.TreeWatcher) {
	if watcher != nil {
		_ = watcher.Close()
	}
}
