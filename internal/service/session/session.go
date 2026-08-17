// Package session runs basalt's interactive loop: draw, wait, react.
package session

import (
	"context"
	"fmt"

	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/keys"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/tui"
	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/service/history"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

const (
	// chromeRows is how many terminal rows the frame spends on borders, the
	// search separator and the search field — the rest is the target viewport.
	chromeRows = 4

	// acknowledgePrompt is shown after a bazel action so its output can be read
	// before basalt paints over it.
	acknowledgePrompt = "Press Enter to return to basalt… "
)

type session struct {
	terminal port.Terminal
	screen   port.Screen
	runner   port.InteractiveRunner
	watcher  port.TreeWatcher
	indexer  indexer.Indexer
	machine  tui.Machine
	renderer Renderer
	provider Actions
	history  history.Recorder
	clock    port.Clock
	initial  indexer.Result
	tests    index.MutableTestIndex
}

// New builds the session, returning the interface (Rule 7.2).
func New(config Config) Session {
	return &session{
		terminal: config.Terminal,
		screen:   config.Screen,
		runner:   config.Runner,
		watcher:  config.Watcher,
		indexer:  config.Indexer,
		machine:  config.Machine,
		renderer: config.Renderer,
		provider: config.Provider,
		history:  config.History,
		clock:    config.Clock,
		initial:  config.Initial,
		tests:    config.Tests,
	}
}

// Run takes over the terminal and loops until the operator quits.
//
// Every acquisition is paired with a deferred release in reverse order, so the
// terminal is handed back intact on any exit path — including a panic, which
// would otherwise leave the operator's shell in raw mode with no echo.
func (s *session) Run(ctx context.Context) error {
	if err := s.terminal.MakeRaw(); err != nil {
		return err
	}
	defer s.terminal.Restore()
	if err := s.screen.Enter(); err != nil {
		return err
	}
	defer s.screen.Leave()

	input := newReader(s.terminal)
	defer input.Close()

	return s.loop(ctx, s.initialState(), input)
}

// initialState seeds the machine with the index built at startup and whatever
// the run history already knows, so previously-run targets are coloured from
// the first frame.
func (s *session) initialState() tui.State {
	s.replaceTests(s.initial.Index)
	return s.machine.Refilter(tui.State{
		Targets:  s.initial.Index.Targets,
		Statuses: s.knownStatuses(),
		Viewport: s.viewport(),
		Status:   fmt.Sprintf("%d targets · %s", len(s.initial.Index.Targets), s.initial.Source),
	})
}

// replaceTests points the action provider at the current target set, so a test
// added since startup is offered without restarting basalt.
func (s *session) replaceTests(current index.Index) {
	if s.tests != nil {
		s.tests.Replace(current)
	}
}

func (s *session) knownStatuses() map[string]runlog.Status {
	if s.history == nil {
		return nil
	}
	return s.history.Statuses()
}

// loop blocks on the three things that can change the screen: a keystroke, a
// resize, and the watcher noticing a BUILD file change.
func (s *session) loop(ctx context.Context, state tui.State, input *reader) error {
	for {
		if err := s.draw(state); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-s.terminal.Resized():
			state = s.resize(state)
		case <-s.changes():
			state = s.reindex(ctx, state, "source tree changed")
		case key, open := <-input.Events():
			if !open {
				return nil
			}
			next, quit, err := s.handle(ctx, state, key, input)
			if err != nil || quit {
				return err
			}
			state = next
		}
	}
}

// changes yields the watcher's channel, or nil when there is no watcher — a nil
// channel blocks forever in a select, which is exactly the right behaviour.
func (s *session) changes() <-chan struct{} {
	if s.watcher == nil {
		return nil
	}
	return s.watcher.Changes()
}

// handle applies one keystroke and carries out whatever it asked for.
func (s *session) handle(ctx context.Context, state tui.State, key keys.Key, input *reader) (tui.State, bool, error) {
	next, outcome := s.machine.Apply(state, key)
	switch outcome {
	case tui.OutcomeQuit:
		return next, true, nil
	case tui.OutcomeRefresh:
		return s.reindex(ctx, next, "re-indexed"), false, nil
	case tui.OutcomeRunAction:
		return s.runAction(ctx, next, input), false, nil
	default:
		return next, false, nil
	}
}

// draw renders the current state at the terminal's present size.
func (s *session) draw(state tui.State) error {
	width, height, err := s.terminal.Size()
	if err != nil {
		return err
	}
	return s.screen.Draw(s.renderer.Render(s.machine.View(state), width, height))
}

// resize recomputes the viewport so scrolling stays correct after a SIGWINCH.
func (s *session) resize(state tui.State) tui.State {
	state.Viewport = s.viewport()
	state.Offset = tui.ScrollOffset(state.Offset, state.Cursor, state.Viewport, len(state.Matches))
	return state
}

// viewport is how many target rows fit on screen.
func (s *session) viewport() int {
	_, height, err := s.terminal.Size()
	if err != nil || height <= chromeRows {
		return 1
	}
	return height - chromeRows
}

// reindex re-queries bazel and replaces the target set, keeping the query and
// the selection. A failed re-index leaves the previous index in place: a stale
// list is far more useful than an empty one.
func (s *session) reindex(ctx context.Context, state tui.State, note string) tui.State {
	result, err := s.indexer.Refresh(ctx)
	if err != nil {
		state.Status = "re-index failed: " + err.Error()
		state.StatusIsError = true
		return state
	}
	state.Targets = result.Index.Targets
	s.replaceTests(result.Index)
	state = s.machine.Refilter(state)
	state.Status = fmt.Sprintf("%s · %d targets", note, len(result.Index.Targets))
	state.StatusIsError = false
	return state
}

// runAction hands the terminal to a child process and takes it back afterwards.
//
// Re-entering the alternate screen is deliberately the last step: until it
// happens the child's output is still on the primary screen, which is what lets
// the acknowledgement prompt be read against the build log it describes.
func (s *session) runAction(ctx context.Context, state tui.State, input *reader) tui.State {
	spec, err := s.commandFor(state)
	if err != nil {
		return failed(state, err)
	}
	reviewable := action.ProducesRunLog(state.Chosen)
	spec.LogPath = s.logPathFor(state, reviewable)

	runErr := s.execute(ctx, spec, input)
	if reviewable {
		s.awaitAcknowledgement(input, runErr)
		state = s.recordRun(state, spec, runErr)
	}
	s.screen.Enter()
	return s.report(state, runErr)
}

// logPathFor is where this run's output should be kept, or empty when the
// action produces nothing worth keeping.
func (s *session) logPathFor(state tui.State, reviewable bool) string {
	selected, ok := state.Selected()
	if !reviewable || !ok || s.history == nil {
		return ""
	}
	return s.history.PrepareLogPath(selected.Label)
}

// recordRun files the outcome against the target and refreshes the colours, so
// a target turns green or red the moment its command finishes rather than at
// the next re-index.
//
// A history that cannot be written is reported in the status line and nothing
// more: the command still ran, and losing the record of it is not worth
// interrupting the operator over.
func (s *session) recordRun(state tui.State, spec port.CommandSpec, runErr error) tui.State {
	selected, ok := state.Selected()
	if !ok || s.history == nil {
		return state
	}
	entry := runlog.Entry{
		Label:    selected.Label,
		Action:   state.Chosen.ID(),
		Status:   runlog.StatusOf(runErr),
		RanAt:    s.clock.Now(),
		LogFile:  spec.LogPath,
		ExitNote: outcomeNote(runErr),
	}
	if err := s.history.Record(entry); err != nil {
		state.Status = "could not save run history: " + err.Error()
		state.StatusIsError = true
	}
	state.Statuses = s.history.Statuses()
	return s.machine.Refilter(state)
}

func outcomeNote(runErr error) string {
	if runErr != nil {
		return runErr.Error()
	}
	return "ok"
}

// execute releases the terminal, runs the child, and takes raw mode back —
// while staying on the primary screen so the child's output remains visible.
//
// The order matters and is reversed on the way out: stop reading input, leave
// the alternate screen, restore cooked mode, run, then re-acquire raw mode
// before the reader resumes. Skip any step and either basalt or the child ends
// up unable to read the keyboard.
func (s *session) execute(ctx context.Context, spec port.CommandSpec, input *reader) error {
	input.Pause()
	defer input.Resume()
	s.screen.Leave()
	s.terminal.Restore()
	defer s.terminal.MakeRaw()
	return s.runner.RunInteractive(ctx, spec)
}

// awaitAcknowledgement holds the child's output on screen until the operator
// presses a key, so a build or test result is not wiped by the next frame.
//
// The key is taken from the input reader rather than read off the terminal
// directly: the reader owns terminal input, and going around it would drop any
// keystroke it had already buffered. Enter is the advertised key, but Escape
// and Ctrl+C dismiss the prompt too — being unable to leave a "press a key"
// prompt is worse than dismissing it early.
func (s *session) awaitAcknowledgement(input *reader, runErr error) {
	fmt.Fprintf(s.terminal, "\r\n%s\r\n%s", outcomeLine(runErr), acknowledgePrompt)
	for key := range input.Events() {
		switch key.Kind {
		case keys.KindEnter, keys.KindEscape, keys.KindInterrupt:
			return
		}
	}
}

// outcomeLine reports how the child ended, so the operator does not have to
// reconstruct it from a screenful of build output.
func outcomeLine(runErr error) string {
	if runErr != nil {
		return "── command failed: " + runErr.Error()
	}
	return "── command finished successfully"
}

// commandFor builds the chosen action's invocation, rebuilding the context so
// the command reflects the files on disk right now rather than at selection.
func (s *session) commandFor(state tui.State) (port.CommandSpec, error) {
	selected, ok := state.Selected()
	if !ok || state.Chosen == nil {
		return port.CommandSpec{}, ErrNothingSelected
	}
	context, err := s.provider.Context(selected)
	if err != nil {
		return port.CommandSpec{}, err
	}
	return state.Chosen.Command(context)
}

func (s *session) report(state tui.State, err error) tui.State {
	if err != nil {
		return failed(state, err)
	}
	state.Status = "action finished"
	state.StatusIsError = false
	return state
}

func failed(state tui.State, err error) tui.State {
	state.Status = err.Error()
	state.StatusIsError = true
	return state
}
