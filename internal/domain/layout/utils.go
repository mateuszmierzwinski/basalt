package layout

import (
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/text"
)

const (
	// MinWidth and MinHeight are the smallest terminal basalt will draw in.
	// Below either, it says so rather than rendering a corrupt frame.
	MinWidth  = 40
	MinHeight = 8

	// maxActionsWidth caps the action panel so a wide terminal gives the extra
	// space to the labels, which are what the operator is actually reading.
	maxActionsWidth = 32
	minActionsWidth = 18

	// borderCells is the three vertical rules: left edge, divider, right edge.
	borderCells = 3
)

// Geometry is the resolved pane sizing for one frame.
type Geometry struct {
	Width        int
	Height       int
	TargetsWidth int
	ActionsWidth int
	BodyHeight   int
	// ToolchainCells is the inner width of each header cell, in order. Empty
	// when there is no header — either nothing to say, or a target column too
	// narrow to divide.
	ToolchainCells []int
	// ActionRowOffset is how many action rows the header consumed. The action
	// panel keeps filling from the top of the frame while the header pushes
	// the target list down, so the two columns are indexed independently.
	ActionRowOffset int
}

// ComputeGeometry divides the terminal between the two panes. The action panel
// takes a third of the width, clamped, and the target list takes the rest.
//
// segments is how many compilers the header describes. A header costs the
// target list two rows — the framed cells and the rule closing them — which the
// action panel does not pay, because its list is unaffected by what sits above
// the targets.
func ComputeGeometry(width, height, segments int) Geometry {
	geometry := computeGeometry(width, height)
	geometry.ToolchainCells = SegmentWidths(geometry.TargetsWidth, segments)
	if len(geometry.ToolchainCells) == 0 {
		return geometry
	}
	geometry.ActionRowOffset = toolchainRows
	geometry.BodyHeight = maxInt(geometry.BodyHeight-toolchainRows, 1)
	return geometry
}

// toolchainRows is what a header costs the target list: the row of framed cells
// and the rule that closes them.
const toolchainRows = 2

// SegmentWidths divides the target column into one cell per segment, separated
// by a single-cell rule.
//
// It returns nothing when the column cannot hold them, which drops the header
// rather than drawing cells too narrow to read — the target list is the point
// of the pane, and it gets the space back.
func SegmentWidths(total, segments int) []int {
	if segments <= 0 {
		return nil
	}
	inner := total - (segments - 1)
	if inner < segments*minSegmentCells {
		return nil
	}
	widths := make([]int, segments)
	base, extra := inner/segments, inner%segments
	for i := range widths {
		widths[i] = base
		if i < extra {
			widths[i]++
		}
	}
	return widths
}

// minSegmentCells is the narrowest a header cell may be and still say anything.
// "🐹 go 1.2 → 1.3" is 15; below that the versions are the first thing lost, so
// there is no point spending the rows.
const minSegmentCells = 15

func computeGeometry(width, height int) Geometry {
	actions := clamp(width/3, minActionsWidth, maxActionsWidth)
	if actions > width-borderCells-minActionsWidth {
		actions = maxInt(width-borderCells-minActionsWidth, minActionsWidth)
	}
	return Geometry{
		Width:        width,
		Height:       height,
		ActionsWidth: actions,
		TargetsWidth: maxInt(width-borderCells-actions, 1),
		// Four rows are chrome: top border, search separator, search, bottom.
		BodyHeight: maxInt(height-4, 1),
	}
}

// TooSmall reports whether the terminal cannot hold a usable frame.
func TooSmall(width, height int) bool {
	return width < MinWidth || height < MinHeight
}

// StyledCell renders text into an exactly-width run of spans, optionally
// highlighting a byte range.
func StyledCell(content string, width, start, end int, base, accent Style) []Span {
	clean := text.Sanitize(content)
	padded := text.PadRight(clean, width)
	if !highlightable(content, clean, width, start, end) {
		return []Span{{Text: padded, Style: base}}
	}
	return []Span{
		{Text: padded[:start], Style: base},
		{Text: padded[start:end], Style: accent},
		{Text: padded[end:], Style: base},
	}
}

// highlightable reports whether a byte range can be applied to the padded cell.
//
// It requires both that sanitizing left the bytes untouched and that the cell
// was not truncated, because either one invalidates an offset measured against
// the original string — and slicing a truncated cell at a stale byte offset
// cuts a multi-byte rune in half, which corrupts the whole frame. Dropping the
// highlight is the right trade: the row is still legible, just unhighlighted.
func highlightable(content, clean string, width, start, end int) bool {
	if clean != content || start < 0 || end <= start {
		return false
	}
	return end <= len(clean) && text.Width(clean) <= width
}

// TitleBar builds a bordered heading such as "─[BAZEL TARGETS 3/1304]───".
func TitleBar(title string, width int) string {
	label := "[" + title + "]"
	if text.Width(label) >= width {
		return text.Truncate(label, width)
	}
	return label + strings.Repeat("─", width-text.Width(label))
}

func clamp(value, low, high int) int {
	return maxInt(low, minInt(value, high))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
