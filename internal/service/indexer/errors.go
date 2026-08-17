package indexer

import "errors"

// ErrCacheStale is the reason a well-formed cache was rejected. It is reported
// through Result.CacheError, never returned from Load — a stale cache is an
// expected condition whose recovery is a re-index.
var ErrCacheStale = errors.New("target cache is older than the staleness window")

// ErrCacheDisabled records that -nocache bypassed an otherwise usable cache.
var ErrCacheDisabled = errors.New("target cache bypassed by -nocache")
