package tui

import (
	"strings"
	"unicode"

	"github.com/mateuszmierzwinski/basalt/internal/domain/filter"
	"github.com/mateuszmierzwinski/basalt/internal/domain/grouping"
)

// FirstSelectable returns the first row the cursor may rest on, or 0 when the
// list holds nothing selectable.
func FirstSelectable(rows []grouping.Row) int {
	for index, row := range rows {
		if row.IsSelectable() {
			return index
		}
	}
	return 0
}

// LastSelectable returns the last row the cursor may rest on.
func LastSelectable(rows []grouping.Row) int {
	for index := len(rows) - 1; index >= 0; index-- {
		if rows[index].IsSelectable() {
			return index
		}
	}
	return 0
}

// StepSelectable moves the cursor by whole selectable rows, stepping over the
// package headings between them and stopping at the ends rather than wrapping.
func StepSelectable(rows []grouping.Row, from, steps int) int {
	direction := 1
	if steps < 0 {
		direction = -1
	}
	current := from
	for remaining := abs(steps); remaining > 0; remaining-- {
		next := scanSelectable(rows, current+direction, direction)
		if next < 0 {
			break
		}
		current = next
	}
	return current
}

// PageSelectable moves about one screen and lands on the nearest selectable row.
// Paging is about the viewport, so it counts drawn rows rather than targets.
func PageSelectable(rows []grouping.Row, from, viewport, direction int) int {
	if len(rows) == 0 {
		return from
	}
	landing := ClampCursor(from+direction*viewport, len(rows))
	if next := scanSelectable(rows, landing, direction); next >= 0 {
		return next
	}
	if back := scanSelectable(rows, landing, -direction); back >= 0 {
		return back
	}
	return from
}

// EnsureSelectable pulls a cursor off a heading, which is where clamping after
// a re-filter can leave it.
func EnsureSelectable(rows []grouping.Row, cursor int) int {
	if len(rows) == 0 {
		return 0
	}
	bounded := ClampCursor(cursor, len(rows))
	if rows[bounded].IsSelectable() {
		return bounded
	}
	if next := scanSelectable(rows, bounded, 1); next >= 0 {
		return next
	}
	if back := scanSelectable(rows, bounded, -1); back >= 0 {
		return back
	}
	return bounded
}

// scanSelectable walks from index in the given direction, returning the first
// selectable row or -1 when the list runs out.
func scanSelectable(rows []grouping.Row, index, direction int) int {
	for i := index; i >= 0 && i < len(rows); i += direction {
		if rows[i].IsSelectable() {
			return i
		}
	}
	return -1
}

// ScrollAnchor is the row that must stay on screen: the cursor, or the heading
// directly above it, so the operator can always see which package the selection
// belongs to.
func ScrollAnchor(rows []grouping.Row, cursor, viewport int) int {
	if viewport < 2 || cursor <= 0 || cursor > len(rows) {
		return cursor
	}
	if rows[cursor-1].Kind == grouping.KindGroup {
		return cursor - 1
	}
	return cursor
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// ClampCursor keeps a cursor inside a list, returning 0 for an empty one.
func ClampCursor(cursor, length int) int {
	if length <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

// ScrollOffset returns the smallest scroll position that keeps the cursor
// visible in a viewport of the given height.
func ScrollOffset(offset, cursor, viewport, length int) int {
	if viewport <= 0 || length <= 0 {
		return 0
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+viewport {
		offset = cursor - viewport + 1
	}
	return clampOffset(offset, viewport, length)
}

func clampOffset(offset, viewport, length int) int {
	if maximum := length - viewport; offset > maximum {
		offset = maximum
	}
	if offset < 0 {
		return 0
	}
	return offset
}

// DeleteLastRune removes the final character of a query, which is what
// Backspace does.
func DeleteLastRune(query string) string {
	runes := []rune(query)
	if len(runes) == 0 {
		return query
	}
	return string(runes[:len(runes)-1])
}

// DeleteLastWord removes the trailing word and any whitespace before it, which
// is what Ctrl+W does. Bazel labels have no spaces, so the separators that
// matter are the path delimiters.
func DeleteLastWord(query string) string {
	trimmed := strings.TrimRightFunc(query, isSeparator)
	index := strings.LastIndexFunc(trimmed, isSeparator)
	if index < 0 {
		return ""
	}
	return trimmed[:index+1]
}

func isSeparator(r rune) bool {
	return unicode.IsSpace(r) || r == '/' || r == ':' || r == '_' || r == '-'
}

// ClearQuery empties the search field, which is what Ctrl+U does.
func ClearQuery(string) string {
	return ""
}

// ToggleMode switches between literal and regex search.
func ToggleMode(mode filter.Mode) filter.Mode {
	if mode == filter.ModeLiteral {
		return filter.ModeRegex
	}
	return filter.ModeLiteral
}

// ModeName is the label shown in the search prompt.
func ModeName(mode filter.Mode) string {
	if mode == filter.ModeRegex {
		return "regex"
	}
	return "literal"
}

// NextFocus cycles panel focus forward; PreviousFocus cycles it back. With two
// panels they are the same operation, but naming both keeps the call sites
// readable and survives a third panel.
func NextFocus(focus int) int {
	return (focus + 1) % focusCount
}

// PreviousFocus cycles panel focus backwards.
func PreviousFocus(focus int) int {
	return (focus + focusCount - 1) % focusCount
}

// focusCount is how many panels participate in the focus cycle. The search
// field is not one of them: it always receives typing, whichever panel is
// focused, so that narrowing the list never costs a keystroke.
const focusCount = 2
