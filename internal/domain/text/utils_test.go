package text

import (
	"strings"
	"testing"
)

func TestSanitizePassesCleanText(t *testing.T) {
	for _, s := range []string{"", "falcon", "//src/go/tools/basalt:basalt", "[probe:probe]", "zażółć"} {
		if got := Sanitize(s); got != s {
			t.Fatalf("Sanitize(%q) = %q, want unchanged", s, got)
		}
	}
}

// The attack this exists to stop: a target or file name carrying escape
// sequences that would repaint the operator's screen when listed.
func TestSanitizeStripsEscapeSequences(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"raw ESC", "evil\x1b[2Jtarget"},
		{"alternate screen switch", "\x1b[?1049h"},
		{"cursor move", "a\x1b[10;10Hb"},
		{"OSC title set", "\x1b]0;pwned\x07"},
		{"carriage return overwrite", "safe\rmalicious"},
		{"bell", "ding\x07"},
		{"backspace overwrite", "abc\x08\x08\x08xyz"},
		{"C1 CSI", "ab"},
		{"NUL", "a\x00b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Sanitize(tc.in)
			if strings.ContainsRune(got, 0x1B) {
				t.Fatalf("Sanitize(%q) = %q — ESC survived", tc.in, got)
			}
			for _, r := range got {
				if r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F) {
					t.Fatalf("Sanitize(%q) = %q — control rune %U survived", tc.in, got, r)
				}
			}
		})
	}
}

func TestSanitizePreservesLength(t *testing.T) {
	got := Sanitize("a\x1bb")
	if Width(got) != 3 {
		t.Fatalf("Sanitize() = %q, width %d, want 3 — a stripped rune must be replaced", got, Width(got))
	}
}

func TestSanitizeHandlesInvalidUTF8(t *testing.T) {
	got := Sanitize(string([]byte{'a', 0xFF, 0xFE, 'b'}))
	if strings.ContainsAny(got, "\xff\xfe") {
		t.Fatalf("Sanitize() = %q — invalid bytes survived", got)
	}
}

func TestWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "abc", 3},
		{"latin with diacritics", "zażółć", 6},
		// The status markers must measure one cell each — the whole frame
		// layout rests on it.
		{"tick", "✓", 1},
		{"skull", "☠", 1},
		{"arrow", "→", 1},
		{"block", "▦", 1},
		{"ellipsis", "…", 1},
		// Emoji are two cells despite being one rune.
		{"hamster", "🐹", 2},
		{"package", "📦", 2},
		{"wrench", "🔧", 2},
		{"emoji in a sentence", "a🐹b", 4},
		// CJK, which a file name could carry.
		{"cjk", "中文", 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Width(tc.in); got != tc.want {
				t.Fatalf("Width(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// A double-width rune must never be split by truncation: half an emoji corrupts
// the line it is on and every line after it.
func TestTruncateNeverSplitsAWideRune(t *testing.T) {
	for _, in := range []string{"🐹🐹🐹", "a🐹b🐹c", "中文中文", "📦 basalt_rpm"} {
		for width := 0; width <= Width(in)+2; width++ {
			got := Truncate(in, width)
			if Width(got) > width {
				t.Fatalf("Truncate(%q, %d) = %q, width %d — exceeds the cell budget",
					in, width, got, Width(got))
			}
		}
	}
}

func TestPadRightIsExactWithWideRunes(t *testing.T) {
	for _, in := range []string{"🐹 go", "📦 rpm", "中文", "✓ plain"} {
		for width := 1; width < 20; width++ {
			if got := Width(PadRight(in, width)); got != width {
				t.Fatalf("PadRight(%q, %d) has width %d", in, width, got)
			}
		}
	}
}

func TestRuneWidth(t *testing.T) {
	for r, want := range map[rune]int{
		'a': 1, '·': 1, '▦': 1, '✓': 1, '☠': 1, '→': 1, '…': 1, 'ł': 1,
		'🐹': 2, '📦': 2, '🔧': 2, '中': 2,
	} {
		if got := RuneWidth(r); got != want {
			t.Fatalf("RuneWidth(%q) = %d, want %d", r, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"fits exactly", "abcde", 5, "abcde"},
		{"shorter than width", "abc", 5, "abc"},
		{"needs cutting", "abcdefgh", 5, "abcd…"},
		{"width one", "abcdef", 1, "…"},
		{"width zero", "abcdef", 0, ""},
		{"negative width", "abcdef", -3, ""},
		{"empty input", "", 5, ""},
		{"multibyte cut", "zażółć", 4, "zaż…"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Truncate(tc.in, tc.width); got != tc.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

func TestTruncateNeverExceedsWidth(t *testing.T) {
	for width := 0; width < 12; width++ {
		if got := Width(Truncate("//src/go/tools/basalt:basalt", width)); got > width {
			t.Fatalf("Truncate(_, %d) produced width %d", width, got)
		}
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"pads", "ab", 5, "ab   "},
		{"exact", "abcde", 5, "abcde"},
		{"truncates", "abcdefgh", 5, "abcd…"},
		{"zero width", "abc", 0, ""},
		{"empty input", "", 3, "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PadRight(tc.in, tc.width); got != tc.want {
				t.Fatalf("PadRight(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

func TestPadRightAlwaysExactWidth(t *testing.T) {
	for width := 1; width < 20; width++ {
		for _, in := range []string{"", "a", "abcdefghijklmnop", "zażółć"} {
			if got := Width(PadRight(in, width)); got != width {
				t.Fatalf("PadRight(%q, %d) has width %d", in, width, got)
			}
		}
	}
}

func BenchmarkSanitizeClean(b *testing.B) {
	for b.Loop() {
		Sanitize("//src/go/tools/basalt/cmd/basalt:basalt")
	}
}

func BenchmarkSanitizeDirty(b *testing.B) {
	for b.Loop() {
		Sanitize("//src/go/\x1b[2Jtools:evil")
	}
}

func BenchmarkWidth(b *testing.B) {
	for b.Loop() {
		Width("//src/go/tools/basalt/cmd/basalt:basalt")
	}
}

func BenchmarkTruncate(b *testing.B) {
	for b.Loop() {
		Truncate("//src/go/tools/basalt/cmd/basalt:basalt", 30)
	}
}

func BenchmarkPadRight(b *testing.B) {
	for b.Loop() {
		PadRight("//src/go/tools/basalt:basalt", 80)
	}
}
