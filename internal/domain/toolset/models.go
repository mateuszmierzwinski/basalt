package toolset

// Tool names an external program basalt can drive on the operator's behalf.
type Tool string

const (
	ToolBazel   Tool = "bazel"
	ToolNano    Tool = "nano"
	ToolLess    Tool = "less"
	ToolClaude  Tool = "claude"
	ToolGoland  Tool = "goland"
	ToolClion   Tool = "clion"
	ToolInstall Tool = "install"
)

// KnownTools is the probe order, which is also the order missing tools are
// reported in.
var KnownTools = []Tool{ToolBazel, ToolNano, ToolLess, ToolClaude, ToolGoland, ToolClion, ToolInstall}

// Availability records where a tool was found, or that it is absent. Actions
// bound to an absent tool are hidden from the action panel rather than offered
// and then failing on selection.
type Availability struct {
	Tool  Tool
	Path  string
	Found bool
}

// Set is the resolved availability of every tool basalt knows about. It is a
// value type so the action panel can hold a snapshot without synchronizing
// against a later re-probe.
type Set struct {
	entries map[Tool]Availability
}
