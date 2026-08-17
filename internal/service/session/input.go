package session

import (
	"sync"

	"github.com/mateuszmierzwinski/basalt/internal/domain/keys"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

const (
	// readChunkSize is a comfortable multiple of the longest escape sequence.
	readChunkSize = 256
	// eventBuffer lets a burst of keystrokes queue while a frame is drawing.
	eventBuffer = 64
)

// reader turns terminal bytes into key events on a channel.
//
// Goroutine ownership (Rule 20.6): newReader spawns exactly one goroutine,
// owned by the reader. It exits when Close closes done, and Close waits for it
// via the WaitGroup before returning. The events channel is closed only after
// the goroutine has stopped, so no send can race with the close.
type reader struct {
	terminal port.Terminal
	events   chan keys.Key
	done     chan struct{}
	stopped  sync.WaitGroup

	// gate is held for the duration of each read. Pause acquires it to
	// guarantee no read is in flight before a child process takes the tty.
	gate sync.Mutex

	closeOnce sync.Once
}

// newReader starts decoding terminal input.
func newReader(terminal port.Terminal) *reader {
	r := &reader{
		terminal: terminal,
		events:   make(chan keys.Key, eventBuffer),
		done:     make(chan struct{}),
	}
	r.stopped.Add(1)
	go r.run()
	return r
}

// Events yields decoded key presses.
func (r *reader) Events() <-chan keys.Key { return r.events }

// Pause blocks until no read is in flight and keeps the reader out of the tty
// until Resume. It is what stops basalt from swallowing the first keystroke
// meant for nano or claude.
func (r *reader) Pause() { r.gate.Lock() }

// Resume returns the reader to the terminal.
func (r *reader) Resume() { r.gate.Unlock() }

// Close stops the reader and waits for its goroutine to finish.
func (r *reader) Close() {
	r.closeOnce.Do(func() {
		close(r.done)
		r.stopped.Wait()
		close(r.events)
	})
}

// run reads and decodes until Close. Bytes that form only part of an escape
// sequence are carried across reads; a read that returns nothing proves the
// sequence will never complete, so the remainder is flushed.
func (r *reader) run() {
	defer r.stopped.Done()
	pending := make([]byte, 0, readChunkSize)
	chunk := make([]byte, readChunkSize)

	for {
		select {
		case <-r.done:
			return
		default:
		}
		count, err := r.read(chunk)
		if err != nil {
			return
		}
		pending = r.consume(append(pending, chunk[:count]...), count == 0)
	}
}

// read takes the gate for the duration of the call, so a paused reader cannot
// be sitting inside read(2) while a child owns the terminal.
func (r *reader) read(chunk []byte) (int, error) {
	r.gate.Lock()
	defer r.gate.Unlock()
	return r.terminal.Read(chunk)
}

// consume decodes every complete key in the buffer and returns what is left.
func (r *reader) consume(pending []byte, timedOut bool) []byte {
	for len(pending) > 0 {
		key, size, ok := keys.Decode(pending)
		if !ok {
			if !timedOut {
				return pending
			}
			key, size = keys.Flush(pending)
		}
		pending = pending[size:]
		if !r.emit(key) {
			return nil
		}
	}
	return pending[:0]
}

// emit queues a key, reporting false once the reader is closing.
func (r *reader) emit(key keys.Key) bool {
	select {
	case r.events <- key:
		return true
	case <-r.done:
		return false
	}
}
