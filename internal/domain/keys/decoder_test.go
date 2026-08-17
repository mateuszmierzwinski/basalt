package keys

import "testing"

func TestDecodeControlKeys(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want Kind
	}{
		{"Ctrl+C", []byte{0x03}, KindInterrupt},
		{"Ctrl+D", []byte{0x04}, KindEOF},
		{"Ctrl+H as backspace", []byte{0x08}, KindBackspace},
		{"DEL as backspace", []byte{0x7F}, KindBackspace},
		{"Tab", []byte{0x09}, KindTab},
		{"LF as enter", []byte{0x0A}, KindEnter},
		{"CR as enter", []byte{0x0D}, KindEnter},
		{"Ctrl+L", []byte{0x0C}, KindRedraw},
		{"Ctrl+R", []byte{0x12}, KindToggleMode},
		{"Ctrl+U", []byte{0x15}, KindClearLine},
		{"Ctrl+W", []byte{0x17}, KindDeleteWord},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n, ok := Decode(tc.in)
			if !ok || n != 1 || got.Kind != tc.want {
				t.Fatalf("Decode(%v) = %v, %d, %t", tc.in, got.Kind, n, ok)
			}
		})
	}
}

func TestDecodeEscapeSequences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Kind
		size int
	}{
		{"up", "\x1b[A", KindUp, 3},
		{"down", "\x1b[B", KindDown, 3},
		{"right", "\x1b[C", KindRight, 3},
		{"left", "\x1b[D", KindLeft, 3},
		{"home letter", "\x1b[H", KindHome, 3},
		{"end letter", "\x1b[F", KindEnd, 3},
		{"shift tab", "\x1b[Z", KindBackTab, 3},
		{"ss3 up", "\x1bOA", KindUp, 3},
		{"ss3 home", "\x1bOH", KindHome, 3},
		{"page up", "\x1b[5~", KindPageUp, 4},
		{"page down", "\x1b[6~", KindPageDown, 4},
		{"home numeric", "\x1b[1~", KindHome, 4},
		{"end numeric", "\x1b[4~", KindEnd, 4},
		{"F5", "\x1b[15~", KindRefresh, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n, ok := Decode([]byte(tc.in))
			if !ok || got.Kind != tc.want || n != tc.size {
				t.Fatalf("Decode(%q) = %v, %d, %t; want %v, %d", tc.in, got.Kind, n, ok, tc.want, tc.size)
			}
		})
	}
}

func TestDecodeRunes(t *testing.T) {
	got, n, ok := Decode([]byte("p"))
	if !ok || !got.IsRune() || got.Rune != 'p' || n != 1 {
		t.Fatalf("Decode(p) = %+v, %d, %t", got, n, ok)
	}
	got, n, ok = Decode([]byte("ł"))
	if !ok || !got.IsRune() || got.Rune != 'ł' || n != 2 {
		t.Fatalf("Decode(ł) = %+v, %d, %t", got, n, ok)
	}
}

func TestDecodeReportsIncompleteInput(t *testing.T) {
	for _, in := range []string{"", "\x1b", "\x1b[", "\x1b[1", "\x1bO"} {
		if _, _, ok := Decode([]byte(in)); ok {
			t.Fatalf("Decode(%q) reported complete, want incomplete", in)
		}
	}
}

func TestDecodeIncompleteRune(t *testing.T) {
	if _, _, ok := Decode([]byte{0xC5}); ok {
		t.Fatal("Decode of a partial UTF-8 sequence reported complete")
	}
}

func TestFlushResolvesDanglingEscape(t *testing.T) {
	got, n := Flush([]byte{escape})
	if got.Kind != KindEscape || n != 1 {
		t.Fatalf("Flush(ESC) = %v, %d", got.Kind, n)
	}
}

func TestFlushDiscardsUnterminatedSequence(t *testing.T) {
	got, n := Flush([]byte("\x1b[1;2"))
	if got.Kind != KindUnknown || n != len("\x1b[1;2") {
		t.Fatalf("Flush() = %v, %d — an unterminated sequence must be dropped whole", got.Kind, n)
	}
}

func TestFlushPassesThroughCompleteKeys(t *testing.T) {
	got, n := Flush([]byte("\x1b[A"))
	if got.Kind != KindUp || n != 3 {
		t.Fatalf("Flush() = %v, %d", got.Kind, n)
	}
}

// A hostile terminal must not be able to make the decoder buffer without bound.
func TestDecodeBoundsRunawaySequences(t *testing.T) {
	runaway := append([]byte("\x1b["), make([]byte, 4096)...)
	for i := 2; i < len(runaway); i++ {
		runaway[i] = ';'
	}
	_, n, ok := Decode(runaway)
	if !ok {
		t.Fatal("Decode() should give up on a runaway sequence rather than ask for more")
	}
	if n > maxSequenceLength {
		t.Fatalf("consumed %d bytes, want at most %d", n, maxSequenceLength)
	}
}

// Two Escapes pressed quickly arrive in one read; both must register, or the
// back-out-then-quit sequence silently does nothing.
func TestDecodeDoubleEscapeYieldsTwoEscapes(t *testing.T) {
	buf := []byte{escape, escape}
	first, n, ok := Decode(buf)
	if !ok || first.Kind != KindEscape || n != 1 {
		t.Fatalf("first = %v, %d, %t; want Escape, 1, true", first.Kind, n, ok)
	}
	second, n := Flush(buf[n:])
	if second.Kind != KindEscape || n != 1 {
		t.Fatalf("second = %v, %d; want Escape, 1", second.Kind, n)
	}
}

func TestDecodeEscapeThenArrowStillWorks(t *testing.T) {
	buf := []byte("\x1b\x1b[A")
	first, n, ok := Decode(buf)
	if !ok || first.Kind != KindEscape || n != 1 {
		t.Fatalf("first = %v, %d, %t", first.Kind, n, ok)
	}
	second, m, ok := Decode(buf[n:])
	if !ok || second.Kind != KindUp || m != 3 {
		t.Fatalf("second = %v, %d, %t; want Up, 3", second.Kind, m, ok)
	}
}

func TestDecodeMetaPrefixedKeyIsConsumed(t *testing.T) {
	got, n, ok := Decode([]byte("\x1bx"))
	if !ok || got.Kind != KindUnknown || n != 2 {
		t.Fatalf("Decode(ESC x) = %v, %d, %t — both bytes must be consumed", got.Kind, n, ok)
	}
}

func BenchmarkDecodeRune(b *testing.B) {
	buf := []byte("p")
	for b.Loop() {
		Decode(buf)
	}
}

func BenchmarkDecodeArrow(b *testing.B) {
	buf := []byte("\x1b[A")
	for b.Loop() {
		Decode(buf)
	}
}

func BenchmarkFlush(b *testing.B) {
	buf := []byte{escape}
	for b.Loop() {
		Flush(buf)
	}
}
