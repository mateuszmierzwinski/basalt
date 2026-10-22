package action

import (
	"path/filepath"
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// installAction copies a binary target's build artifact into the operator's
// ~/.local/bin, for trying the program out by name from any shell.
//
// It is one bazel invocation: `bazel run --run_under=<install …>` builds the
// target and hands the artifact's path to install(1) instead of executing it.
// That keeps it a bazel action in every respect the session cares about — its
// output is logged, basalt pauses on the result, and the target is marked
// passed or failed — and it never puts a label through a shell.
type installAction struct{}

func newInstallLocal() Action { return &installAction{} }

func (a *installAction) ID() string    { return IDInstallLocal }
func (a *installAction) Label() string { return "Bazel > Copy to ~/.local/bin" }
func (a *installAction) Kind() Kind    { return KindBazel }

// Applies offers the copy for executables only, and only when both bazel and
// install are present and the destination can be quoted safely.
func (a *installAction) Applies(ctx Context) bool {
	return ctx.Tools.Has(toolset.ToolBazel) && ctx.Tools.Has(toolset.ToolInstall) &&
		IsRunnable(ctx.Target.Kind) && IsQuotableDirectory(ctx.LocalBin)
}

// Command runs from the monorepo root, where the label resolves.
func (a *installAction) Command(ctx Context) (port.CommandSpec, error) {
	if !IsQuotableDirectory(ctx.LocalBin) {
		return port.CommandSpec{}, ErrNoDestination
	}
	if !ctx.Tools.Has(toolset.ToolInstall) {
		return port.CommandSpec{}, ErrToolUnavailable
	}
	return toolCommand(ctx, toolset.ToolBazel, ctx.Root, "run", RunUnderInstall(ctx.Tools.PathOf(toolset.ToolInstall), ctx.LocalBin), ctx.Target.Label)
}

// RunUnderInstall is the --run_under flag that makes `bazel run` copy the
// artifact into dir rather than execute it. -D creates dir when it is missing.
// Bazel splits the value as a shell would, so both paths are single-quoted;
// IsQuotableDirectory has already refused a quote inside dir.
func RunUnderInstall(installPath, dir string) string {
	return "--run_under='" + installPath + "' -D -m 0755 -t '" + dir + "'"
}

// IsQuotableDirectory reports whether dir is an absolute path that can sit
// inside single quotes unchanged — the one property RunUnderInstall needs.
func IsQuotableDirectory(dir string) bool {
	return filepath.IsAbs(dir) && !strings.ContainsAny(dir, "'\x00\n")
}
