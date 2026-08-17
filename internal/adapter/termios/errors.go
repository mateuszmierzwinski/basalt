package termios

import "errors"

// ErrNotATerminal means basalt was not started from an interactive terminal.
// It is a plain failure rather than a fallback: the whole tool is a full-screen
// UI, and there is nothing sensible to do without a tty.
var ErrNotATerminal = errors.New("basalt must be run from an interactive terminal")
