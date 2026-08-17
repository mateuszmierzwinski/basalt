// Package text sanitizes and measures strings for terminal display.
//
// Everything basalt renders — target labels, package paths, file names — comes
// from the filesystem or from bazel's output, and a file name may contain any
// byte except NUL and '/'. A target named with a literal ESC would otherwise be
// able to move the cursor, repaint the screen or emit its own escape sequences
// when drawn. Nothing reaches the screen without passing through Sanitize.
package text

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// replacementRune stands in for anything removed, so a stripped control
// character is visible rather than silently shortening the line.
const replacementRune = '?'

// Sanitize strips every control character, replacing each with a visible
// placeholder. C0 (including ESC), C1 and DEL all go, as do invalid UTF-8
// bytes; ordinary printable text passes through untouched.
func Sanitize(s string) string {
	if isClean(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(sanitizeRune(r))
	}
	return b.String()
}

// isClean is the fast path: the overwhelming majority of labels are already
// plain ASCII, and the target list is re-rendered on every keystroke.
func isClean(s string) bool {
	for _, r := range s {
		if sanitizeRune(r) != r {
			return false
		}
	}
	return utf8.ValidString(s)
}

func sanitizeRune(r rune) rune {
	if r == utf8.RuneError || !unicode.IsPrint(r) {
		return replacementRune
	}
	return r
}

// Width is the number of terminal cells a sanitized string occupies.
//
// Counting runes is not enough: an emoji occupies two cells but is one rune, so
// a row containing one would measure a column short and every line in the frame
// would come out misaligned. RuneWidth resolves the wide ranges that basalt can
// actually meet — the status and language markers, and CJK in a file name.
func Width(s string) int {
	cells := 0
	for _, r := range s {
		cells += RuneWidth(r)
	}
	return cells
}

// RuneWidth is how many terminal cells a rune occupies: two for the East-Asian
// Wide and Emoji ranges, one for everything else.
//
// This is deliberately a short table rather than the full Unicode width
// database. It covers what basalt renders — its own markers, and names that
// might carry CJK — and treats anything unknown as single-width, which is the
// safe default for the Latin and symbol text that makes up the rest.
func RuneWidth(r rune) int {
	for _, span := range wideRanges {
		if r >= span.lo && r <= span.hi {
			return 2
		}
	}
	return 1
}

type runeSpan struct{ lo, hi rune }

// wideRanges are the double-width blocks. Note what is deliberately absent:
// U+2600–U+27BF, which holds ✓ (U+2713), ☠ (U+2620) and → (U+2192). Those are
// East-Asian-Ambiguous and render in one cell, which is exactly why the status
// markers were chosen from there.
var wideRanges = []runeSpan{
	{0x1100, 0x115F},   // Hangul Jamo
	{0x2E80, 0x303E},   // CJK radicals, Kangxi, CJK symbols
	{0x3041, 0x33FF},   // Hiragana … CJK compatibility
	{0x3400, 0x4DBF},   // CJK Unified Ideographs Extension A
	{0x4E00, 0x9FFF},   // CJK Unified Ideographs
	{0xA000, 0xA4CF},   // Yi
	{0xAC00, 0xD7A3},   // Hangul syllables
	{0xF900, 0xFAFF},   // CJK compatibility ideographs
	{0xFE30, 0xFE6F},   // CJK compatibility forms, small form variants
	{0xFF00, 0xFF60},   // Fullwidth forms
	{0xFFE0, 0xFFE6},   // Fullwidth signs
	{0x1F300, 0x1F64F}, // Emoji: symbols and pictographs, emoticons
	{0x1F680, 0x1F6FF}, // Emoji: transport and map
	{0x1F900, 0x1F9FF}, // Emoji: supplemental symbols and pictographs
	{0x1FA70, 0x1FAFF}, // Emoji: extended-A
	{0x20000, 0x3FFFD}, // CJK Unified Ideographs Extensions B onwards
}

// Truncate shortens a string to at most width cells, marking the cut with an
// ellipsis so the operator can tell a clipped label from a short one.
//
// The accumulation is by cell rather than by rune, so a double-width character
// is never split across the boundary — half an emoji would corrupt the line it
// sits on and every line after it.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return truncateToCells(s, width-1) + "…"
}

// truncateToCells returns the longest prefix of s that fits in the given cells.
func truncateToCells(s string, cells int) string {
	used := 0
	for index, r := range s {
		size := RuneWidth(r)
		if used+size > cells {
			return s[:index]
		}
		used += size
	}
	return s
}

// PadRight extends a string with spaces to exactly width cells, truncating if
// it is longer. Every rendered line is padded to the terminal width so a redraw
// fully overwrites the previous frame without a screen clear.
func PadRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	truncated := Truncate(s, width)
	if gap := width - Width(truncated); gap > 0 {
		return truncated + strings.Repeat(" ", gap)
	}
	return truncated
}
