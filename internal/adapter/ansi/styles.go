// Package ansi renders frames as ANSI escape sequences.
//
// This is the only place in basalt that emits escape sequences, and it emits
// them exclusively from its own constant table — never from target names or
// file names, which have been stripped of control characters by domain/text
// before they arrive here.
package ansi

import "github.com/mateuszmierzwinski/basalt/internal/domain/layout"

// Escape sequences basalt uses. Kept as named constants so every byte written
// to the terminal is traceable to a declaration in this file.
const (
	enterAltScreen = "\x1b[?1049h"
	leaveAltScreen = "\x1b[?1049l"
	hideCursor     = "\x1b[?25l"
	showCursor     = "\x1b[?25h"
	cursorHome     = "\x1b[H"
	resetStyle     = "\x1b[0m"
)

// styleCodes maps a layout style to its SGR sequence.
//
// A map rather than a chain of conditionals, so adding a style is a data
// change. StyleNormal is absent on purpose: it needs no sequence, and emitting
// one per span would multiply the bytes written on every redraw.
var styleCodes = map[layout.Style]string{
	layout.StyleBorder:   "\x1b[38;5;240m",
	layout.StyleTitle:    "\x1b[1;38;5;110m",
	layout.StyleIndex:    "\x1b[38;5;244m",
	layout.StyleSelected: "\x1b[1;7m",
	layout.StyleMatch:    "\x1b[1;38;5;214m",
	layout.StylePrompt:   "\x1b[1;38;5;170m",
	layout.StyleError:    "\x1b[38;5;203m",
	layout.StyleDim:      "\x1b[38;5;244m",
	// Package headings are a cool blue against the targets' default
	// foreground, so the tree's structure reads separately from its contents.
	layout.StyleGroup: "\x1b[1;38;5;74m",
	// A target's last run: green for passed, red for failed. Both are distinct
	// from StyleError's red, which marks a message rather than a target.
	layout.StyleSuccess: "\x1b[38;5;114m",
	layout.StyleFailure: "\x1b[38;5;167m",
	// The same outcomes under the cursor: reverse video keeps the row marked as
	// selected while the colour survives.
	layout.StyleSelectedSuccess: "\x1b[1;7;38;5;114m",
	layout.StyleSelectedFailure: "\x1b[1;7;38;5;167m",
}

// codeFor returns the SGR sequence for a style, or "" when none is needed.
func codeFor(s layout.Style) string { return styleCodes[s] }
