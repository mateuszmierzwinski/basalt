package ansi

import (
	"bufio"
	"io"
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// writeBufferSize is sized to hold a full frame on a large terminal, so a
// redraw is one write syscall rather than many.
const writeBufferSize = 32 * 1024

// screen paints frames onto a writer.
type screen struct {
	out    *bufio.Writer
	inAlt  bool
	buffer strings.Builder
}

// New returns a Screen writing to w, returning the interface rather than the
// concrete type (Rule 7.2).
func New(w io.Writer) port.Screen {
	return &screen{out: bufio.NewWriterSize(w, writeBufferSize)}
}

// Enter switches to the alternate screen buffer and hides the cursor.
//
// The alternate buffer is what lets basalt take over the whole terminal and
// hand the operator's scrollback back untouched on exit.
func (s *screen) Enter() error {
	if s.inAlt {
		return nil
	}
	s.inAlt = true
	return s.emit(enterAltScreen + hideCursor)
}

// Leave restores the primary screen buffer and the cursor. It is idempotent so
// it can run from a defer, from a signal handler, and before every child
// process that takes over the terminal.
func (s *screen) Leave() error {
	if !s.inAlt {
		return nil
	}
	s.inAlt = false
	return s.emit(resetStyle + showCursor + leaveAltScreen)
}

// Draw paints a frame.
//
// The whole frame is assembled in memory and written in one call, so the
// terminal never shows a half-painted screen. Every line is already exactly the
// terminal's width, so repainting from the home position overwrites the
// previous frame completely without needing a clear.
func (s *screen) Draw(frame layout.Frame) error {
	s.buffer.Reset()
	s.buffer.WriteString(cursorHome)
	for i, line := range frame.Lines {
		if i > 0 {
			s.buffer.WriteString("\r\n")
		}
		s.writeLine(line)
	}
	return s.emit(s.buffer.String())
}

// writeLine appends one styled line to the frame buffer.
func (s *screen) writeLine(line layout.Line) {
	for _, span := range line.Spans {
		s.writeSpan(span)
	}
}

// writeSpan appends a styled run, resetting afterwards so a style never bleeds
// into the following span.
func (s *screen) writeSpan(span layout.Span) {
	code := codeFor(span.Style)
	if code == "" {
		s.buffer.WriteString(span.Text)
		return
	}
	s.buffer.WriteString(code)
	s.buffer.WriteString(span.Text)
	s.buffer.WriteString(resetStyle)
}

// emit writes a string and flushes it, so what has been drawn is on screen by
// the time Draw returns.
func (s *screen) emit(text string) error {
	if _, err := s.out.WriteString(text); err != nil {
		return err
	}
	return s.out.Flush()
}
