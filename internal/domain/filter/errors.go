package filter

import "errors"

// Query failures. Both are recoverable and expected: the operator is typing.
var (
	ErrInvalidRegex = errors.New("invalid regular expression")
	ErrQueryTooLong = errors.New("query is too long")
)
