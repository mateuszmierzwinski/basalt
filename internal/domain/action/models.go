package action

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
)

// Context is everything an action needs to decide whether it applies and to
// build its command. It is a pure data carrier (Rule 7.1).
type Context struct {
	Target target.Target
	// Root is the validated monorepo root; bazel commands run here.
	Root string
	// Directory is the target's package directory, already proven to be inside
	// Root by target.Directory.
	Directory string
	Tools     toolset.Set
	// Markdown lists the .md files present in Directory, which is what the
	// Nano actions expand into.
	Markdown []string
	// LastLog is the output kept from this target's last bazel command, or
	// empty when there is none. It arrives already validated as living inside
	// the cache directory — no action derives it from the label.
	LastLog string
	// PackageHasTests reports whether any test rule lives in the target's
	// package or below it. It is what lets a library or a binary offer to run
	// its project's tests, which is where they almost always live — beside the
	// code rather than in the same rule.
	PackageHasTests bool
	// LocalBin is the operator's ~/.local/bin, where "Copy to ~/.local/bin" puts a
	// binary. Empty when the home directory is unknown, which hides that action.
	LocalBin string
}

// Kind groups actions for display, so the panel can show them in a stable
// order regardless of which are available for a given target.
type Kind int

const (
	KindBazel Kind = iota
	KindEditor
	KindClaude
	KindIDE
)
