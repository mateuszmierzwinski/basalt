package tui

import (
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
)

func TestClampCursor(t *testing.T) {
	cases := []struct {
		name           string
		cursor, length int
		want           int
	}{
		{"inside", 3, 10, 3},
		{"at the end", 9, 10, 9},
		{"past the end", 50, 10, 9},
		{"negative", -5, 10, 0},
		{"empty list", 3, 0, 0},
		{"empty list, negative cursor", -3, 0, 0},
		{"single entry", 7, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClampCursor(tc.cursor, tc.length); got != tc.want {
				t.Fatalf("ClampCursor(%d, %d) = %d, want %d", tc.cursor, tc.length, got, tc.want)
			}
		})
	}
}

func TestScrollOffset(t *testing.T) {
	cases := []struct {
		name                             string
		offset, cursor, viewport, length int
		want                             int
	}{
		{"already visible", 0, 3, 10, 100, 0},
		{"cursor above the window", 20, 5, 10, 100, 5},
		{"cursor below the window", 0, 15, 10, 100, 6},
		{"cursor at the last row", 0, 99, 10, 100, 90},
		{"list shorter than viewport", 0, 2, 10, 5, 0},
		{"empty list", 5, 0, 10, 0, 0},
		{"zero viewport", 5, 3, 0, 100, 0},
		{"offset past the end is pulled back", 95, 50, 10, 100, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScrollOffset(tc.offset, tc.cursor, tc.viewport, tc.length)
			if got != tc.want {
				t.Fatalf("ScrollOffset(%d, %d, %d, %d) = %d, want %d",
					tc.offset, tc.cursor, tc.viewport, tc.length, got, tc.want)
			}
		})
	}
}

// Whatever the inputs, the cursor must end up inside the visible window.
func TestScrollOffsetAlwaysKeepsCursorVisible(t *testing.T) {
	for _, viewport := range []int{1, 5, 20} {
		for _, length := range []int{1, 7, 100} {
			for cursor := 0; cursor < length; cursor++ {
				offset := ScrollOffset(0, cursor, viewport, length)
				if cursor < offset || cursor >= offset+viewport {
					t.Fatalf("cursor %d invisible at offset %d (viewport %d, length %d)",
						cursor, offset, viewport, length)
				}
			}
		}
	}
}

func TestDeleteLastRune(t *testing.T) {
	cases := []struct{ in, want string }{
		{"falcon", "falco"},
		{"p", ""},
		{"", ""},
		{"zażółć", "zażół"},
	}
	for _, tc := range cases {
		if got := DeleteLastRune(tc.in); got != tc.want {
			t.Fatalf("DeleteLastRune(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDeleteLastWord(t *testing.T) {
	cases := []struct {
		name     string
		in, want string
	}{
		{"path segment", "//src/go/tools", "//src/go/"},
		{"trailing slash", "//src/go/", "//src/"},
		{"after a colon", "//src/go:basalt", "//src/go:"},
		{"underscore separated", "falcon_web", "falcon_"},
		{"single word", "falcon", ""},
		{"empty", "", ""},
		{"only separators", "///", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeleteLastWord(tc.in); got != tc.want {
				t.Fatalf("DeleteLastWord(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestClearQuery(t *testing.T) {
	if got := ClearQuery("anything at all"); got != "" {
		t.Fatalf("ClearQuery() = %q", got)
	}
}

func TestToggleMode(t *testing.T) {
	if got := ToggleMode(filter.ModeLiteral); got != filter.ModeRegex {
		t.Fatalf("ToggleMode(literal) = %v", got)
	}
	if got := ToggleMode(filter.ModeRegex); got != filter.ModeLiteral {
		t.Fatalf("ToggleMode(regex) = %v", got)
	}
}

func TestModeName(t *testing.T) {
	if got := ModeName(filter.ModeLiteral); got != "literal" {
		t.Fatalf("ModeName(literal) = %q", got)
	}
	if got := ModeName(filter.ModeRegex); got != "regex" {
		t.Fatalf("ModeName(regex) = %q", got)
	}
}

func TestFocusCycles(t *testing.T) {
	if got := NextFocus(0); got != 1 {
		t.Fatalf("NextFocus(0) = %d", got)
	}
	if got := NextFocus(1); got != 0 {
		t.Fatalf("NextFocus(1) = %d", got)
	}
	if got := PreviousFocus(0); got != 1 {
		t.Fatalf("PreviousFocus(0) = %d", got)
	}
	if got := PreviousFocus(1); got != 0 {
		t.Fatalf("PreviousFocus(1) = %d", got)
	}
}

func BenchmarkClampCursor(b *testing.B) {
	for b.Loop() {
		ClampCursor(500, 1304)
	}
}

func BenchmarkScrollOffset(b *testing.B) {
	for b.Loop() {
		ScrollOffset(10, 500, 30, 1304)
	}
}

func BenchmarkDeleteLastRune(b *testing.B) {
	for b.Loop() {
		DeleteLastRune("//src/go/tools/basalt")
	}
}

func BenchmarkDeleteLastWord(b *testing.B) {
	for b.Loop() {
		DeleteLastWord("//src/go/tools/basalt")
	}
}

func BenchmarkToggleMode(b *testing.B) {
	for b.Loop() {
		ToggleMode(filter.ModeLiteral)
	}
}

func BenchmarkModeName(b *testing.B) {
	for b.Loop() {
		ModeName(filter.ModeRegex)
	}
}

func BenchmarkClearQuery(b *testing.B) {
	for b.Loop() {
		ClearQuery("falcon")
	}
}

func BenchmarkNextFocus(b *testing.B) {
	for b.Loop() {
		NextFocus(0)
	}
}

func BenchmarkPreviousFocus(b *testing.B) {
	for b.Loop() {
		PreviousFocus(0)
	}
}
