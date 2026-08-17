package fswatch

import (
	"time"

	"github.com/fsnotify/fsnotify"
)

// run is the watcher's event loop. It coalesces every relevant event into a
// single signal per debounce window, and re-watches directories as they appear
// so a newly created package is indexed without a restart.
func (w *watcher) run() {
	defer w.stopped.Done()
	timer := newIdleTimer(w.debounce)
	defer timer.Stop()

	pending := false
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.inner.Events:
			if !ok {
				return
			}
			pending = w.handleEvent(event, timer, pending)
		case <-timer.C:
			pending = false
			w.signal()
		case _, ok := <-w.inner.Errors:
			if !ok {
				return
			}
		}
	}
}

// newIdleTimer returns a timer that is already stopped and drained, ready to be
// armed by the first relevant event rather than firing on its own.
func newIdleTimer(debounce time.Duration) *time.Timer {
	timer := time.NewTimer(debounce)
	if !timer.Stop() {
		<-timer.C
	}
	return timer
}

// handleEvent decides whether an event matters and (re)arms the debounce timer.
func (w *watcher) handleEvent(event fsnotify.Event, timer *time.Timer, pending bool) bool {
	w.trackNewDirectory(event)
	if !IsRelevant(event.Name, event.Op) {
		return pending
	}
	if pending && !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(w.debounce)
	return true
}

// trackNewDirectory extends the watch to directories created after startup —
// a new project would otherwise never be noticed.
func (w *watcher) trackNewDirectory(event fsnotify.Event) {
	if !event.Has(fsnotify.Create) || !isDirectory(event.Name) {
		return
	}
	addRecursive(w.inner, event.Name)
}

// signal is a non-blocking send: one pending notification already means "the
// index may be stale", so a second adds nothing.
func (w *watcher) signal() {
	select {
	case w.changes <- struct{}{}:
	default:
	}
}
