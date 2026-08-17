package session

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/tui"
	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

// fakeTerminal is a scripted tty: it replays a byte script, then blocks, so a
// session loop can be driven to completion without a real terminal.
type fakeTerminal struct {
	mu       sync.Mutex
	script   []byte
	written  []byte
	raw      bool
	rawCalls int
	restores int
	resized  chan struct{}
	closed   bool
}

func newFakeTerminal(script string) *fakeTerminal {
	return &fakeTerminal{script: []byte(script), resized: make(chan struct{}, 1)}
}

func (f *fakeTerminal) MakeRaw() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.raw = true
	f.rawCalls++
	return nil
}

func (f *fakeTerminal) Restore() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.raw = false
	f.restores++
	return nil
}

func (f *fakeTerminal) Size() (int, int, error) { return 100, 24, nil }

// Read drains the script, then returns zero bytes the way a timed-out read on a
// real tty does.
func (f *fakeTerminal) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.script) == 0 {
		time.Sleep(time.Millisecond)
		return 0, nil
	}
	n := copy(p, f.script)
	f.script = f.script[n:]
	return n, nil
}

func (f *fakeTerminal) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.written = append(f.written, p...)
	return len(p), nil
}

func (f *fakeTerminal) Resized() <-chan struct{} { return f.resized }

func (f *fakeTerminal) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeTerminal) isRaw() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.raw
}

// output is everything basalt wrote straight to the terminal rather than
// through the screen — which is exactly the acknowledgement prompt.
func (f *fakeTerminal) output() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.written)
}

// fakeScreen records the alternate-buffer transitions and the frames drawn.
type fakeScreen struct {
	mu      sync.Mutex
	entered int
	left    int
	inAlt   bool
	frames  int
}

func (f *fakeScreen) Enter() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entered++
	f.inAlt = true
	return nil
}

func (f *fakeScreen) Leave() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.left++
	f.inAlt = false
	return nil
}

func (f *fakeScreen) Draw(layout.Frame) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frames++
	return nil
}

func (f *fakeScreen) counts() (int, int, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entered, f.left, f.inAlt
}

// fakeRunner records what was executed and asserts the terminal state at the
// moment the child ran.
type fakeRunner struct {
	mu       sync.Mutex
	specs    []port.CommandSpec
	err      error
	observe  func()
	rawAtRun []bool
	terminal *fakeTerminal
	screen   *fakeScreen
}

func (f *fakeRunner) RunInteractive(_ context.Context, spec port.CommandSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.specs = append(f.specs, spec)
	if f.terminal != nil {
		f.rawAtRun = append(f.rawAtRun, f.terminal.isRaw())
	}
	if f.observe != nil {
		f.observe()
	}
	return f.err
}

func (f *fakeRunner) calls() []port.CommandSpec {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]port.CommandSpec(nil), f.specs...)
}

// fakeIndexer counts refreshes.
type fakeIndexer struct {
	mu        sync.Mutex
	result    indexer.Result
	err       error
	refreshes int
}

func (f *fakeIndexer) Load(context.Context) (indexer.Result, error) { return f.result, f.err }

func (f *fakeIndexer) Refresh(context.Context) (indexer.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshes++
	return f.result, f.err
}

func (f *fakeIndexer) refreshCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshes
}

func testIndex(t *testing.T) indexer.Result {
	t.Helper()
	tgt, err := target.New(target.KindGoBinary, "//src/go/tools/basalt/cmd/basalt:basalt")
	if err != nil {
		t.Fatalf("target.New: %v", err)
	}
	return indexer.Result{
		Index:  index.Index{Targets: []target.Target{tgt}},
		Source: indexer.SourceCache,
	}
}

type harness struct {
	terminal *fakeTerminal
	screen   *fakeScreen
	runner   *fakeRunner
	index    *fakeIndexer
	history  *fakeRecorder
	session  Session
}

func newHarness(t *testing.T, script string) harness {
	t.Helper()
	terminal := newFakeTerminal(script)
	screen := &fakeScreen{}
	runner := &fakeRunner{terminal: terminal, screen: screen}
	idx := &fakeIndexer{result: testIndex(t)}

	recorder := newFakeRecorder()
	provider := NewProvider(ProviderConfig{
		Registry: action.NewRegistry(),
		Store: fakeStore{
			names: map[string][]string{"/repo/src/go/tools/basalt/cmd/basalt": {"README.md"}},
		},
		Tools:   allTools(),
		Root:    "/repo",
		History: recorder,
	})

	return harness{
		terminal: terminal,
		screen:   screen,
		runner:   runner,
		index:    idx,
		history:  recorder,
		session: New(Config{
			Terminal: terminal,
			Screen:   screen,
			Runner:   runner,
			Indexer:  idx,
			Machine:  tui.NewMachine(filter.NewMatcher(), provider, nil),
			Renderer: layout.NewRenderer(),
			Provider: provider,
			History:  recorder,
			Clock:    fixedClock{},
			Initial:  testIndex(t),
		}),
	}
}

// fixedClock stamps every recorded run with the same instant, so assertions do
// not depend on wall time.
type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC) }

// fakeRecorder is an in-memory run history.
type fakeRecorder struct {
	mu       sync.Mutex
	entries  map[string]runlog.Entry
	logs     map[string]bool
	recordEr error
}

func newFakeRecorder() *fakeRecorder {
	return &fakeRecorder{entries: map[string]runlog.Entry{}, logs: map[string]bool{}}
}

func (f *fakeRecorder) Record(entry runlog.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[entry.Label] = entry
	if entry.LogFile != "" {
		f.logs[entry.Label] = true
	}
	return f.recordEr
}

func (f *fakeRecorder) Statuses() map[string]runlog.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	statuses := make(map[string]runlog.Status, len(f.entries))
	for label, entry := range f.entries {
		statuses[label] = entry.Status
	}
	return statuses
}

func (f *fakeRecorder) LogPathFor(label string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.logs[label] {
		return "", false
	}
	return f.entries[label].LogFile, true
}

func (f *fakeRecorder) PrepareLogPath(label string) string {
	return runlog.LogPath("/repo/basalt_target_cache", label)
}

func (f *fakeRecorder) recorded(label string) (runlog.Entry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entry, ok := f.entries[label]
	return entry, ok
}

func runWithTimeout(t *testing.T, session Session) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- session.Run(ctx) }()
	select {
	case err := <-done:
		return err
	case <-time.After(6 * time.Second):
		t.Fatal("session did not finish")
		return nil
	}
}

func TestRunQuitsOnCtrlC(t *testing.T) {
	h := newHarness(t, "\x03")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

// The terminal must be handed back exactly as it was found.
func TestRunRestoresTheTerminal(t *testing.T) {
	h := newHarness(t, "\x03")
	runWithTimeout(t, h.session)

	if h.terminal.isRaw() {
		t.Fatal("the terminal was left in raw mode")
	}
	entered, left, inAlt := h.screen.counts()
	if inAlt {
		t.Fatal("the alternate screen buffer was not left")
	}
	if entered != left {
		t.Fatalf("Enter called %d times, Leave %d — they must balance", entered, left)
	}
}

func TestRunDrawsAtLeastOneFrame(t *testing.T) {
	h := newHarness(t, "\x03")
	runWithTimeout(t, h.session)
	if _, _, _ = h.screen.counts(); h.screen.frames == 0 {
		t.Fatal("no frame was drawn")
	}
}

func TestTypingFiltersAndQuits(t *testing.T) {
	h := newHarness(t, "basalt\x03")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestEscapeQuitsFromACleanState(t *testing.T) {
	h := newHarness(t, "\x1b")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

// Enter focuses the actions, Enter again runs the first one.
func TestEnterTwiceRunsAnAction(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	calls := h.runner.calls()
	if len(calls) != 1 {
		t.Fatalf("%d commands ran, want 1", len(calls))
	}
	if calls[0].Path != "/usr/bin/bazel" {
		t.Fatalf("ran %q, want bazel", calls[0].Path)
	}
	if len(calls[0].Args) < 2 || calls[0].Args[0] != "run" {
		t.Fatalf("args = %v, want a bazel run", calls[0].Args)
	}
}

// The whole point of the suspend/resume model: while the child runs, basalt
// must not hold the terminal in raw mode or occupy the alternate screen.
func TestActionRunsWithTheTerminalReleased(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	var inAltDuringRun bool
	h.runner.observe = func() {
		_, _, inAltDuringRun = h.screen.counts()
	}
	runWithTimeout(t, h.session)

	if len(h.runner.rawAtRun) != 1 {
		t.Fatalf("child ran %d times", len(h.runner.rawAtRun))
	}
	if h.runner.rawAtRun[0] {
		t.Fatal("the terminal was still in raw mode while the child ran")
	}
	if inAltDuringRun {
		t.Fatal("basalt was still on the alternate screen while the child ran")
	}
}

func TestTerminalIsReacquiredAfterAnAction(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	runWithTimeout(t, h.session)

	entered, left, _ := h.screen.counts()
	if entered < 2 || left < 2 {
		t.Fatalf("Enter/Leave = %d/%d — the screen was not re-entered after the child", entered, left)
	}
}

func TestFailedActionIsReportedNotFatal(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	h.runner.err = errors.New("child exploded")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v — a failed action must not end the session", err)
	}
}

// A bazel action's output would be wiped by the next frame, so basalt holds the
// screen until the operator acknowledges it.
func TestBazelActionPausesForReview(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(h.terminal.output(), "Press Enter") {
		t.Fatalf("no acknowledgement prompt was written; terminal output: %q", h.terminal.output())
	}
	if !strings.Contains(h.terminal.output(), "finished successfully") {
		t.Fatalf("the prompt did not report the outcome: %q", h.terminal.output())
	}
}

func TestPausePromptReportsAFailure(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	h.runner.err = errors.New("build broke")
	runWithTimeout(t, h.session)

	if !strings.Contains(h.terminal.output(), "command failed") {
		t.Fatalf("a failed command was not reported at the prompt: %q", h.terminal.output())
	}
	if !strings.Contains(h.terminal.output(), "build broke") {
		t.Fatalf("the prompt did not carry the reason: %q", h.terminal.output())
	}
}

// The alternate screen is re-entered only after the acknowledgement, so the
// child's output stays readable while the prompt is up.
func TestScreenIsRestoredOnlyAfterAcknowledgement(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	var inAltAtPrompt bool
	h.runner.observe = func() {
		_, _, inAltAtPrompt = h.screen.counts()
	}
	runWithTimeout(t, h.session)

	if inAltAtPrompt {
		t.Fatal("basalt was back on the alternate screen while the child ran")
	}
	if _, _, inAlt := h.screen.counts(); inAlt {
		t.Fatal("the session ended on the alternate screen")
	}
}

// An editor is quit deliberately and leaves nothing to review, so pausing would
// only cost a keystroke. Here the third action (Nano > README.md) is selected.
func TestEditorActionDoesNotPause(t *testing.T) {
	h := newHarness(t, "\r\x1b[B\x1b[B\r\x03")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	calls := h.runner.calls()
	if len(calls) != 1 || !strings.Contains(calls[0].Path, "nano") {
		t.Fatalf("expected the nano action to run, got %+v", calls)
	}
	if strings.Contains(h.terminal.output(), "Press Enter") {
		t.Fatalf("an editor action paused for review: %q", h.terminal.output())
	}
}

// A bazel run is filed against the target so the list can show it as green or
// red, and so its output can be found again later.
func TestBazelRunIsRecordedInTheHistory(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	runWithTimeout(t, h.session)

	entry, ok := h.history.recorded("//src/go/tools/basalt/cmd/basalt:basalt")
	if !ok {
		t.Fatal("the run was not recorded")
	}
	if entry.Status != runlog.StatusPassed {
		t.Fatalf("status = %v, want passed", entry.Status)
	}
	if entry.Action != "bazel.run" {
		t.Fatalf("action = %q", entry.Action)
	}
	if entry.LogFile == "" {
		t.Fatal("no log file was recorded")
	}
}

func TestAFailedRunIsRecordedAsFailed(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	h.runner.err = errors.New("build broke")
	runWithTimeout(t, h.session)

	entry, ok := h.history.recorded("//src/go/tools/basalt/cmd/basalt:basalt")
	if !ok {
		t.Fatal("the run was not recorded")
	}
	if entry.Status != runlog.StatusFailed {
		t.Fatalf("status = %v, want failed", entry.Status)
	}
	if entry.ExitNote != "build broke" {
		t.Fatalf("exit note = %q", entry.ExitNote)
	}
}

// The command must be told where to write its output, or there would be no log
// to review afterwards.
func TestBazelRunCapturesItsOutputToALog(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	runWithTimeout(t, h.session)

	calls := h.runner.calls()
	if len(calls) != 1 {
		t.Fatalf("%d commands ran", len(calls))
	}
	if calls[0].LogPath == "" {
		t.Fatal("the bazel command was not given a log path")
	}
	if !runlog.IsInside(runlog.LogsDirectory("/repo/basalt_target_cache"), calls[0].LogPath) {
		t.Fatalf("log path %q is outside the cache", calls[0].LogPath)
	}
}

// An editor produces nothing worth keeping, so it is neither logged nor filed.
func TestEditorRunIsNotRecorded(t *testing.T) {
	h := newHarness(t, "\r\x1b[B\x1b[B\r\x03")
	runWithTimeout(t, h.session)

	calls := h.runner.calls()
	if len(calls) != 1 || !strings.Contains(calls[0].Path, "nano") {
		t.Fatalf("expected the nano action to run, got %+v", calls)
	}
	if calls[0].LogPath != "" {
		t.Fatalf("an editor command was given a log path: %q", calls[0].LogPath)
	}
	if _, ok := h.history.recorded("//src/go/tools/basalt/cmd/basalt:basalt"); ok {
		t.Fatal("an editor run was filed in the run history")
	}
}

// A history that cannot be written is reported, not fatal: the command ran.
func TestAFailedHistoryWriteIsReportedNotFatal(t *testing.T) {
	h := newHarness(t, "\r\r\r\x03")
	h.history.recordEr = errors.New("read-only filesystem")

	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v — a failed history write must not end the session", err)
	}
}

func TestRefreshKeyReindexes(t *testing.T) {
	h := newHarness(t, "\x1b[15~\x03")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if h.index.refreshCount() != 1 {
		t.Fatalf("refreshes = %d, want 1", h.index.refreshCount())
	}
}

func TestFailedReindexIsNotFatal(t *testing.T) {
	h := newHarness(t, "\x1b[15~\x03")
	h.index.err = errors.New("bazel is unhappy")
	if err := runWithTimeout(t, h.session); err != nil {
		t.Fatalf("Run() error = %v — a failed re-index must not end the session", err)
	}
}

func TestContextCancellationEndsTheSession(t *testing.T) {
	h := newHarness(t, "")
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- h.session.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancellation did not end the session")
	}
	if h.terminal.isRaw() {
		t.Fatal("cancellation left the terminal in raw mode")
	}
}

func TestResizeRedraws(t *testing.T) {
	h := newHarness(t, "")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.session.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	h.terminal.resized <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if h.screen.frames < 2 {
		t.Fatalf("frames = %d, want a redraw after the resize", h.screen.frames)
	}
}

func TestWatcherTriggersReindex(t *testing.T) {
	h := newHarness(t, "")
	changes := make(chan struct{}, 1)
	watcher := &fakeWatcher{changes: changes}

	provider := NewProvider(ProviderConfig{Registry: action.NewRegistry(), Store: fakeStore{}, Tools: allTools()})
	s := New(Config{
		Terminal: h.terminal,
		Screen:   h.screen,
		Runner:   h.runner,
		Watcher:  watcher,
		Indexer:  h.index,
		Machine:  tui.NewMachine(filter.NewMatcher(), provider, nil),
		Renderer: layout.NewRenderer(),
		Provider: provider,
		Initial:  testIndex(t),
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	changes <- struct{}{}
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	if h.index.refreshCount() == 0 {
		t.Fatal("a watcher signal did not trigger a re-index")
	}
}

// A nil watcher must be tolerated: fswatch.New can fail — inotify limits, an
// unreadable tree — and basalt runs without it, just without noticing a BUILD
// change until the next manual re-index. The nil channel it yields must block
// in the select rather than spinning or panicking.
func TestNilWatcherIsTolerated(t *testing.T) {
	h := newHarness(t, "")
	provider := NewProvider(ProviderConfig{Registry: action.NewRegistry(), Store: fakeStore{}, Tools: allTools()})
	s := New(Config{
		Terminal: h.terminal,
		Screen:   h.screen,
		Runner:   h.runner,
		Watcher:  nil,
		Indexer:  h.index,
		Machine:  tui.NewMachine(filter.NewMatcher(), provider, nil),
		Renderer: layout.NewRenderer(),
		Provider: provider,
		Initial:  testIndex(t),
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a nil watcher wedged the event loop")
	}
	if h.index.refreshCount() != 0 {
		t.Fatal("a nil watcher must never trigger a re-index")
	}
}

type fakeWatcher struct {
	changes chan struct{}
	closed  bool
}

func (f *fakeWatcher) Changes() <-chan struct{} { return f.changes }

func (f *fakeWatcher) Close() error {
	f.closed = true
	return nil
}
