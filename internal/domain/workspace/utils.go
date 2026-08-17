package workspace

import (
	"path/filepath"
	"strings"
)

const (
	monorepoEnvVar = "MONOREPO"
	sourceDirName  = "src"
	cacheDirName   = "basalt_target_cache"
)

// NormalizeRoot cleans a monorepo root path. The operator's exported $MONOREPO
// carries a trailing separator, which would otherwise produce doubled
// separators in every path derived from it.
func NormalizeRoot(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	return filepath.Clean(trimmed)
}

// SourceDirectory returns the source tree basalt indexes and watches.
func SourceDirectory(root string) string {
	return filepath.Join(root, sourceDirName)
}

// CacheDirectory returns the on-disk location of the JSON target cache.
func CacheDirectory(root string) string {
	return filepath.Join(root, cacheDirName)
}
