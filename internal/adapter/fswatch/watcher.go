// Package fswatch watches the source tree for changes that invalidate the
// target index.
package fswatch

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// DefaultDebounce coalesces bursts. This tree is synced by Dropbox, which
// rewrites files in storms unrelated to real BUILD edits; without this every
// sync would trigger a re-index.
const DefaultDebounce = 2 * time.Second

type watcher struct {
	inner    *fsnotify.Watcher
	changes  chan struct{}
	debounce time.Duration

	mu      sync.Mutex // protects closed
	closed  bool
	done    chan struct{}
	stopped sync.WaitGroup
}

// New starts watching root recursively and returns a TreeWatcher that emits at
// most one signal per debounce window.
//
// Ownership: New spawns one goroutine that owns the fsnotify event loop. It
// terminates when Close is called, which closes done, waits for the goroutine
// via stopped, and only then closes the changes channel.
func New(root string, debounce time.Duration) (port.TreeWatcher, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := addRecursive(inner, root); err != nil {
		_ = inner.Close()
		return nil, err
	}
	w := &watcher{
		inner:    inner,
		changes:  make(chan struct{}, 1),
		debounce: debounce,
		done:     make(chan struct{}),
	}
	w.stopped.Add(1)
	go w.run()
	return w, nil
}

// Changes yields a signal when the index may be out of date. The channel has a
// buffer of one and a non-blocking send: a consumer busy re-indexing misses
// nothing, because one pending signal means the same thing as ten.
func (w *watcher) Changes() <-chan struct{} {
	return w.changes
}

// Close stops the watcher and releases its inotify descriptors. It is safe to
// call more than once.
func (w *watcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.done)
	w.mu.Unlock()

	err := w.inner.Close()
	w.stopped.Wait()
	close(w.changes)
	return err
}
