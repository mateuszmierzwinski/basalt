package session

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/service/history"
)

// provider builds the action list for a target, and rebuilds the context an
// action needs when it is actually run. It needs the filesystem to see which
// Markdown files a package contains, which is why the state machine takes it as
// an interface rather than reaching for disk itself.
type provider struct {
	registry action.Registry
	store    port.FileStore
	tools    toolset.Set
	root     string
	history  history.Recorder
	tests    index.TestIndex
}

// ProviderConfig groups the provider's dependencies. History may be nil, in
// which case no target offers its last run log.
type ProviderConfig struct {
	Registry action.Registry
	Store    port.FileStore
	Tools    toolset.Set
	Root     string
	History  history.Recorder
	// Tests decides whether a non-test target offers to run its project's
	// suite. It may be nil, in which case none does.
	Tests index.TestIndex
}

// NewProvider builds the actions provider, returning the interface (Rule 7.2).
func NewProvider(config ProviderConfig) Actions {
	return &provider{
		registry: config.Registry,
		store:    config.Store,
		tools:    config.Tools,
		root:     config.Root,
		history:  config.History,
		tests:    config.Tests,
	}
}

// Actions returns what can be done with a target. A target whose label does not
// resolve to a directory inside the monorepo gets no actions at all, so a
// poisoned cache entry that somehow survived validation still cannot launch
// anything.
func (p *provider) Actions(selected target.Target) []action.Action {
	context, err := p.Context(selected)
	if err != nil {
		return nil
	}
	return p.registry.For(context)
}

// Context resolves the target's directory and the files in it. It is the single
// place a label becomes a filesystem path, and it refuses any label that would
// escape the monorepo.
func (p *provider) Context(selected target.Target) (action.Context, error) {
	directory, err := target.Directory(p.root, selected)
	if err != nil {
		return action.Context{}, err
	}
	return action.Context{
		Target:          selected,
		Root:            p.root,
		Directory:       directory,
		Tools:           p.tools,
		Markdown:        p.markdown(directory),
		LastLog:         p.lastLog(selected.Label),
		PackageHasTests: p.hasTests(selected.Package),
	}, nil
}

// hasTests reports whether anything under the target's package is testable.
// Without a test index nothing is, which is the right default: offering an
// action that would find no tests is worse than not offering it.
func (p *provider) hasTests(pkg string) bool {
	return p.tests != nil && p.tests.HasTestsUnder(pkg)
}

// lastLog is the output kept from the target's last bazel command, or empty
// when there is none. The history service is what proves the path is inside the
// cache directory and still on disk.
func (p *provider) lastLog(label string) string {
	if p.history == nil {
		return ""
	}
	path, ok := p.history.LogPathFor(label)
	if !ok {
		return ""
	}
	return path
}

// markdown lists the Markdown files in a package directory. A directory that
// cannot be read simply has none — an unreadable package is not worth
// interrupting the operator for.
func (p *provider) markdown(directory string) []string {
	names, err := p.store.ListNames(directory)
	if err != nil {
		return nil
	}
	return action.SelectMarkdown(names)
}
