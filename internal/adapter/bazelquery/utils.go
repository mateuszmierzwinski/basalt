package bazelquery

import (
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// ParseQueryOutput turns `--output=label_kind` output into targets.
//
// Unparseable lines are skipped rather than fatal: bazel occasionally emits a
// notice on stdout, and one stray line should not cost the operator the entire
// index. A result with no usable targets at all is still an error, because that
// means the format changed rather than that one line was odd.
func ParseQueryOutput(stdout string) ([]target.Target, error) {
	var targets []target.Target
	for _, line := range strings.Split(stdout, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if parsed, err := target.ParseLabelKind(line); err == nil {
			targets = append(targets, parsed)
		}
	}
	if len(targets) == 0 {
		return nil, ErrNoTargets
	}
	return targets, nil
}

// lastLine extracts the final non-empty line of a stream, which for a failed
// bazel invocation is the actual error rather than the progress spam above it.
func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if trimmed := strings.TrimSpace(lines[i]); trimmed != "" {
			return trimmed
		}
	}
	return "no output"
}
