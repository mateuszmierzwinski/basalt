// Package osfs adapts the real filesystem to basalt's filesystem ports.
package osfs

import (
	"os"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// Probe is the filesystem-inspection surface this package provides. One adapter
// satisfies both ports, so callers take whichever they need.
type Probe interface {
	port.DirectoryProbe
	port.FileProbe
}

type probe struct{}

// NewProbe builds the real filesystem probe.
func NewProbe() Probe {
	return &probe{}
}

// IsDirectory reports whether the path exists and is a directory. A stat error
// of any kind — missing, permission denied, broken symlink — is reported as
// "not a directory"; basalt cannot use the path either way.
func (p *probe) IsDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsFile reports whether the path exists and is a regular file.
//
// Regular specifically: a log path that resolved to a FIFO or a device would
// hang or misbehave the pager opening it, and neither is something basalt wrote.
func (p *probe) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
