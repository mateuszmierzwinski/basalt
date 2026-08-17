package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/index"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/domain/workspace"
	"github.com/mateuszmierzwinski/basalt/internal/service/indexer"
)

type stubResolver struct {
	result workspace.Workspace
	err    error
}

func (s stubResolver) Resolve() (workspace.Workspace, error) { return s.result, s.err }

type stubDetector struct {
	result toolset.Set
	err    error
}

func (s stubDetector) Detect() (toolset.Set, error) { return s.result, s.err }

type stubIndexer struct {
	result indexer.Result
	err    error
}

func (s stubIndexer) Load(context.Context) (indexer.Result, error)    { return s.result, s.err }
func (s stubIndexer) Refresh(context.Context) (indexer.Result, error) { return s.result, s.err }

type stubIndexerFactory struct {
	indexer   indexer.Indexer
	sawOpts   Options
	callCount int
}

func (s *stubIndexerFactory) For(_ Preflight, options Options) indexer.Indexer {
	s.sawOpts = options
	s.callCount++
	return s.indexer
}

type stubSession struct {
	started int
	err     error
}

func (s *stubSession) Start(context.Context, Preflight, indexer.Indexer, indexer.Result) error {
	s.started++
	return s.err
}

func availableTools(tools ...toolset.Tool) toolset.Set {
	entries := make(map[toolset.Tool]toolset.Availability, len(toolset.KnownTools))
	for _, tool := range toolset.KnownTools {
		entries[tool] = toolset.Availability{Tool: tool, Found: false}
	}
	for _, tool := range tools {
		entries[tool] = toolset.Availability{Tool: tool, Path: "/usr/bin/" + string(tool), Found: true}
	}
	return toolset.NewSet(entries)
}

func sampleResult(count int) indexer.Result {
	targets := make([]target.Target, 0, count)
	for i := 0; i < count; i++ {
		targets = append(targets, target.Target{Label: "//src/go:thing"})
	}
	return indexer.Result{
		Index:  index.Index{Targets: targets},
		Source: indexer.SourceCache,
	}
}

type harness struct {
	stdout  bytes.Buffer
	stderr  bytes.Buffer
	factory *stubIndexerFactory
	session *stubSession
	app     App
}

func newHarness(resolver stubResolver, detector stubDetector, index stubIndexer) *harness {
	h := &harness{
		factory: &stubIndexerFactory{indexer: index},
		session: &stubSession{},
	}
	h.app = NewApp(Dependencies{
		Resolver: resolver,
		Detector: detector,
		Indexers: h.factory,
		Session:  h.session,
		Stdout:   &h.stdout,
		Stderr:   &h.stderr,
	})
	return h
}

func workingHarness() *harness {
	return newHarness(
		stubResolver{result: workspace.Workspace{
			Root:       "/repo",
			SourceRoot: "/repo/src",
			CacheDir:   "/repo/basalt_target_cache",
		}},
		stubDetector{result: availableTools(toolset.KnownTools...)},
		stubIndexer{result: sampleResult(1304)},
	)
}

func TestRunStartsTheSession(t *testing.T) {
	h := workingHarness()
	if code := h.app.Run(context.Background(), Options{}); code != exitOK {
		t.Fatalf("Run() = %d, want %d (stderr: %s)", code, exitOK, h.stderr.String())
	}
	if h.session.started != 1 {
		t.Fatalf("session started %d times, want 1", h.session.started)
	}
}

func TestIndexOnlyReportsWithoutStartingTheSession(t *testing.T) {
	h := workingHarness()
	if code := h.app.Run(context.Background(), Options{IndexOnly: true}); code != exitOK {
		t.Fatalf("Run() = %d", code)
	}
	if h.session.started != 0 {
		t.Fatal("-index must not start the interactive session")
	}
	out := h.stdout.String()
	for _, want := range []string{"/repo", "/repo/basalt_target_cache", "1304"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout %q does not contain %q", out, want)
		}
	}
}

func TestIndexOnlyNamesMissingTools(t *testing.T) {
	h := newHarness(
		stubResolver{result: workspace.Workspace{Root: "/repo"}},
		stubDetector{result: availableTools(toolset.ToolBazel, toolset.ToolNano, toolset.ToolClaude)},
		stubIndexer{result: sampleResult(3)},
	)
	h.app.Run(context.Background(), Options{IndexOnly: true})

	out := h.stdout.String()
	if !strings.Contains(out, "goland") || !strings.Contains(out, "clion") {
		t.Fatalf("stdout %q should name the missing IDEs", out)
	}
	if strings.Contains(out, "bazel not found") {
		t.Fatalf("stdout %q should not report an available tool as missing", out)
	}
}

func TestNoCacheIsPassedToTheIndexer(t *testing.T) {
	h := workingHarness()
	h.app.Run(context.Background(), Options{NoCache: true, IndexOnly: true})
	if !h.factory.sawOpts.NoCache {
		t.Fatal("-nocache did not reach the indexer factory")
	}
}

func TestRunFailsOnWorkspaceError(t *testing.T) {
	h := newHarness(
		stubResolver{err: workspace.ErrMonorepoUnset},
		stubDetector{},
		stubIndexer{},
	)
	if code := h.app.Run(context.Background(), Options{}); code != exitFailure {
		t.Fatalf("Run() = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(h.stderr.String(), "$MONOREPO is not set") {
		t.Fatalf("stderr %q should explain the failure", h.stderr.String())
	}
	if h.stdout.Len() != 0 {
		t.Fatalf("stdout %q should stay empty on failure", h.stdout.String())
	}
	if h.factory.callCount != 0 {
		t.Fatal("indexing should not be attempted without a workspace")
	}
}

func TestRunFailsOnDetectorError(t *testing.T) {
	h := newHarness(
		stubResolver{result: workspace.Workspace{Root: "/repo"}},
		stubDetector{err: toolset.ErrBazelMissing},
		stubIndexer{},
	)
	if code := h.app.Run(context.Background(), Options{}); code != exitFailure {
		t.Fatalf("Run() = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(h.stderr.String(), "bazel was not found") {
		t.Fatalf("stderr %q should explain the missing bazel", h.stderr.String())
	}
}

func TestRunFailsWhenIndexingFails(t *testing.T) {
	h := newHarness(
		stubResolver{result: workspace.Workspace{Root: "/repo"}},
		stubDetector{result: availableTools(toolset.ToolBazel)},
		stubIndexer{err: errors.New("bazel query exploded")},
	)
	if code := h.app.Run(context.Background(), Options{}); code != exitFailure {
		t.Fatalf("Run() = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(h.stderr.String(), "bazel query exploded") {
		t.Fatalf("stderr = %q", h.stderr.String())
	}
	if h.session.started != 0 {
		t.Fatal("the session must not start without an index")
	}
}

// A rejected cache is reported but never fatal: basalt re-indexed and carried
// on, and a cache that is silently discarded every run is a bug nobody notices.
func TestCacheProblemIsReportedButNotFatal(t *testing.T) {
	result := sampleResult(5)
	result.CacheError = index.ErrCorruptCache
	h := newHarness(
		stubResolver{result: workspace.Workspace{Root: "/repo"}},
		stubDetector{result: availableTools(toolset.KnownTools...)},
		stubIndexer{result: result},
	)
	if code := h.app.Run(context.Background(), Options{}); code != exitOK {
		t.Fatalf("Run() = %d, want %d", code, exitOK)
	}
	if !strings.Contains(h.stderr.String(), "not valid JSON") {
		t.Fatalf("stderr %q should mention the rejected cache", h.stderr.String())
	}
	if h.session.started != 1 {
		t.Fatal("the session should still have started")
	}
}

func TestRunFailsWhenTheSessionFails(t *testing.T) {
	h := workingHarness()
	h.session.err = errors.New("no terminal")
	if code := h.app.Run(context.Background(), Options{}); code != exitFailure {
		t.Fatalf("Run() = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(h.stderr.String(), "no terminal") {
		t.Fatalf("stderr = %q", h.stderr.String())
	}
}
