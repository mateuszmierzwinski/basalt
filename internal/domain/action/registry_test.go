package action

import (
	"errors"
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

func tools(available ...toolset.Tool) toolset.Set {
	entries := make(map[toolset.Tool]toolset.Availability, len(toolset.KnownTools))
	for _, tool := range toolset.KnownTools {
		entries[tool] = toolset.Availability{Tool: tool, Found: false}
	}
	for _, tool := range available {
		entries[tool] = toolset.Availability{Tool: tool, Path: "/usr/bin/" + string(tool), Found: true}
	}
	return toolset.NewSet(entries)
}

func contextFor(kind target.Kind, label string, set toolset.Set, markdown ...string) Context {
	tgt, _ := target.New(kind, label)
	return Context{
		Target:    tgt,
		Root:      "/repo",
		Directory: "/repo/src/go/tools/basalt",
		Tools:     set,
		Markdown:  markdown,
	}
}

func ids(actions []Action) []string {
	got := make([]string, 0, len(actions))
	for _, a := range actions {
		got = append(got, a.ID())
	}
	return got
}

func has(actions []Action, id string) bool {
	for _, a := range actions {
		if a.ID() == id {
			return true
		}
	}
	return false
}

func TestRegistryOffersRunOnlyForBinaries(t *testing.T) {
	registry := NewRegistry()
	all := tools(toolset.KnownTools...)

	if !has(registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", all)), "bazel.run") {
		t.Fatal("a go_binary should offer Bazel > Run")
	}
	if has(registry.For(contextFor(target.KindGoLibrary, "//src/go/a:b", all)), "bazel.run") {
		t.Fatal("a go_library must not offer Bazel > Run")
	}
}

func TestRegistryOffersTestOnlyForTests(t *testing.T) {
	registry := NewRegistry()
	all := tools(toolset.KnownTools...)

	if !has(registry.For(contextFor(target.KindGoTest, "//src/go/a:b_test", all)), "bazel.test") {
		t.Fatal("a go_test should offer Bazel > Test")
	}
	if has(registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", all)), "bazel.test") {
		t.Fatal("a go_binary must not offer Bazel > Test")
	}
}

func TestRegistryAlwaysOffersBuild(t *testing.T) {
	registry := NewRegistry()
	all := tools(toolset.KnownTools...)
	for _, kind := range []target.Kind{
		target.KindGoBinary, target.KindGoLibrary, target.KindGoTest,
		target.KindCcLibrary, "genrule", "filegroup",
	} {
		if !has(registry.For(contextFor(kind, "//src/go/a:b", all)), "bazel.build") {
			t.Fatalf("%q should offer Bazel > Build", kind)
		}
	}
}

func TestRegistryPicksTheRightIDE(t *testing.T) {
	registry := NewRegistry()
	all := tools(toolset.KnownTools...)

	goActions := registry.For(contextFor(target.KindGoLibrary, "//src/go/a:b", all))
	if !has(goActions, "goland.open") || has(goActions, "clion.open") {
		t.Fatalf("a go_library should offer GoLand only, got %v", ids(goActions))
	}

	ccActions := registry.For(contextFor(target.KindCcLibrary, "//src/c/a:b", all))
	if !has(ccActions, "clion.open") || has(ccActions, "goland.open") {
		t.Fatalf("a cc_library should offer CLion only, got %v", ids(ccActions))
	}
}

// The operator's machine has neither IDE installed; this is the normal case.
func TestRegistryHidesActionsForMissingTools(t *testing.T) {
	registry := NewRegistry()
	partial := tools(toolset.ToolBazel, toolset.ToolNano, toolset.ToolClaude)
	got := registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", partial, "README.md"))

	if has(got, "goland.open") || has(got, "clion.open") {
		t.Fatalf("actions for absent tools must be hidden, got %v", ids(got))
	}
	if !has(got, "bazel.run") || !has(got, "claude.dsp") {
		t.Fatalf("actions for present tools must be offered, got %v", ids(got))
	}
}

func TestRegistryOffersNothingBazelWithoutBazel(t *testing.T) {
	registry := NewRegistry()
	got := registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolNano)))
	for _, id := range ids(got) {
		if strings.HasPrefix(id, "bazel.") {
			t.Fatalf("bazel actions offered without bazel: %v", ids(got))
		}
	}
}

func TestRegistryExpandsMarkdownFiles(t *testing.T) {
	registry := NewRegistry()
	all := tools(toolset.KnownTools...)
	got := registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", all, "README.md", "CLAUDE.md"))

	if !has(got, "nano.README.md") || !has(got, "nano.CLAUDE.md") {
		t.Fatalf("both Markdown files should be offered, got %v", ids(got))
	}
}

func TestRegistryOffersNoNanoActionsWithoutMarkdown(t *testing.T) {
	registry := NewRegistry()
	got := registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.KnownTools...)))
	for _, id := range ids(got) {
		if strings.HasPrefix(id, "nano.") {
			t.Fatalf("no Markdown files present, yet %q was offered", id)
		}
	}
}

func TestRegistryOrderIsStable(t *testing.T) {
	registry := NewRegistry()
	all := tools(toolset.KnownTools...)
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", all, "README.md")

	first := ids(registry.For(ctx))
	for i := 0; i < 5; i++ {
		if got := ids(registry.For(ctx)); strings.Join(got, ",") != strings.Join(first, ",") {
			t.Fatalf("order changed between calls: %v then %v", first, got)
		}
	}
	if first[0] != "bazel.run" {
		t.Fatalf("first action = %q, want bazel.run", first[0])
	}
}

// withTests is contextFor plus the fact that something under the package is
// testable, which is what a library or binary needs to be offered the suite.
func withTests(kind target.Kind, label string, set toolset.Set) Context {
	ctx := contextFor(kind, label, set)
	ctx.PackageHasTests = true
	return ctx
}

func TestRegistryOffersAPackagesTestsToItsNonTestTargets(t *testing.T) {
	registry := NewRegistry()

	for _, kind := range []target.Kind{target.KindGoBinary, target.KindGoLibrary, target.KindCcLibrary} {
		actions := registry.For(withTests(kind, "//src/go/a:b", tools(toolset.ToolBazel)))
		if !has(actions, IDBazelTestPackage) {
			t.Errorf("%s beside a test rule was not offered its package's tests", kind)
		}
	}
}

func TestRegistryWithholdsPackageTestsWithoutAny(t *testing.T) {
	registry := NewRegistry()

	actions := registry.For(contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolBazel)))
	if has(actions, IDBazelTestPackage) {
		t.Fatal("a package with nothing testable beneath it was offered its tests")
	}
}

// A test rule already has the precise action; offering both would be two ways
// to do overlapping things with no way to tell them apart.
func TestRegistryOffersOnlyOneTestActionToATestRule(t *testing.T) {
	registry := NewRegistry()

	actions := registry.For(withTests(target.KindGoTest, "//src/go/a:b_test", tools(toolset.ToolBazel)))
	if !has(actions, IDBazelTest) {
		t.Fatal("a test rule lost its own test action")
	}
	if has(actions, IDBazelTestPackage) {
		t.Fatal("a test rule was offered the package wildcard as well")
	}
}

func TestPackageTestCommandRunsTheWildcard(t *testing.T) {
	ctx := withTests(target.KindGoBinary, "//src/go/apps/falcon:linux", tools(toolset.ToolBazel))
	spec := commandOf(t, ctx, IDBazelTestPackage)

	want := "test //src/go/apps/falcon/..."
	if strings.Join(spec.Args, " ") != want {
		t.Fatalf("Args = %v, want %q", spec.Args, want)
	}
	if spec.Dir != "/repo" {
		t.Fatalf("Dir = %q, want the monorepo root", spec.Dir)
	}
}

func TestTestCommandOnATestRuleNamesThatRuleAlone(t *testing.T) {
	ctx := withTests(target.KindGoTest, "//src/go/apps/falcon:falcon_test", tools(toolset.ToolBazel))
	spec := commandOf(t, ctx, IDBazelTest)

	want := "test //src/go/apps/falcon:falcon_test"
	if strings.Join(spec.Args, " ") != want {
		t.Fatalf("Args = %v, want %q", spec.Args, want)
	}
}

// commandOf builds the command for one offered action, failing when it is not
// offered at all.
func commandOf(t *testing.T, ctx Context, id string) port.CommandSpec {
	t.Helper()
	for _, candidate := range NewRegistry().For(ctx) {
		if candidate.ID() != id {
			continue
		}
		spec, err := candidate.Command(ctx)
		if err != nil {
			t.Fatalf("%s Command() error = %v", id, err)
		}
		return spec
	}
	t.Fatalf("%s was not offered for %v", id, ids(NewRegistry().For(ctx)))
	return port.CommandSpec{}
}

func BenchmarkPackageTestCommand(b *testing.B) {
	registry := NewRegistry()
	tgt, _ := target.New(target.KindGoBinary, "//src/go/apps/falcon:linux")
	ctx := Context{Target: tgt, Root: "/repo", Directory: "/repo/src", Tools: tools(toolset.ToolBazel), PackageHasTests: true}
	for b.Loop() {
		registry.For(ctx)
	}
}

func TestBazelCommandRunsFromTheRoot(t *testing.T) {
	registry := NewRegistry()
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolBazel))
	for _, candidate := range registry.For(ctx) {
		if candidate.ID() != "bazel.run" {
			continue
		}
		spec, err := candidate.Command(ctx)
		if err != nil {
			t.Fatalf("Command() error = %v", err)
		}
		if spec.Dir != "/repo" {
			t.Fatalf("Dir = %q, want the monorepo root — labels resolve from there", spec.Dir)
		}
		if spec.Path != "/usr/bin/bazel" {
			t.Fatalf("Path = %q, want the resolved bazel path", spec.Path)
		}
		want := []string{"run", "//src/go/a:b"}
		if strings.Join(spec.Args, " ") != strings.Join(want, " ") {
			t.Fatalf("Args = %v, want %v", spec.Args, want)
		}
		return
	}
	t.Fatal("bazel.run was not offered")
}

func TestClaudeDSPCarriesTheBypassFlag(t *testing.T) {
	registry := NewRegistry()
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolClaude))
	for _, candidate := range registry.For(ctx) {
		if candidate.ID() != "claude.dsp" {
			continue
		}
		spec, err := candidate.Command(ctx)
		if err != nil {
			t.Fatalf("Command() error = %v", err)
		}
		if len(spec.Args) != 1 || spec.Args[0] != dangerousSkipFlag {
			t.Fatalf("Args = %v, want %v", spec.Args, []string{dangerousSkipFlag})
		}
		if spec.Dir != "/repo/src/go/tools/basalt" {
			t.Fatalf("Dir = %q, want the target's own directory", spec.Dir)
		}
		return
	}
	t.Fatal("claude.dsp was not offered")
}

func TestClaudePlainCarriesNoBypassFlag(t *testing.T) {
	registry := NewRegistry()
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolClaude))
	for _, candidate := range registry.For(ctx) {
		if candidate.ID() != "claude.run" {
			continue
		}
		spec, _ := candidate.Command(ctx)
		for _, arg := range spec.Args {
			if arg == dangerousSkipFlag {
				t.Fatal("the plain Claude action must never pass the bypass flag")
			}
		}
		return
	}
	t.Fatal("claude.run was not offered")
}

func TestNanoCommandUsesTheBaseNameOnly(t *testing.T) {
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolNano), "README.md")
	spec, err := newNanoAction("../../../etc/passwd").Command(ctx)
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if len(spec.Args) != 1 || spec.Args[0] != "passwd" {
		t.Fatalf("Args = %v — a traversal in the file name must be reduced to its base name", spec.Args)
	}
}

func TestCommandRefusesAnUnavailableTool(t *testing.T) {
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools())
	if _, err := newClaude().Command(ctx); !errors.Is(err, ErrToolUnavailable) {
		t.Fatalf("Command() error = %v, want ErrToolUnavailable", err)
	}
}

func TestCommandRefusesAnEmptyDirectory(t *testing.T) {
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.ToolClaude))
	ctx.Directory = ""
	if _, err := newClaude().Command(ctx); !errors.Is(err, ErrNoDirectory) {
		t.Fatalf("Command() error = %v, want ErrNoDirectory", err)
	}
}

func BenchmarkRegistryFor(b *testing.B) {
	registry := NewRegistry()
	ctx := contextFor(target.KindGoBinary, "//src/go/a:b", tools(toolset.KnownTools...), "README.md", "CLAUDE.md")
	b.ResetTimer()
	for b.Loop() {
		registry.For(ctx)
	}
}
