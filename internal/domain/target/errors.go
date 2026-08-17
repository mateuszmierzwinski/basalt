package target

import "errors"

// Parsing failures. `bazel query` output is machine-generated and should always
// be well-formed, but the same parser is fed the JSON cache, which is a
// user-writable file in the repository root — so malformed input is an
// expected condition, not an impossible one.
var (
	ErrMalformedQueryLine = errors.New("not a bazel label_kind line")
	ErrMalformedLabel     = errors.New("not a bazel label")
	ErrLabelEscapesRoot   = errors.New("label resolves outside the monorepo")
)
