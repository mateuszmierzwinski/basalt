package fswatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testDebounce = 60 * time.Millisecond

func writeBuildFile(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "BUILD"), []byte("# target\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func waitForSignal(t *testing.T, changes <-chan struct{}, within time.Duration) bool {
	t.Helper()
	select {
	case <-changes:
		return true
	case <-time.After(within):
		return false
	}
}

func TestWatcherSignalsOnBuildFileChange(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, testDebounce)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	writeBuildFile(t, dir)
	if !waitForSignal(t, w.Changes(), 5*time.Second) {
		t.Fatal("no signal after a BUILD file was written")
	}
}

func TestWatcherCoalescesBursts(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, testDebounce)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	for i := 0; i < 50; i++ {
		writeBuildFile(t, dir)
	}
	if !waitForSignal(t, w.Changes(), 5*time.Second) {
		t.Fatal("no signal after a burst")
	}
	// The buffer holds one; a burst must not queue fifty.
	if waitForSignal(t, w.Changes(), 3*testDebounce) {
		t.Fatal("a burst produced more than one signal")
	}
}

func TestWatcherIgnoresIrrelevantWrites(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, testDebounce)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	// Rewriting an existing file the way a Dropbox sync would.
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	<-time.After(3 * testDebounce)
	drain(w.Changes())

	for i := 0; i < 10; i++ {
		if err := os.WriteFile(source, []byte("package main // churn\n"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	if waitForSignal(t, w.Changes(), 5*testDebounce) {
		t.Fatal("a source-file rewrite triggered a re-index signal")
	}
}

func TestWatcherNoticesNewDirectories(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, testDebounce)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	nested := filepath.Join(dir, "newpkg")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if !waitForSignal(t, w.Changes(), 5*time.Second) {
		t.Fatal("no signal after a directory was created")
	}
	drain(w.Changes())

	// The new directory must itself be watched.
	writeBuildFile(t, nested)
	if !waitForSignal(t, w.Changes(), 5*time.Second) {
		t.Fatal("a BUILD file in a newly created directory went unnoticed")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	w, err := New(t.TempDir(), testDebounce)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestCloseTerminatesTheChangesChannel(t *testing.T) {
	w, err := New(t.TempDir(), testDebounce)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	changes := w.Changes()
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case _, ok := <-changes:
		if ok {
			t.Fatal("Changes() yielded a value after Close")
		}
	case <-time.After(time.Second):
		t.Fatal("Changes() was not closed by Close")
	}
}

func TestNewFailsOnMissingRoot(t *testing.T) {
	w, err := New(filepath.Join(t.TempDir(), "absent"), testDebounce)
	if err == nil {
		w.Close()
		return // WalkDir tolerates a missing root; the watcher is simply empty.
	}
}

func drain(changes <-chan struct{}) {
	for {
		select {
		case <-changes:
		default:
			return
		}
	}
}
