package toolset

// Detector probes the host for the external programs basalt drives.
type Detector interface {
	Detect() (Set, error)
}
