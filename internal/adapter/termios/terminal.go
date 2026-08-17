// Package termios implements the Terminal port against a Linux tty.
//
// It talks to the kernel directly through TCGETS/TCSETS/TIOCGWINSZ rather than
// taking a dependency on a terminal library, keeping the third-party graph
// untouched (CLAUDE.md, Tech stack).
package termios

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// DevTTY is the controlling terminal. basalt draws here rather than on stdout,
// so redirecting stdout never corrupts the UI and never mixes a child's output
// into the frame.
const DevTTY = "/dev/tty"

// terminal is a tty under basalt's control.
//
// The mutex guards saved and raw, which the restore path touches from both the
// deferred cleanup and the signal handler.
type terminal struct {
	file    *os.File
	mu      sync.Mutex
	saved   syscall.Termios
	raw     bool
	resized chan struct{}
	signals chan os.Signal
	once    sync.Once
}

// Open acquires the controlling terminal and starts watching for resizes.
//
// It returns the interface rather than the concrete type (Rule 7.2). The caller
// owns the result and must Close it.
func Open() (port.Terminal, error) {
	file, err := os.OpenFile(DevTTY, os.O_RDWR, 0)
	if err != nil {
		return nil, ErrNotATerminal
	}
	t := &terminal{
		file:    file,
		resized: make(chan struct{}, 1),
		signals: make(chan os.Signal, 1),
	}
	if _, err := getTermios(t.fd()); err != nil {
		file.Close()
		return nil, ErrNotATerminal
	}
	t.watchResize()
	return t, nil
}

func (t *terminal) fd() uintptr { return t.file.Fd() }

// MakeRaw disables echo and line buffering, remembering the previous settings
// so Restore can put them back. It is idempotent, which is what lets the
// session call it again after every suspended child process.
func (t *terminal) MakeRaw() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.raw {
		return nil
	}
	saved, err := getTermios(t.fd())
	if err != nil {
		return err
	}
	if err := setTermios(t.fd(), makeRaw(saved)); err != nil {
		return err
	}
	t.saved, t.raw = saved, true
	return nil
}

// Restore returns the terminal to its original mode.
//
// It is idempotent and safe to call from anywhere, because it runs from the
// deferred cleanup, from the signal handler, and before every child process
// that needs the tty in canonical mode.
func (t *terminal) Restore() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.raw {
		return nil
	}
	t.raw = false
	return setTermios(t.fd(), t.saved)
}

// Size returns the terminal's dimensions in cells.
func (t *terminal) Size() (int, int, error) {
	ws, err := getWinsize(t.fd())
	if err != nil {
		return 0, 0, err
	}
	return int(ws.cols), int(ws.rows), nil
}

// Read waits for terminal input, returning (0, nil) when the read times out.
//
// syscall.Read is used rather than os.File.Read because Go maps a zero-byte
// read to io.EOF, and under VMIN=0/VTIME=1 a zero-byte return is an ordinary
// timeout rather than the terminal going away. Treating it as EOF killed the
// input loop a tenth of a second after startup — the UI painted correctly and
// then ignored every key.
//
// EINTR is likewise not a failure: a SIGWINCH arriving mid-read interrupts it,
// and the caller simply reads again.
func (t *terminal) Read(p []byte) (int, error) {
	count, err := syscall.Read(int(t.fd()), p)
	if count < 0 {
		count = 0
	}
	if err == syscall.EINTR || err == syscall.EAGAIN {
		return 0, nil
	}
	return count, err
}

// Write emits bytes to the terminal.
func (t *terminal) Write(p []byte) (int, error) { return t.file.Write(p) }

// Resized reports terminal resizes, coalescing bursts into a single wakeup.
func (t *terminal) Resized() <-chan struct{} { return t.resized }

// watchResize starts the SIGWINCH forwarder.
//
// Goroutine ownership (Rule 20.6): the goroutine is owned by this terminal. It
// ranges over the signal channel and exits when Close stops the notifier and
// closes it. It never blocks, because a full resized channel already means a
// redraw is pending.
func (t *terminal) watchResize() {
	signal.Notify(t.signals, syscall.SIGWINCH)
	go func() {
		for range t.signals {
			select {
			case t.resized <- struct{}{}:
			default:
			}
		}
		close(t.resized)
	}()
}

// Close restores the terminal, stops the resize watcher and releases the tty.
//
// Restoring here as well as in the caller's defer is deliberate belt and
// braces: closing a tty still in raw mode leaves the operator's shell unusable.
func (t *terminal) Close() error {
	err := t.Restore()
	t.once.Do(func() {
		signal.Stop(t.signals)
		close(t.signals)
	})
	if cerr := t.file.Close(); err == nil {
		err = cerr
	}
	return err
}
