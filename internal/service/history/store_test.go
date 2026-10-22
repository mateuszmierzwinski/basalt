package history

import (
	"errors"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
)

const (
	testRoot  = "/repo"
	testCache = "/repo/basalt_target_cache"
	testLabel = "//src/go/tools/basalt/cmd/basalt:basalt"
)

// fakeStore is an in-memory FileStore. The surface the history uses is small
// enough that a fake reads better here than a generated mock.
type fakeStore struct {
	mu       sync.Mutex
	files    map[string][]byte
	writeErr error
	readErr  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{files: map[string][]byte{}}
}

func (f *fakeStore) Read(path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.readErr != nil {
		return nil, f.readErr
	}
	data, ok := f.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

func (f *fakeStore) Write(path string, data []byte, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	f.files[path] = append([]byte(nil), data...)
	return nil
}

func (f *fakeStore) MkdirAll(string, fs.FileMode) error { return nil }
func (f *fakeStore) ListNames(string) ([]string, error) { return nil, nil }

func (f *fakeStore) contents(path string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.files[path]
	return string(data), ok
}

// fakeProbe reports which log files exist.
type fakeProbe struct{ present map[string]bool }

func (f fakeProbe) IsDirectory(string) bool { return true }
func (f fakeProbe) IsFile(path string) bool { return f.present[path] }

func newRecorder(t *testing.T, store *fakeStore, probe fakeProbe) Recorder {
	t.Helper()
	recorder, err := New(Config{Store: store, Probe: probe, CacheDir: testCache})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return recorder
}

func sampleEntry(status runlog.Status) runlog.Entry {
	return runlog.Entry{
		Label:   testLabel,
		Action:  "bazel.build",
		Status:  status,
		RanAt:   time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
		LogFile: runlog.LogPath(testCache, testLabel),
	}
}

func TestNewWithNoHistoryOnDisk(t *testing.T) {
	recorder := newRecorder(t, newFakeStore(), fakeProbe{})
	if got := recorder.Statuses(); len(got) != 0 {
		t.Fatalf("Statuses() = %v, want empty", got)
	}
}

func TestRecordThenReadBack(t *testing.T) {
	store := newFakeStore()
	recorder := newRecorder(t, store, fakeProbe{})

	if err := recorder.Record(sampleEntry(runlog.StatusPassed)); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if got := recorder.Statuses()[testLabel]; got != runlog.StatusPassed {
		t.Fatalf("status = %v, want passed", got)
	}
	if _, written := store.contents(runlog.FilePath(testCache)); !written {
		t.Fatal("the history was not persisted")
	}
}

func TestRecordOverwritesThePreviousOutcome(t *testing.T) {
	recorder := newRecorder(t, newFakeStore(), fakeProbe{})
	recorder.Record(sampleEntry(runlog.StatusFailed))
	recorder.Record(sampleEntry(runlog.StatusPassed))

	if got := recorder.Statuses()[testLabel]; got != runlog.StatusPassed {
		t.Fatalf("status = %v, want the later outcome", got)
	}
}

// A history written now must be loadable by the next basalt.
func TestHistorySurvivesAcrossSessions(t *testing.T) {
	store := newFakeStore()
	first := newRecorder(t, store, fakeProbe{})
	if err := first.Record(sampleEntry(runlog.StatusFailed)); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	second := newRecorder(t, store, fakeProbe{})
	if got := second.Statuses()[testLabel]; got != runlog.StatusFailed {
		t.Fatalf("status after reload = %v, want failed", got)
	}
}

func TestLoadReportsARejectedHistory(t *testing.T) {
	store := newFakeStore()
	_ = store.Write(runlog.FilePath(testCache), []byte("{not json"), 0o644)

	_, err := New(Config{Store: store, Probe: fakeProbe{}, CacheDir: testCache})
	if !errors.Is(err, runlog.ErrCorruptHistory) {
		t.Fatalf("New() error = %v, want ErrCorruptHistory", err)
	}
}

// A rejected history is reported, but the recorder still works — it just starts
// empty, exactly like a fresh checkout.
func TestARejectedHistoryStillYieldsAUsableRecorder(t *testing.T) {
	store := newFakeStore()
	_ = store.Write(runlog.FilePath(testCache), []byte("{not json"), 0o644)

	recorder, _ := New(Config{Store: store, Probe: fakeProbe{}, CacheDir: testCache})
	if recorder == nil {
		t.Fatal("New() returned no recorder alongside the error")
	}
	if len(recorder.Statuses()) != 0 {
		t.Fatal("a rejected history should start empty")
	}
	if err := recorder.Record(sampleEntry(runlog.StatusPassed)); err != nil {
		t.Fatalf("Record() after a rejected load: %v", err)
	}
}

func TestRecordReportsAWriteFailure(t *testing.T) {
	store := newFakeStore()
	store.writeErr = errors.New("read-only filesystem")
	recorder := newRecorder(t, store, fakeProbe{})

	if err := recorder.Record(sampleEntry(runlog.StatusPassed)); err == nil {
		t.Fatal("Record() error = nil on a failing store")
	}
	// The in-memory view still updated, so the UI reflects the run that
	// happened even though it could not be saved.
	if got := recorder.Statuses()[testLabel]; got != runlog.StatusPassed {
		t.Fatalf("status = %v, want the outcome to be visible regardless", got)
	}
}

func TestLogPathForAnExistingLog(t *testing.T) {
	logPath := runlog.LogPath(testCache, testLabel)
	recorder := newRecorder(t, newFakeStore(), fakeProbe{present: map[string]bool{logPath: true}})
	recorder.Record(sampleEntry(runlog.StatusPassed))

	got, ok := recorder.LogPathFor(testLabel)
	if !ok || got != logPath {
		t.Fatalf("LogPathFor() = %q, %t", got, ok)
	}
}

func TestLogPathForATargetNeverRun(t *testing.T) {
	recorder := newRecorder(t, newFakeStore(), fakeProbe{})
	if _, ok := recorder.LogPathFor("//src/go/never:run"); ok {
		t.Fatal("a target that never ran reported a log")
	}
}

// A log deleted between runs must not be offered.
func TestLogPathForADeletedLog(t *testing.T) {
	recorder := newRecorder(t, newFakeStore(), fakeProbe{present: map[string]bool{}})
	recorder.Record(sampleEntry(runlog.StatusPassed))

	if _, ok := recorder.LogPathFor(testLabel); ok {
		t.Fatal("a log that is no longer on disk was offered")
	}
}

// The containment guard is re-asserted at the point of use, because the name
// comes from a JSON file anyone can edit and ends up in a pager.
func TestLogPathForRefusesAPathOutsideTheCache(t *testing.T) {
	forged := "/etc/shadow"
	recorder := newRecorder(t, newFakeStore(), fakeProbe{present: map[string]bool{forged: true}})

	entry := sampleEntry(runlog.StatusPassed)
	entry.LogFile = forged
	recorder.Record(entry)

	if got, ok := recorder.LogPathFor(testLabel); ok {
		t.Fatalf("LogPathFor() = %q — a path outside the cache was offered", got)
	}
}

// The history is synced between machines that mount the checkout at different
// paths, so it records what a log is called, never where it sat.
func TestRecordPersistsTheLogsNameNotItsPath(t *testing.T) {
	store := newFakeStore()
	recorder := newRecorder(t, store, fakeProbe{})
	recorder.Record(sampleEntry(runlog.StatusPassed))

	written, ok := store.contents(runlog.FilePath(testCache))
	if !ok {
		t.Fatal("nothing was written")
	}
	if strings.Contains(written, testCache) {
		t.Fatalf("the history recorded an absolute path:\n%s", written)
	}
	if !strings.Contains(written, runlog.LogFileName(testLabel)) {
		t.Fatalf("the history did not record the log name:\n%s", written)
	}
}

// The other half of that: a history written under one cache directory resolves
// its logs against whichever directory is running now.
func TestLogPathForResolvesAgainstTheRunningCacheDirectory(t *testing.T) {
	store := newFakeStore()
	elsewhere := runlog.History{
		SchemaVersion: runlog.SchemaVersion,
		Entries: map[string]runlog.Entry{
			testLabel: {Label: testLabel, Status: runlog.StatusPassed, LogFile: runlog.LogFileName(testLabel)},
		},
	}
	data, err := runlog.Encode(elsewhere)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	store.files[runlog.FilePath(testCache)] = data

	here := runlog.LogPath(testCache, testLabel)
	recorder := newRecorder(t, store, fakeProbe{present: map[string]bool{here: true}})

	got, ok := recorder.LogPathFor(testLabel)
	if !ok || got != here {
		t.Fatalf("LogPathFor() = %q, %t; want %q resolved under this cache directory", got, ok, here)
	}
}

func TestPrepareLogPathIsInsideTheCache(t *testing.T) {
	recorder := newRecorder(t, newFakeStore(), fakeProbe{})
	path := recorder.PrepareLogPath(testLabel)

	if !runlog.IsInside(runlog.LogsDirectory(testCache), path) {
		t.Fatalf("PrepareLogPath() = %q, outside the logs directory", path)
	}
	if !strings.HasSuffix(path, ".log") {
		t.Fatalf("PrepareLogPath() = %q, want a .log file", path)
	}
}

func TestStatusesIsASnapshot(t *testing.T) {
	recorder := newRecorder(t, newFakeStore(), fakeProbe{})
	recorder.Record(sampleEntry(runlog.StatusPassed))

	snapshot := recorder.Statuses()
	snapshot[testLabel] = runlog.StatusFailed

	if got := recorder.Statuses()[testLabel]; got != runlog.StatusPassed {
		t.Fatal("mutating the returned map changed the recorder's state")
	}
}

func BenchmarkStatuses(b *testing.B) {
	store := newFakeStore()
	recorder, _ := New(Config{Store: store, Probe: fakeProbe{}, CacheDir: testCache})
	for i := 0; i < 500; i++ {
		recorder.Record(runlog.Entry{Label: "//src/go/a:b", Status: runlog.StatusPassed})
	}
	b.ResetTimer()
	for b.Loop() {
		recorder.Statuses()
	}
}

func BenchmarkRecord(b *testing.B) {
	recorder, _ := New(Config{Store: newFakeStore(), Probe: fakeProbe{}, CacheDir: testCache})
	entry := sampleEntry(runlog.StatusPassed)
	b.ResetTimer()
	for b.Loop() {
		recorder.Record(entry)
	}
}
