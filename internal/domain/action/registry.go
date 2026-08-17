package action

import "sync"

// Action identifiers. They are the registry's keys, the ordering list's
// entries and each action's own ID, so they are named once here rather than
// repeated as literals.
const (
	IDBazelRun         = "bazel.run"
	IDBazelTest        = "bazel.test"
	IDBazelTestPackage = "bazel.test.package"
	IDBazelBuild       = "bazel.build"
	IDLessLog          = "less.log"
	IDClaudeDSP        = "claude.dsp"
	IDClaudeRun        = "claude.run"
	IDGoland           = "goland.open"
	IDClion            = "clion.open"
)

// staticOrder fixes the order actions appear in the panel, so the list does not
// reshuffle as different targets are selected. Nano actions are spliced in at
// the editor position because their number varies per target.
var staticOrder = []string{
	IDBazelRun,
	IDBazelTest,
	IDBazelTestPackage,
	IDBazelBuild,
	IDLessLog,
	IDClaudeDSP,
	IDClaudeRun,
	IDGoland,
	IDClion,
}

var (
	staticOnce    sync.Once
	staticActions map[string]Action
)

// loadStatic populates the package-scoped action map exactly once (Rule 6.1,
// 6.2). Every value is a named struct implementing Action (Rule 6.3).
func loadStatic() {
	staticActions = map[string]Action{
		IDBazelRun:         newBazelRun(),
		IDBazelTest:        newBazelTest(),
		IDBazelTestPackage: newBazelTestPackage(),
		IDBazelBuild:       newBazelBuild(),
		IDLessLog:          newLessAction(),
		IDClaudeDSP:        newClaudeDSP(),
		IDClaudeRun:        newClaude(),
		IDGoland:           newGoland(),
		IDClion:            newClion(),
	}
}

type registry struct{}

// NewRegistry builds the action registry, returning the interface (Rule 7.2).
func NewRegistry() Registry {
	staticOnce.Do(loadStatic)
	return &registry{}
}

// For returns the actions available for a target, in panel order. An action
// that does not apply — wrong rule kind, missing tool — is absent rather than
// shown and then failing on selection.
func (r *registry) For(ctx Context) []Action {
	available := make([]Action, 0, len(staticOrder)+len(ctx.Markdown))
	for _, id := range staticOrder {
		available = appendIfApplies(available, staticActions[id], ctx)
		if id == IDBazelBuild {
			available = append(available, markdownActions(ctx)...)
		}
	}
	return available
}

// markdownActions builds one Nano action per Markdown file in the target's
// directory, which is what makes the panel adapt to the files actually present.
func markdownActions(ctx Context) []Action {
	built := make([]Action, 0, len(ctx.Markdown))
	for _, file := range ctx.Markdown {
		built = appendIfApplies(built, newNanoAction(file), ctx)
	}
	return built
}

func appendIfApplies(actions []Action, candidate Action, ctx Context) []Action {
	if candidate != nil && candidate.Applies(ctx) {
		return append(actions, candidate)
	}
	return actions
}
