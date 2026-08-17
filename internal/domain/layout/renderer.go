package layout

import (
	"fmt"
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/text"
)

// cursorGlyph marks the insertion point. basalt hides the real terminal cursor
// while in the alternate screen, so the caret is drawn rather than positioned.
const cursorGlyph = "█"

type renderer struct{}

// NewRenderer builds the frame renderer, returning the interface (Rule 7.2).
func NewRenderer() Renderer {
	return &renderer{}
}

// Render draws the three-panel frame: targets and actions side by side, the
// search field beneath the target list.
func (r *renderer) Render(view View, width, height int) Frame {
	if TooSmall(width, height) {
		return tooSmallFrame(width, height)
	}
	geometry := ComputeGeometry(width, height, len(view.Toolchain))
	lines := make([]Line, 0, height)
	lines = append(lines, r.topBorder(view, geometry))
	lines = append(lines, r.toolchainLines(view, geometry)...)
	lines = append(lines, r.body(view, geometry)...)
	lines = append(lines, r.searchSeparator(geometry))
	lines = append(lines, r.searchField(view, geometry))
	lines = append(lines, r.bottomBorder(geometry))
	return Frame{Lines: lines}
}

// topBorder carries the pane titles, and the connectors dividing the header
// cells beneath it — so each toolchain's frame is closed at the top by the
// border the whole panel already has.
func (r *renderer) topBorder(view View, geometry Geometry) Line {
	spans := []Span{
		{Text: "┌─", Style: StyleBorder},
		{Text: TitleBar(targetsTitle(view), firstCellWidth(geometry)-1), Style: StyleTitle},
	}
	for _, width := range cellsAfterTheFirst(geometry) {
		spans = append(spans, Span{Text: "┬" + strings.Repeat("─", width), Style: StyleBorder})
	}
	return Line{Spans: append(spans,
		Span{Text: "┬─", Style: StyleBorder},
		Span{Text: TitleBar("TARGET ACTIONS", geometry.ActionsWidth-1), Style: StyleTitle},
		Span{Text: "┐", Style: StyleBorder},
	)}
}

// firstCellWidth is how much of the target column the pane title spans: the
// first header cell when there is a header, the whole column when there is not.
func firstCellWidth(geometry Geometry) int {
	if len(geometry.ToolchainCells) == 0 {
		return geometry.TargetsWidth
	}
	return geometry.ToolchainCells[0]
}

// cellsAfterTheFirst are the header cells the pane title does not already span,
// each of which needs its own connector in the border above it.
func cellsAfterTheFirst(geometry Geometry) []int {
	if len(geometry.ToolchainCells) < 2 {
		return nil
	}
	return geometry.ToolchainCells[1:]
}

// toolchainLines draw the header: one framed cell per compiler the selection
// needs, and the rule that closes them off from the target list.
//
// The frames sit inside the target column so the divider to the action panel
// stays unbroken, and the action panel keeps filling from the top — it loses no
// rows to a header that is not about it.
func (r *renderer) toolchainLines(view View, geometry Geometry) []Line {
	if len(geometry.ToolchainCells) == 0 {
		return nil
	}
	return []Line{
		{Spans: r.toolchainCells(view, geometry)},
		{Spans: r.toolchainRule(view, geometry)},
	}
}

func (r *renderer) toolchainCells(view View, geometry Geometry) []Span {
	spans := []Span{{Text: "│", Style: StyleBorder}}
	for i, width := range geometry.ToolchainCells {
		if i > 0 {
			spans = append(spans, Span{Text: "│", Style: StyleBorder})
		}
		spans = append(spans, segmentSpans(view.Toolchain[i], width)...)
	}
	spans = append(spans, Span{Text: "│", Style: StyleBorder})
	spans = append(spans, r.actionCell(view, geometry, 0)...)
	return append(spans, Span{Text: "│", Style: StyleBorder})
}

// segmentSpans draws one header cell. A toolchain the host cannot satisfy is
// drawn as an error on its own, so the alarm names which compiler is behind
// rather than colouring the whole header.
func segmentSpans(segment ToolchainSegment, width int) []Span {
	style := StyleDim
	if segment.Warn {
		style = StyleError
	}
	return StyledCell(" "+segment.Text, width, -1, -1, style, style)
}

// toolchainRule closes the header cells and reopens the target column as one.
func (r *renderer) toolchainRule(view View, geometry Geometry) []Span {
	rule := "├"
	for i, width := range geometry.ToolchainCells {
		if i > 0 {
			rule += "┴"
		}
		rule += strings.Repeat("─", width)
	}
	spans := []Span{{Text: rule + "┤", Style: StyleBorder}}
	spans = append(spans, r.actionCell(view, geometry, 1)...)
	return append(spans, Span{Text: "│", Style: StyleBorder})
}

func (r *renderer) body(view View, geometry Geometry) []Line {
	lines := make([]Line, 0, geometry.BodyHeight)
	for row := 0; row < geometry.BodyHeight; row++ {
		lines = append(lines, Line{Spans: r.bodyRow(view, geometry, row)})
	}
	return lines
}

func (r *renderer) bodyRow(view View, geometry Geometry, row int) []Span {
	spans := []Span{{Text: "│", Style: StyleBorder}}
	spans = append(spans, r.targetCell(view, geometry, row)...)
	spans = append(spans, Span{Text: "│", Style: StyleBorder})
	spans = append(spans, r.actionCell(view, geometry, row+geometry.ActionRowOffset)...)
	return append(spans, Span{Text: "│", Style: StyleBorder})
}

func (r *renderer) targetCell(view View, geometry Geometry, row int) []Span {
	index := view.RowOffset + row
	if index < 0 || index >= len(view.Rows) {
		return []Span{{Text: strings.Repeat(" ", geometry.TargetsWidth)}}
	}
	entry := view.Rows[index]
	if index == view.RowCursor && !entry.IsGroup {
		// The selection is drawn as one run so the highlight is unbroken; the
		// marker gives up its own colour and takes the selection's.
		return selectedCell(markerPrefix(entry)+entry.Text+suffixOf(entry, geometry),
			geometry.TargetsWidth, selectedStyleFor(entry))
	}
	suffix := suffixOf(entry, geometry)
	body := geometry.TargetsWidth - markerWidth(entry) - text.Width(suffix)
	spans := append(
		markerSpans(entry),
		StyledCell(entry.Text, body, entry.MatchStart, entry.MatchEnd,
			baseStyleFor(entry), StyleMatch)...,
	)
	if suffix == "" {
		return spans
	}
	return append(spans, Span{Text: suffix, Style: StyleDim})
}

// suffixOf returns the row's trailing glyphs, or nothing when the row is too
// narrow to spend cells on them.
//
// Decoration is the first thing to go: a name clipped to make room for an icon
// would be a bad trade, so the suffix is dropped whole rather than shortened.
func suffixOf(entry Row, geometry Geometry) string {
	if entry.Suffix == "" {
		return ""
	}
	suffix := " " + entry.Suffix
	if geometry.TargetsWidth-markerWidth(entry)-text.Width(suffix) < minimumNameCells {
		return ""
	}
	return suffix
}

// minimumNameCells is how much of a name must survive for a decorated row to be
// worth decorating.
const minimumNameCells = 12

// markerPrefix is the marker plus its trailing separator, or nothing when the
// row carries no marker.
func markerPrefix(entry Row) string {
	if entry.Marker == "" {
		return ""
	}
	return entry.Marker + " "
}

func markerWidth(entry Row) int {
	return text.Width(markerPrefix(entry))
}

// markerSpans draws the marker in its own colour, so an unrun target can be
// grey-marked while its name stays legible.
func markerSpans(entry Row) []Span {
	prefix := markerPrefix(entry)
	if prefix == "" {
		return nil
	}
	return []Span{{Text: prefix, Style: entry.MarkerStyle}}
}

// baseStyleFor separates the tree's structure from its contents — a package
// heading is styled distinctly from the targets beneath it — and lets a target
// carry the outcome of its last run.
func baseStyleFor(entry Row) Style {
	if entry.IsGroup {
		return StyleGroup
	}
	return entry.Style
}

// actionCell draws one row of the action panel. The index is the action's own,
// not the target row's: the header pushes the target list down without pushing
// the actions with it.
func (r *renderer) actionCell(view View, geometry Geometry, row int) []Span {
	if row < 0 || row >= len(view.Actions) {
		return []Span{{Text: strings.Repeat(" ", geometry.ActionsWidth)}}
	}
	content := " " + view.Actions[row]
	if row == view.ActionCursor && view.Focus == FocusActions {
		return selectedCell(content, geometry.ActionsWidth, StyleSelected)
	}
	return StyledCell(content, geometry.ActionsWidth, -1, -1, StyleNormal, StyleMatch)
}

func (r *renderer) searchSeparator(geometry Geometry) Line {
	return Line{Spans: []Span{
		{Text: "├─", Style: StyleBorder},
		{Text: TitleBar("SEARCH", geometry.TargetsWidth-1), Style: StyleTitle},
		{Text: "┼", Style: StyleBorder},
		{Text: strings.Repeat("─", geometry.ActionsWidth), Style: StyleBorder},
		{Text: "┤", Style: StyleBorder},
	}}
}

// searchField draws the prompt, the query and — in the action panel's column —
// the status line, which is where a regex error or a cache warning appears.
func (r *renderer) searchField(view View, geometry Geometry) Line {
	prompt := promptFor(view)
	spans := []Span{
		{Text: "│", Style: StyleBorder},
		{Text: prompt, Style: StylePrompt},
	}
	spans = append(spans, r.queryCell(view, geometry, prompt)...)
	spans = append(spans, Span{Text: "│", Style: StyleBorder})
	spans = append(spans, statusSpans(view, geometry)...)
	return Line{Spans: append(spans, Span{Text: "│", Style: StyleBorder})}
}

// queryCell fills the remainder of the target column after the prompt, so the
// prompt and the query together are exactly one column wide.
func (r *renderer) queryCell(view View, geometry Geometry, prompt string) []Span {
	width := maxInt(geometry.TargetsWidth-text.Width(prompt), 1)
	return StyledCell(view.Query+cursorGlyph, width, -1, -1, StyleNormal, StyleMatch)
}

func (r *renderer) bottomBorder(geometry Geometry) Line {
	return Line{Spans: []Span{{
		Text: "└" + strings.Repeat("─", geometry.TargetsWidth) +
			"┴" + strings.Repeat("─", geometry.ActionsWidth) + "┘",
		Style: StyleBorder,
	}}}
}

func statusSpans(view View, geometry Geometry) []Span {
	style := StyleDim
	if view.StatusIsError {
		style = StyleError
	}
	return StyledCell(" "+view.Status, geometry.ActionsWidth, -1, -1, style, style)
}

func selectedCell(content string, width int, style Style) []Span {
	return []Span{{Text: text.PadRight(text.Sanitize(content), width), Style: style}}
}

// selectedStyleFor keeps a target's outcome visible while it is selected. The
// row that just ran is the row under the cursor, so dropping the colour there
// would hide it at the one moment it matters most.
func selectedStyleFor(entry Row) Style {
	switch entry.Style {
	case StyleSuccess:
		return StyleSelectedSuccess
	case StyleFailure:
		return StyleSelectedFailure
	default:
		return StyleSelected
	}
}

func targetsTitle(view View) string {
	return fmt.Sprintf("BAZEL TARGETS %d/%d", view.MatchCount, view.TotalTargets)
}

func promptFor(view View) string {
	if view.ModeName == "" {
		return " > "
	}
	return " " + view.ModeName + " > "
}

// tooSmallFrame is what a terminal below the minimum gets: one honest line
// rather than a mangled layout.
func tooSmallFrame(width, height int) Frame {
	message := fmt.Sprintf("basalt needs at least %dx%d (have %dx%d)", MinWidth, MinHeight, width, height)
	return Frame{Lines: []Line{{Spans: []Span{
		{Text: text.PadRight(message, maxInt(width, 1)), Style: StyleError},
	}}}}
}
