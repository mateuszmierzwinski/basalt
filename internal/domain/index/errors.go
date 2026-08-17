package index

import "errors"

// Cache-load failures. None of them is fatal to basalt: every one means
// "re-index from bazel", which is the same recovery path as a cache miss. They
// are distinct so the reason can be reported rather than silently swallowed.
var (
	ErrCorruptCache   = errors.New("target cache is not valid JSON")
	ErrSchemaMismatch = errors.New("target cache was written by a different basalt version")
	ErrPoisonedEntry  = errors.New("target cache contains an invalid target")
)
