package toolset

// NewSet wraps resolved availabilities in a Set. It copies the map so a caller
// holding the original cannot mutate a snapshot the UI is rendering from.
func NewSet(entries map[Tool]Availability) Set {
	copied := make(map[Tool]Availability, len(entries))
	for tool, availability := range entries {
		copied[tool] = availability
	}
	return Set{entries: copied}
}

// Has reports whether a tool is usable, and therefore whether the actions bound
// to it belong in the action panel.
func (s Set) Has(tool Tool) bool {
	return s.entries[tool].Found
}

// PathOf returns the resolved executable path, or the empty string when the
// tool is absent. Commands are built from this path rather than the bare name
// so a $PATH change mid-session cannot redirect an action.
func (s Set) PathOf(tool Tool) string {
	return s.entries[tool].Path
}

// Missing lists the absent tools in KnownTools order, for the startup notice
// that explains why some actions are not offered.
func (s Set) Missing() []Tool {
	var missing []Tool
	for _, tool := range KnownTools {
		if !s.entries[tool].Found {
			missing = append(missing, tool)
		}
	}
	return missing
}
