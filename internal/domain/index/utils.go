package index

import (
	"path/filepath"
	"time"
)

const (
	// SchemaVersion is bumped whenever Index or Target changes shape. An older
	// or newer cache is discarded rather than misparsed.
	//
	// 2 dropped the recorded root, which made the cache checkout-portable. A
	// version 1 cache is discarded on first run and rebuilt in about a second.
	SchemaVersion = 2

	// TTL is the brief's one-day staleness window.
	TTL = 24 * time.Hour

	cacheFileName = "targets.json"
)

// FilePath returns the cache file inside the cache directory.
func FilePath(cacheDir string) string {
	return filepath.Join(cacheDir, cacheFileName)
}

// IsStale reports whether an index is older than the TTL, or was generated in
// the future — which a clock change or a copied cache can produce, and which
// would otherwise pin a stale index forever.
func IsStale(idx Index, now time.Time) bool {
	age := now.Sub(idx.GeneratedAt)
	return age < 0 || age >= TTL
}

// Age is how long ago the index was generated, for the status line.
func Age(idx Index, now time.Time) time.Duration {
	return now.Sub(idx.GeneratedAt)
}
