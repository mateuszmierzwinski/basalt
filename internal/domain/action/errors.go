package action

import "errors"

// Command-construction failures. Each means the action should never have been
// offered, so reaching one is a bug in the gating rather than operator error.
var (
	ErrToolUnavailable = errors.New("the tool this action needs is not available")
	ErrNoDirectory     = errors.New("the target has no resolved directory")
	ErrNoDestination   = errors.New("there is no usable ~/.local/bin to copy into")
)
