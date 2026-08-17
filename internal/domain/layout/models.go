// Package layout turns view state into a styled frame.
//
// It emits no escape sequences and performs no I/O: a Frame is plain text plus
// style tags, and the ansi adapter is the only thing that knows what a style
// looks like on a terminal. That split is what makes the whole UI testable by
// comparing strings.
package layout

// Style names a visual treatment. The renderer never emits colour codes — it
// tags spans, and the adapter resolves them.
type Style int

const (
	StyleNormal Style = iota
	StyleBorder
	StyleTitle
	StyleIndex
	StyleSelected
	StyleMatch
	StylePrompt
	StyleError
	StyleDim
	// StyleGroup is the package path heading a group of targets. It is
	// deliberately distinct from the targets beneath it — telling the tree's
	// structure apart from its contents at a glance is the whole point of
	// grouping the list.
	StyleGroup
	// StyleSuccess and StyleFailure mark a target by how its last bazel
	// command went, so the list doubles as a record of what is currently
	// building and what is not.
	StyleSuccess
	StyleFailure
	// StyleSelectedSuccess and StyleSelectedFailure are the selection combined
	// with an outcome. They exist because the row you just ran is the row the
	// cursor is on: plain reverse video would hide exactly the colour you were
	// waiting to see.
	StyleSelectedSuccess
	StyleSelectedFailure
)

// Span is a run of text sharing one style.
type Span struct {
	Text  string
	Style Style
}

// Line is one terminal row, already exactly the terminal's width.
type Line struct {
	Spans []Span
}

// Frame is a complete screen.
type Frame struct {
	Lines []Line
}

// Focus identifies which panel receives navigation keys.
type Focus int

const (
	FocusTargets Focus = iota
	FocusActions
)

// Markers lead every row, so the tree's shape and each target's last outcome
// are readable without relying on colour.
//
// All four are single-cell glyphs. That is a hard requirement, not a
// preference: text.Width counts runes, so a double-width character — any emoji,
// including the obvious 💀 and 📦 — would occupy two cells while measuring as
// one, and every line in the frame would come out a column short. These are
// East-Asian-Ambiguous, which terminals render single-width.
const (
	// MarkerGroup heads a package. A filled block, as close to Bazel's stacked
	// cube as a single cell gets.
	MarkerGroup = "▦"
	// MarkerPassed, MarkerFailed and MarkerUnrun report a target's last run.
	MarkerPassed = "✓"
	MarkerFailed = "☠"
	MarkerUnrun  = "→"
)

// Language and packaging glyphs. These are emoji and therefore two cells wide —
// which is fine because text.Width resolves them, and only because it does.
const (
	// MarkerGo is the gopher's nearest available relative; MarkerC stands for
	// the native toolchain.
	MarkerGo = "🐹"
	MarkerC  = "🔧"
	// MarkerPackaging marks a target that produces a distributable artifact.
	MarkerPackaging = "📦"
)

// ToolchainSegment is one compiler's cell in the header row. Each is framed
// separately, so two toolchains never run together into one sentence.
type ToolchainSegment struct {
	Text string
	// Warn draws this cell — and only this cell — as an error, for a
	// requirement the host cannot meet.
	Warn bool
}

// Row is one line of the target list, already composed by the domain. The
// renderer draws Marker, then Text, highlights the match span within Text, and
// styles a group heading differently — it needs to know nothing about labels or
// packages to do that.
type Row struct {
	IsGroup bool
	// Marker is the leading glyph, drawn in MarkerStyle. It carries its own
	// indentation, which is what sets a target in from the heading above it.
	Marker      string
	MarkerStyle Style
	Text        string
	// Suffix trails the text — the language glyphs on a heading, the packaging
	// glyph on a target. It is the first thing dropped when the row is too
	// narrow, because the name matters more than the decoration.
	Suffix string
	// Style is the row's base treatment. Zero means the default, so a row that
	// says nothing about itself is drawn normally; a target whose last run
	// passed or failed carries StyleSuccess or StyleFailure.
	Style      Style
	MatchStart int
	MatchEnd   int
}

// View is the snapshot the renderer draws. The renderer never reads live state.
type View struct {
	Rows         []Row
	RowOffset    int
	RowCursor    int
	MatchCount   int
	TotalTargets int

	Actions      []string
	ActionCursor int

	// Toolchain is the header: one framed cell per compiler the selection
	// needs, saying what it asks for and what the host provides. Empty hides
	// the header and gives the rows back to the target list.
	Toolchain []ToolchainSegment

	Query    string
	ModeName string
	Focus    Focus

	Status        string
	StatusIsError bool
}
