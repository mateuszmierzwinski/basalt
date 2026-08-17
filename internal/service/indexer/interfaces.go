package indexer

import "context"

// Indexer produces the target index, from cache when it is fresh and from
// bazel when it is not.
type Indexer interface {
	Load(ctx context.Context) (Result, error)
	Refresh(ctx context.Context) (Result, error)
}
