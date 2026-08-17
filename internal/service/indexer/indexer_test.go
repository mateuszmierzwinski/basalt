package indexer

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/port/mocks"
	"go.uber.org/mock/gomock"
)

var (
	fixedNow  = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	cachePath = "/repo/basalt_target_cache/targets.json"
)

func sampleTargets(t *testing.T) []target.Target {
	t.Helper()
	tgt, err := target.New(target.KindGoBinary, "//src/go/tools/basalt/cmd/basalt:basalt")
	if err != nil {
		t.Fatalf("target.New() error = %v", err)
	}
	return []target.Target{tgt}
}

func encodedCache(t *testing.T, generatedAt time.Time) []byte {
	t.Helper()
	data, err := index.Encode(index.Index{
		SchemaVersion: index.SchemaVersion,
		GeneratedAt:   generatedAt,
		Targets:       sampleTargets(t),
	})
	if err != nil {
		t.Fatalf("index.Encode() error = %v", err)
	}
	return data
}

type harness struct {
	source *mocks.MockTargetSource
	store  *mocks.MockFileStore
	clock  *mocks.MockClock
}

func newHarness(t *testing.T) harness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := harness{
		source: mocks.NewMockTargetSource(ctrl),
		store:  mocks.NewMockFileStore(ctrl),
		clock:  mocks.NewMockClock(ctrl),
	}
	h.clock.EXPECT().Now().Return(fixedNow).AnyTimes()
	return h
}

func (h harness) indexer(useCache bool) Indexer {
	return New(Config{
		Source:   h.source,
		Store:    h.store,
		Clock:    h.clock,
		Root:     "/repo",
		CacheDir: "/repo/basalt_target_cache",
		UseCache: useCache,
	})
}

func (h harness) expectCacheWrite() {
	h.store.EXPECT().MkdirAll("/repo/basalt_target_cache", fs.FileMode(0o755)).Return(nil)
	h.store.EXPECT().Write(cachePath, gomock.Any(), fs.FileMode(0o644)).Return(nil)
}

func TestLoadUsesAFreshCache(t *testing.T) {
	h := newHarness(t)
	h.store.EXPECT().Read(cachePath).Return(encodedCache(t, fixedNow.Add(-time.Hour)), nil)

	got, err := h.indexer(true).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Source != SourceCache {
		t.Fatalf("Source = %q, want %q", got.Source, SourceCache)
	}
	if len(got.Index.Targets) != 1 {
		t.Fatalf("Targets = %+v", got.Index.Targets)
	}
}

func TestLoadRefreshesWhenCacheIsStale(t *testing.T) {
	h := newHarness(t)
	h.store.EXPECT().Read(cachePath).Return(encodedCache(t, fixedNow.Add(-48*time.Hour)), nil)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(true).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Source != SourceQuery {
		t.Fatalf("Source = %q, want %q", got.Source, SourceQuery)
	}
	if !errors.Is(got.CacheError, ErrCacheStale) {
		t.Fatalf("CacheError = %v, want ErrCacheStale", got.CacheError)
	}
}

func TestLoadRefreshesWhenCacheIsMissing(t *testing.T) {
	h := newHarness(t)
	h.store.EXPECT().Read(cachePath).Return(nil, fs.ErrNotExist)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(true).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Source != SourceQuery || !errors.Is(got.CacheError, fs.ErrNotExist) {
		t.Fatalf("Result = %+v", got)
	}
}

func TestLoadRefreshesWhenCacheIsCorrupt(t *testing.T) {
	h := newHarness(t)
	h.store.EXPECT().Read(cachePath).Return([]byte("{not json"), nil)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(true).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !errors.Is(got.CacheError, index.ErrCorruptCache) {
		t.Fatalf("CacheError = %v, want ErrCorruptCache", got.CacheError)
	}
}

func TestLoadRefreshesWhenCacheIsPoisoned(t *testing.T) {
	h := newHarness(t)
	poisoned, err := index.Encode(index.Index{
		SchemaVersion: index.SchemaVersion,
		GeneratedAt:   fixedNow,
		Targets: []target.Target{{
			Kind: "go_binary", Package: "//../../etc", Name: "passwd",
			Label: "//../../etc:passwd", Display: "[etc:passwd]",
		}},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	h.store.EXPECT().Read(cachePath).Return(poisoned, nil)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(true).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !errors.Is(got.CacheError, index.ErrPoisonedEntry) {
		t.Fatalf("CacheError = %v, want ErrPoisonedEntry", got.CacheError)
	}
}

func TestLoadBypassesCacheWhenDisabled(t *testing.T) {
	h := newHarness(t)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(false).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !errors.Is(got.CacheError, ErrCacheDisabled) {
		t.Fatalf("CacheError = %v, want ErrCacheDisabled", got.CacheError)
	}
}

func TestLoadFailsWhenQueryFails(t *testing.T) {
	h := newHarness(t)
	queryErr := errors.New("bazel exploded")
	h.store.EXPECT().Read(cachePath).Return(nil, fs.ErrNotExist)
	h.source.EXPECT().Targets(gomock.Any()).Return(nil, queryErr)

	if _, err := h.indexer(true).Load(context.Background()); !errors.Is(err, queryErr) {
		t.Fatalf("Load() error = %v, want the query error", err)
	}
}

// An index basalt cannot persist is still a perfectly usable index.
func TestRefreshSurvivesACacheWriteFailure(t *testing.T) {
	h := newHarness(t)
	writeErr := errors.New("read-only filesystem")
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.store.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Return(nil)
	h.store.EXPECT().Write(cachePath, gomock.Any(), gomock.Any()).Return(writeErr)

	got, err := h.indexer(true).Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if !errors.Is(got.CacheError, writeErr) {
		t.Fatalf("CacheError = %v, want the write error", got.CacheError)
	}
	if len(got.Index.Targets) != 1 {
		t.Fatal("the index should still be returned")
	}
}

func TestRefreshStampsTheIndex(t *testing.T) {
	h := newHarness(t)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(true).Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if !got.Index.GeneratedAt.Equal(fixedNow) {
		t.Fatalf("GeneratedAt = %v, want %v", got.Index.GeneratedAt, fixedNow)
	}
	if got.Index.SchemaVersion != index.SchemaVersion {
		t.Fatalf("Index = %+v", got.Index)
	}
}

func TestRefreshAlwaysQueriesEvenWithAFreshCache(t *testing.T) {
	h := newHarness(t)
	h.source.EXPECT().Targets(gomock.Any()).Return(sampleTargets(t), nil)
	h.expectCacheWrite()

	got, err := h.indexer(true).Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if got.Source != SourceQuery {
		t.Fatalf("Source = %q, want %q", got.Source, SourceQuery)
	}
}
