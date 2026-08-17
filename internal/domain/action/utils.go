package action

import (
	"fmt"
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

const markdownExtension = ".md"

// IsRunnable reports whether a rule kind produces something `bazel run` can
// execute.
func IsRunnable(kind target.Kind) bool {
	return strings.HasSuffix(string(kind), "_binary")
}

// IsTestable reports whether a rule kind is a test `bazel test` can run.
func IsTestable(kind target.Kind) bool {
	return strings.HasSuffix(string(kind), "_test")
}

// IsGo reports whether a rule kind belongs to a Go package, which is what
// makes GoLand the right IDE for it.
func IsGo(kind target.Kind) bool {
	return strings.HasPrefix(string(kind), "go_")
}

// IsNative reports whether a rule kind belongs to a C/C++ package, which is
// what makes CLion the right IDE for it.
func IsNative(kind target.Kind) bool {
	return strings.HasPrefix(string(kind), "cc_")
}

// ProducesRunLog reports whether an action runs a command whose output is worth
// keeping and reading.
//
// Three behaviours follow from it, which is why it is one predicate rather than
// three: the output is captured to a log, basalt pauses afterwards so the
// result can be read before the next frame wipes it, and the outcome is
// recorded against the target so the list can show it as passed or failed.
//
// A bazel build or test prints a result and exits. An editor, a pager or an IDE
// is quit deliberately and leaves nothing to review, so none of the three would
// mean anything for them.
func ProducesRunLog(candidate Action) bool {
	return candidate != nil && candidate.Kind() == KindBazel
}

// IsMarkdown reports whether a file name is a Markdown document, which is what
// the Nano actions offer.
func IsMarkdown(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), markdownExtension)
}

// SelectMarkdown filters directory entry names down to the Markdown files.
func SelectMarkdown(names []string) []string {
	selected := make([]string, 0, len(names))
	for _, name := range names {
		if IsMarkdown(name) {
			selected = append(selected, name)
		}
	}
	return selected
}

// toolCommand builds a CommandSpec for a tool, refusing when the tool is not
// available. Every action's Command goes through here, so no action can ever
// produce an invocation of a program that is not on the machine.
func toolCommand(ctx Context, tool toolset.Tool, dir string, args ...string) (port.CommandSpec, error) {
	if !ctx.Tools.Has(tool) {
		return port.CommandSpec{}, fmt.Errorf("%w: %s", ErrToolUnavailable, tool)
	}
	if dir == "" {
		return port.CommandSpec{}, ErrNoDirectory
	}
	return port.CommandSpec{Path: ctx.Tools.PathOf(tool), Args: args, Dir: dir}, nil
}
