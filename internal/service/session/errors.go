package session

import "errors"

// ErrNothingSelected means an action was triggered with no target highlighted,
// which the state machine should already prevent — reaching it is a bug in the
// gating rather than operator error.
var ErrNothingSelected = errors.New("no target is selected")
