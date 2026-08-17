package toolset

import "errors"

// ErrBazelMissing stops startup. Every other tool gates a single action and its
// absence merely hides that action, but without bazel there is nothing to
// index and therefore nothing to show.
var ErrBazelMissing = errors.New("bazel was not found on $PATH")
