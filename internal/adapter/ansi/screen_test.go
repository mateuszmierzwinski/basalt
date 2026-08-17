package ansi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
)

func TestEnterSwitchesToTheAlternateBuffer(t *testing.T) {
	var out bytes.Buffer
	if err := New(&out).Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, enterAltScreen) || !strings.Contains(got, hideCursor) {
		t.Fatalf("Enter() wrote %q", got)
	}
}

func TestEnterIsIdempotent(t *testing.T) {
	var out bytes.Buffer
	screen := New(&out)
	screen.Enter()
	first := out.Len()
	screen.Enter()
	if out.Len() != first {
		t.Fatal("a second Enter wrote more bytes")
	}
}

func TestLeaveRestoresThePrimaryBuffer(t *testing.T) {
	var out bytes.Buffer
	screen := New(&out)
	screen.Enter()
	out.Reset()

	if err := screen.Leave(); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{resetStyle, showCursor, leaveAltScreen} {
		if !strings.Contains(got, want) {
			t.Fatalf("Leave() wrote %q, missing %q", got, want)
		}
	}
}

func TestLeaveWithoutEnterWritesNothing(t *testing.T) {
	var out bytes.Buffer
	if err := New(&out).Leave(); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("Leave() wrote %q before Enter", out.String())
	}
}

func TestLeaveIsIdempotent(t *testing.T) {
	var out bytes.Buffer
	screen := New(&out)
	screen.Enter()
	screen.Leave()
	after := out.Len()
	screen.Leave()
	if out.Len() != after {
		t.Fatal("a second Leave wrote more bytes")
	}
}

func TestDrawWritesEveryLine(t *testing.T) {
	var out bytes.Buffer
	frame := layout.Frame{Lines: []layout.Line{
		{Spans: []layout.Span{{Text: "first"}}},
		{Spans: []layout.Span{{Text: "second"}}},
	}}
	if err := New(&out).Draw(frame); err != nil {
		t.Fatalf("Draw() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") {
		t.Fatalf("Draw() wrote %q", got)
	}
	if !strings.HasPrefix(got, cursorHome) {
		t.Fatal("Draw() did not start from the home position")
	}
	if strings.Count(got, "\r\n") != 1 {
		t.Fatalf("Draw() wrote %d line breaks for 2 lines", strings.Count(got, "\r\n"))
	}
}

func TestDrawStylesSpansAndResetsAfterEach(t *testing.T) {
	var out bytes.Buffer
	frame := layout.Frame{Lines: []layout.Line{{Spans: []layout.Span{
		{Text: "plain"},
		{Text: "match", Style: layout.StyleMatch},
	}}}}
	New(&out).Draw(frame)

	got := out.String()
	if !strings.Contains(got, codeFor(layout.StyleMatch)+"match"+resetStyle) {
		t.Fatalf("styled span not emitted correctly: %q", got)
	}
}

func TestDrawEmitsNoSequenceForNormalStyle(t *testing.T) {
	var out bytes.Buffer
	frame := layout.Frame{Lines: []layout.Line{{Spans: []layout.Span{{Text: "plain"}}}}}
	New(&out).Draw(frame)

	body := strings.TrimPrefix(out.String(), cursorHome)
	if body != "plain" {
		t.Fatalf("an unstyled span emitted %q, want the bare text", body)
	}
}

// basalt never clears the screen: every frame is full-width and repaints from
// home. Asserting it strictly means a regression that starts emitting ESC[2J
// (which would destroy scrollback) fails the build.
func TestDrawNeverClearsTheScreen(t *testing.T) {
	var out bytes.Buffer
	frame := layout.Frame{Lines: []layout.Line{{Spans: []layout.Span{
		{Text: "row", Style: layout.StyleSelected},
	}}}}
	screen := New(&out)
	screen.Enter()
	screen.Draw(frame)

	for _, forbidden := range []string{"\x1b[2J", "\x1b[3J", "\x1b[J"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("a screen-clearing sequence %q was emitted", forbidden)
		}
	}
}

func TestDrawEmptyFrame(t *testing.T) {
	var out bytes.Buffer
	if err := New(&out).Draw(layout.Frame{}); err != nil {
		t.Fatalf("Draw() error = %v", err)
	}
	if out.String() != cursorHome {
		t.Fatalf("Draw(empty) wrote %q", out.String())
	}
}

// A writer that always fails, to prove errors are propagated rather than eaten.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

var errWrite = &writeError{}

type writeError struct{}

func (*writeError) Error() string { return "write failed" }

func TestDrawPropagatesWriteFailures(t *testing.T) {
	frame := layout.Frame{Lines: []layout.Line{{Spans: []layout.Span{{Text: strings.Repeat("x", 64*1024)}}}}}
	if err := New(failingWriter{}).Draw(frame); err == nil {
		t.Fatal("Draw() error = nil on a failing writer")
	}
}

func BenchmarkDraw(b *testing.B) {
	lines := make([]layout.Line, 0, 40)
	for i := 0; i < 40; i++ {
		lines = append(lines, layout.Line{Spans: []layout.Span{
			{Text: "│", Style: layout.StyleBorder},
			{Text: strings.Repeat("x", 100)},
			{Text: "│", Style: layout.StyleBorder},
		}})
	}
	frame := layout.Frame{Lines: lines}
	screen := New(&bytes.Buffer{})
	b.ResetTimer()
	for b.Loop() {
		screen.Draw(frame)
	}
}
