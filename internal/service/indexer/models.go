package indexer

import "github.com/mateuszmierzwinski/basalt/internal/domain/index"

// Source records where an index came from, so the UI can say "cached 3h ago"
// rather than leaving the operator guessing whether a re-index happened.
type Source string

const (
	SourceCache Source = "cache"
	SourceQuery Source = "bazel query"
)

// Result is an index plus its provenance. CacheError is the non-fatal reason a
// cache was rejected — stale, corrupt, foreign or poisoned — and is reported
// rather than swallowed, because a cache that is silently discarded every run
// is a bug the operator would never notice.
type Result struct {
	Index      index.Index
	Source     Source
	CacheError error
}
