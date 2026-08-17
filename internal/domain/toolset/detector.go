package toolset

import "github.com/mateuszmierzwinski/basalt/internal/port"

type detector struct {
	env port.Environment
}

// NewDetector builds the tool detector, returning the interface (Rule 7.2).
func NewDetector(env port.Environment) Detector {
	return &detector{env: env}
}

// Detect probes $PATH for every known tool. It returns the populated Set even
// on error, so the caller can report exactly which tools were found alongside
// the failure.
func (d *detector) Detect() (Set, error) {
	entries := make(map[Tool]Availability, len(KnownTools))
	for _, tool := range KnownTools {
		entries[tool] = d.probe(tool)
	}
	set := NewSet(entries)
	if !set.Has(ToolBazel) {
		return set, ErrBazelMissing
	}
	return set, nil
}

func (d *detector) probe(tool Tool) Availability {
	path, err := d.env.LookPath(string(tool))
	if err != nil {
		return Availability{Tool: tool, Found: false}
	}
	return Availability{Tool: tool, Path: path, Found: true}
}
