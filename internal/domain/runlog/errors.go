package runlog

import "errors"

// History-load failures. None is fatal: a rejected history simply means no
// target shows a previous outcome until it is run again, which is the same
// state as a fresh checkout.
var (
	ErrCorruptHistory = errors.New("run history is not valid JSON")
	ErrSchemaMismatch = errors.New("run history was written by a different basalt version")
	ErrPoisonedEntry  = errors.New("run history contains an invalid entry")
	ErrLogEscapesDir  = errors.New("log file is outside the cache directory")
)
