// Package osenv adapts the real process environment to the Environment port.
package osenv

import (
	"os"
	"os/exec"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

type environment struct{}

// New builds the real environment adapter.
func New() port.Environment {
	return &environment{}
}

// Lookup reads an environment variable, reporting whether it was set at all —
// an exported-but-empty $MONOREPO is a different failure from an unset one.
func (e *environment) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

// LookPath resolves an executable against $PATH.
func (e *environment) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}
