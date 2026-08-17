package action

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// bazelAction is the shared shape of every bazel verb: same tool, same working
// directory, same argument form — only the subcommand and the applicability
// rule differ (Rule 5.1).
type bazelAction struct {
	id         string
	label      string
	subcommand string
	applies    func(ctx Context) bool
	// target names what the subcommand is pointed at. It defaults to the
	// selected label; the package-wide test action overrides it with a
	// wildcard.
	target func(ctx Context) string
}

func (a *bazelAction) ID() string    { return a.id }
func (a *bazelAction) Label() string { return a.label }
func (a *bazelAction) Kind() Kind    { return KindBazel }

// Applies gates on the rule kind and on bazel being present.
func (a *bazelAction) Applies(ctx Context) bool {
	return ctx.Tools.Has(toolset.ToolBazel) && a.applies(ctx)
}

// Command runs bazel from the monorepo root rather than the package directory,
// because a label is resolved relative to the workspace.
func (a *bazelAction) Command(ctx Context) (port.CommandSpec, error) {
	return toolCommand(ctx, toolset.ToolBazel, ctx.Root, a.subcommand, a.targetOf(ctx))
}

// targetOf is the label the subcommand is pointed at, defaulting to whatever is
// selected.
func (a *bazelAction) targetOf(ctx Context) string {
	if a.target == nil {
		return ctx.Target.Label
	}
	return a.target(ctx)
}

// newBazelRun builds the "Bazel > Run" action, offered only for rules that
// produce an executable.
func newBazelRun() Action {
	return &bazelAction{
		id:         IDBazelRun,
		label:      "Bazel > Run",
		subcommand: "run",
		applies:    func(ctx Context) bool { return IsRunnable(ctx.Target.Kind) },
	}
}

// newBazelTest builds the "Bazel > Test" action, offered only for test rules.
func newBazelTest() Action {
	return &bazelAction{
		id:         IDBazelTest,
		label:      "Bazel > Test",
		subcommand: "test",
		applies:    func(ctx Context) bool { return IsTestable(ctx.Target.Kind) },
	}
}

// newBazelTestPackage builds the "Bazel > Test package" action: the tests that
// cover the selected target, which for a library or a binary live beside it
// rather than in the same rule.
//
// It runs a `/...` wildcard, so selecting any target in a project offers to run
// that project's suite. It is withheld for a test rule, which already has the
// precise action above and would otherwise be offered both.
func newBazelTestPackage() Action {
	return &bazelAction{
		id:         IDBazelTestPackage,
		label:      "Bazel > Test package",
		subcommand: "test",
		applies: func(ctx Context) bool {
			return ctx.PackageHasTests && !IsTestable(ctx.Target.Kind)
		},
		target: func(ctx Context) string { return ctx.Target.Package + "/..." },
	}
}

// newBazelBuild builds the "Bazel > Build" action, which applies to every rule.
func newBazelBuild() Action {
	return &bazelAction{
		id:         IDBazelBuild,
		label:      "Bazel > Build",
		subcommand: "build",
		applies:    func(Context) bool { return true },
	}
}
