package session

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/tui"
	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/service/history"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

// Config is everything the session needs, grouped so the composition root reads
// as a list of names rather than a dozen positional arguments.
//
// Watcher may be nil: basalt runs perfectly well without a live watcher, it
// just will not notice a BUILD file change until the next re-index. History may
// be nil too, in which case no target shows a previous outcome.
type Config struct {
	Terminal port.Terminal
	Screen   port.Screen
	Runner   port.InteractiveRunner
	Watcher  port.TreeWatcher
	Indexer  indexer.Indexer
	Machine  tui.Machine
	Renderer Renderer
	Provider Actions
	History  history.Recorder
	Clock    port.Clock
	Initial  indexer.Result
	// Tests is the provider's test lookup, held here because the session is
	// what learns of a new target set. It may be nil.
	Tests index.MutableTestIndex
}
