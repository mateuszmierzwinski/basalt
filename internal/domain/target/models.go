package target

// Kind is a Bazel rule kind, e.g. "go_binary". It drives which actions a
// target offers, so it is carried through the cache rather than re-derived.
type Kind string

// Rule kinds basalt reasons about by name. Anything else is still indexed and
// still buildable — only the kind-specific actions are withheld.
const (
	KindGoBinary  Kind = "go_binary"
	KindGoLibrary Kind = "go_library"
	KindGoTest    Kind = "go_test"
	KindCcBinary  Kind = "cc_binary"
	KindCcLibrary Kind = "cc_library"
	KindCcTest    Kind = "cc_test"
)

// Target is one Bazel rule as reported by `bazel query --output=label_kind`.
//
// Display is stored rather than computed on render because the target list is
// redrawn on every keystroke and the value never changes.
type Target struct {
	Kind    Kind   `json:"kind"`
	Package string `json:"package"`
	Name    string `json:"name"`
	Label   string `json:"label"`
	Display string `json:"display"`
}
