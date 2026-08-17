// Package indexer produces the target index: from the on-disk cache when it is
// fresh, from `bazel query` when it is not.
package indexer

import (
	"context"
	"io/fs"

	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

const (
	cacheDirPerm  fs.FileMode = 0o755
	cacheFilePerm fs.FileMode = 0o644
)

type service struct {
	source   port.TargetSource
	store    port.FileStore
	clock    port.Clock
	root     string
	cacheDir string
	useCache bool
}

// Config is the indexer's construction parameters, grouped because five
// positional arguments at the call site is worse than one named struct.
type Config struct {
	Source   port.TargetSource
	Store    port.FileStore
	Clock    port.Clock
	Root     string
	CacheDir string
	UseCache bool
}

// New builds the indexer, returning the interface (Rule 7.2).
func New(config Config) Indexer {
	return &service{
		source:   config.Source,
		store:    config.Store,
		clock:    config.Clock,
		root:     config.Root,
		cacheDir: config.CacheDir,
		useCache: config.UseCache,
	}
}

// Load returns the index, preferring a fresh cache. Any reason the cache could
// not be used is carried on the Result rather than raised, because every one of
// them recovers the same way: re-index.
func (s *service) Load(ctx context.Context) (Result, error) {
	cached, cacheErr := s.loadCache()
	if cacheErr == nil {
		return Result{Index: cached, Source: SourceCache}, nil
	}
	result, err := s.Refresh(ctx)
	if err != nil {
		return Result{}, err
	}
	result.CacheError = cacheErr
	return result, nil
}

// Refresh always re-queries bazel and rewrites the cache. A cache write failure
// is not fatal: basalt runs perfectly well with an index it could not persist.
func (s *service) Refresh(ctx context.Context) (Result, error) {
	targets, err := s.source.Targets(ctx)
	if err != nil {
		return Result{}, err
	}
	built := index.Index{
		SchemaVersion: index.SchemaVersion,
		GeneratedAt:   s.clock.Now(),
		Targets:       targets,
	}
	return Result{Index: built, Source: SourceQuery, CacheError: s.saveCache(built)}, nil
}

func (s *service) loadCache() (index.Index, error) {
	if !s.useCache {
		return index.Index{}, ErrCacheDisabled
	}
	data, err := s.store.Read(index.FilePath(s.cacheDir))
	if err != nil {
		return index.Index{}, err
	}
	decoded, err := index.Decode(data, s.root)
	if err != nil {
		return index.Index{}, err
	}
	if index.IsStale(decoded, s.clock.Now()) {
		return index.Index{}, ErrCacheStale
	}
	return decoded, nil
}

func (s *service) saveCache(built index.Index) error {
	data, err := index.Encode(built)
	if err != nil {
		return err
	}
	if err := s.store.MkdirAll(s.cacheDir, cacheDirPerm); err != nil {
		return err
	}
	return s.store.Write(index.FilePath(s.cacheDir), data, cacheFilePerm)
}
