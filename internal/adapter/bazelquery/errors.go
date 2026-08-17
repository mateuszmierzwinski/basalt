package bazelquery

import "errors"

// Query failures. A failed query is fatal to a re-index but not to a running
// basalt: a previously loaded index stays usable.
var (
	ErrQueryFailed  = errors.New("bazel query failed")
	ErrNoTargets    = errors.New("bazel query returned no usable targets")
	ErrQueryTimeout = errors.New("bazel query timed out")
)
