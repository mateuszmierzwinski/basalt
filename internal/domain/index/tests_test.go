package index

import (
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

func indexOf(entries ...target.Target) Index {
	return Index{Targets: entries}
}

func rule(kind target.Kind, pkg, name string) target.Target {
	return target.Target{Kind: kind, Package: pkg, Name: name, Label: pkg + ":" + name}
}

func populated() TestIndex {
	tests := NewTestIndex()
	tests.Replace(indexOf(
		rule(target.KindGoBinary, "//src/go/tools/basalt/cmd/basalt", "basalt"),
		rule(target.KindGoTest, "//src/go/tools/basalt/internal/domain/tui", "tui_test"),
		rule(target.KindCcBinary, "//src/c/mixer", "mixer"),
	))
	return tests
}

// A wildcard at any level above a test reaches it, which is exactly what the
// action offers to run.
func TestHasTestsUnderCoversEveryAncestor(t *testing.T) {
	tests := populated()

	for _, pkg := range []string{
		"//src",
		"//src/go",
		"//src/go/tools/basalt",
		"//src/go/tools/basalt/internal",
		"//src/go/tools/basalt/internal/domain",
		"//src/go/tools/basalt/internal/domain/tui",
	} {
		if !tests.HasTestsUnder(pkg) {
			t.Errorf("%s reports no tests beneath it", pkg)
		}
	}
}

func TestHasTestsUnderRejectsPackagesWithoutAny(t *testing.T) {
	tests := populated()

	for _, pkg := range []string{
		"//src/c/mixer",
		"//src/go/tools/basalt/cmd/basalt",
		"//src/go/tools/basalt/internal/domain/layout",
		"//src/go/ai",
	} {
		if tests.HasTestsUnder(pkg) {
			t.Errorf("%s reports tests beneath it, but there are none", pkg)
		}
	}
}

// A sibling whose name merely starts with the same characters is not beneath
// it: "//src/go/tools/basalt2" must not inherit basalt's tests.
func TestHasTestsUnderIsNotAStringPrefixMatch(t *testing.T) {
	tests := NewTestIndex()
	tests.Replace(indexOf(rule(target.KindGoTest, "//src/go/tools/basalt", "basalt_test")))

	if tests.HasTestsUnder("//src/go/tools/basalt2") {
		t.Fatal("a package sharing a name prefix inherited another's tests")
	}
}

// The walk stops at //src: selecting one target must never offer to test the
// entire monorepo.
func TestHasTestsUnderStopsAtTheSourceRoot(t *testing.T) {
	tests := populated()

	if tests.HasTestsUnder("//") {
		t.Fatal("the whole-monorepo wildcard was reported as covered")
	}
}

func TestReplaceForgetsThePreviousTargetSet(t *testing.T) {
	tests := NewTestIndex()
	tests.Replace(indexOf(rule(target.KindGoTest, "//src/go/gone", "gone_test")))
	tests.Replace(indexOf(rule(target.KindGoTest, "//src/go/kept", "kept_test")))

	if tests.HasTestsUnder("//src/go/gone") {
		t.Fatal("a package removed by a re-index still reports tests")
	}
	if !tests.HasTestsUnder("//src/go/kept") {
		t.Fatal("a package added by a re-index reports no tests")
	}
}

func TestEmptyIndexReportsNothing(t *testing.T) {
	if NewTestIndex().HasTestsUnder("//src/go/tools/basalt") {
		t.Fatal("an unpopulated index claimed to know of tests")
	}
}

func BenchmarkReplace(b *testing.B) {
	entries := make([]target.Target, 0, 1400)
	for i := 0; i < 1400; i++ {
		entries = append(entries, rule(target.KindGoTest, "//src/go/tools/basalt/internal/domain/tui", "tui_test"))
	}
	built := indexOf(entries...)
	tests := NewTestIndex()
	b.ResetTimer()
	for b.Loop() {
		tests.Replace(built)
	}
}

func BenchmarkHasTestsUnder(b *testing.B) {
	tests := populated()
	b.ResetTimer()
	for b.Loop() {
		tests.HasTestsUnder("//src/go/tools/basalt/internal/domain/tui")
	}
}
