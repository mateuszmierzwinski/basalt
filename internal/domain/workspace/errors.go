package workspace

import "errors"

// Startup failures that stop basalt before it draws anything. Each is a
// sentinel so callers can distinguish "you forgot to export $MONOREPO" from
// "the path you exported is not a monorepo" without matching on strings.
var (
	ErrMonorepoUnset        = errors.New("$MONOREPO is not set")
	ErrMonorepoNotAbsolute  = errors.New("$MONOREPO must be an absolute path")
	ErrMonorepoNotDirectory = errors.New("$MONOREPO does not point at a directory")
	ErrSourceRootMissing    = errors.New("$MONOREPO has no src directory")
)
