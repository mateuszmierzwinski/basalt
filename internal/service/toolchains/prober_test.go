package toolchains

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

const testRoot = "/repo"

// fakeEnvironment resolves only the programs it was told exist, so a test can
// describe a host without gcc without touching the real $PATH.
type fakeEnvironment struct {
	present map[string]string
}

func (e fakeEnvironment) Lookup(string) (string, bool) { return "", false }

func (e fakeEnvironment) LookPath(file string) (string, error) {
	if path, ok := e.present[file]; ok {
		return path, nil
	}
	return "", errors.New("not found")
}

// fakeRunner answers each probe by the executable path it was invoked with and
// counts the calls, which is how the "probe once" contract is asserted.
type fakeRunner struct {
	answers map[string]string
	fails   map[string]bool
	calls   int
}

func (r *fakeRunner) Run(_ context.Context, spec port.CommandSpec) (port.CommandResult, error) {
	r.calls++
	if r.fails[spec.Path] {
		return port.CommandResult{}, errors.New("exec failed")
	}
	return port.CommandResult{Stdout: r.answers[spec.Path]}, nil
}

// fakeStore serves a fixed set of files and records what was read, so the
// go.mod walk can be observed without a real tree.
type fakeStore struct {
	files map[string]string
	reads []string
}

func (s *fakeStore) Read(path string) ([]byte, error) {
	s.reads = append(s.reads, path)
	contents, ok := s.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(contents), nil
}

func (s *fakeStore) Write(string, []byte, fs.FileMode) error { return nil }
func (s *fakeStore) MkdirAll(string, fs.FileMode) error      { return nil }
func (s *fakeStore) ListNames(string) ([]string, error)      { return nil, nil }

func fullHost() fakeEnvironment {
	return fakeEnvironment{present: map[string]string{
		"go":  "/usr/bin/go",
		"gcc": "/usr/bin/gcc",
		"g++": "/usr/bin/g++",
	}}
}

func fullRunner() *fakeRunner {
	return &fakeRunner{answers: map[string]string{
		"/usr/bin/go":  "go version go1.26.5 linux/amd64\n",
		"/usr/bin/gcc": "gcc (GCC) 16.1.1 20260101\nCopyright (C) 2026\n",
		"/usr/bin/g++": "g++ (GCC) 16.1.1 20260101\nCopyright (C) 2026\n",
	}}
}

func newProber(t *testing.T, environment port.Environment, runner port.CommandRunner, store port.FileStore) Prober {
	t.Helper()
	return New(context.Background(), Config{
		Environment: environment,
		Runner:      runner,
		Store:       store,
		Root:        testRoot,
	})
}

func goTarget(pkg string) target.Target {
	return target.Target{Kind: target.KindGoBinary, Package: pkg, Name: "app", Label: pkg + ":app"}
}

func TestSummaryReportsAvailableVersions(t *testing.T) {
	prober := newProber(t, fullHost(), fullRunner(), &fakeStore{})

	summary := prober.SummaryFor(goTarget("//src/go/demo"), []target.Language{target.LanguageC})

	if summary.GoAvailable != "1.26.5" {
		t.Fatalf("go available = %q, want 1.26.5", summary.GoAvailable)
	}
	if summary.GCC != "16.1.1" || summary.GXX != "16.1.1" {
		t.Fatalf("gcc/g++ = %q/%q, want 16.1.1", summary.GCC, summary.GXX)
	}
}

func TestSummaryLeavesMissingCompilersEmpty(t *testing.T) {
	environment := fakeEnvironment{present: map[string]string{"go": "/usr/bin/go"}}
	prober := newProber(t, environment, fullRunner(), &fakeStore{})

	summary := prober.SummaryFor(goTarget("//src/go/demo"), []target.Language{target.LanguageC})

	if summary.GCC != "" || summary.GXX != "" {
		t.Fatalf("absent compilers reported as %q/%q, want empty", summary.GCC, summary.GXX)
	}
}

func TestSummaryTreatsFailedProbeAsAbsent(t *testing.T) {
	runner := fullRunner()
	runner.fails = map[string]bool{"/usr/bin/go": true}
	prober := newProber(t, fullHost(), runner, &fakeStore{})

	summary := prober.SummaryFor(goTarget("//src/go/demo"), []target.Language{target.LanguageGo})

	if summary.GoAvailable != "" {
		t.Fatalf("failed probe reported %q, want empty", summary.GoAvailable)
	}
}

func TestSummaryReadsNearestGoModRequirement(t *testing.T) {
	store := &fakeStore{files: map[string]string{
		"/repo/go.mod":                   "module repo\n\ngo 1.20\n",
		"/repo/src/go/demo/nested/BUILD": "",
		"/repo/src/go/demo/go.mod":       "module demo\n\ngo 1.25.8\n",
	}}
	prober := newProber(t, fullHost(), fullRunner(), store)

	summary := prober.SummaryFor(goTarget("//src/go/demo/nested"), []target.Language{target.LanguageGo})

	if summary.GoRequired != "1.25.8" {
		t.Fatalf("required = %q, want the nearest go.mod's 1.25.8", summary.GoRequired)
	}
}

func TestSummarySkipsGoRequirementForNonGoPackages(t *testing.T) {
	store := &fakeStore{files: map[string]string{"/repo/go.mod": "module repo\n\ngo 1.25.8\n"}}
	prober := newProber(t, fullHost(), fullRunner(), store)

	summary := prober.SummaryFor(goTarget("//src/c/audio"), []target.Language{target.LanguageC})

	if summary.GoRequired != "" {
		t.Fatalf("required = %q, want empty for a C-only package", summary.GoRequired)
	}
	if len(store.reads) != 0 {
		t.Fatalf("read %v, want no go.mod lookup at all", store.reads)
	}
}

func TestSummaryLeavesRequirementEmptyWithoutGoMod(t *testing.T) {
	prober := newProber(t, fullHost(), fullRunner(), &fakeStore{})

	summary := prober.SummaryFor(goTarget("//src/go/demo"), []target.Language{target.LanguageGo})

	if summary.GoRequired != "" {
		t.Fatalf("required = %q, want empty when no go.mod governs the target", summary.GoRequired)
	}
}

func TestSummaryRejectsLabelEscapingRoot(t *testing.T) {
	store := &fakeStore{files: map[string]string{"/repo/go.mod": "module repo\n\ngo 1.25.8\n"}}
	prober := newProber(t, fullHost(), fullRunner(), store)

	summary := prober.SummaryFor(goTarget("//../../etc"), []target.Language{target.LanguageGo})

	if summary.GoRequired != "" {
		t.Fatalf("required = %q, want empty for a label outside the root", summary.GoRequired)
	}
	for _, read := range store.reads {
		if !strings.HasPrefix(read, testRoot+"/") {
			t.Fatalf("read %q outside the monorepo root", read)
		}
	}
}

func TestRequirementIsMemoisedPerDirectory(t *testing.T) {
	store := &fakeStore{files: map[string]string{"/repo/src/go/demo/go.mod": "module demo\n\ngo 1.25.8\n"}}
	prober := newProber(t, fullHost(), fullRunner(), store)
	selected := goTarget("//src/go/demo")

	prober.SummaryFor(selected, []target.Language{target.LanguageGo})
	after := len(store.reads)
	prober.SummaryFor(selected, []target.Language{target.LanguageGo})

	if len(store.reads) != after {
		t.Fatalf("second lookup read %v, want the memoised answer", store.reads[after:])
	}
}

func TestCompilersAreProbedOnlyAtConstruction(t *testing.T) {
	runner := fullRunner()
	prober := newProber(t, fullHost(), runner, &fakeStore{})
	probes := runner.calls

	prober.SummaryFor(goTarget("//src/go/demo"), []target.Language{target.LanguageGo})

	if runner.calls != probes {
		t.Fatalf("SummaryFor ran %d extra commands, want none", runner.calls-probes)
	}
	if probes != 3 {
		t.Fatalf("probed %d compilers at construction, want 3", probes)
	}
}

func BenchmarkSummaryFor(b *testing.B) {
	store := &fakeStore{files: map[string]string{"/repo/src/go/demo/go.mod": "module demo\n\ngo 1.25.8\n"}}
	prober := New(context.Background(), Config{
		Environment: fullHost(),
		Runner:      fullRunner(),
		Store:       store,
		Root:        testRoot,
	})
	selected := goTarget("//src/go/demo")
	languages := []target.Language{target.LanguageGo}

	for b.Loop() {
		prober.SummaryFor(selected, languages)
	}
}
