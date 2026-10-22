package toolset

import (
	"reflect"
	"testing"
)

func fullSet() Set {
	entries := make(map[Tool]Availability, len(KnownTools))
	for _, tool := range KnownTools {
		entries[tool] = Availability{Tool: tool, Path: "/usr/bin/" + string(tool), Found: true}
	}
	return NewSet(entries)
}

func TestSetHas(t *testing.T) {
	set := NewSet(map[Tool]Availability{
		ToolBazel: {Tool: ToolBazel, Path: "/usr/local/bin/bazel", Found: true},
		ToolNano:  {Tool: ToolNano, Found: false},
	})
	if !set.Has(ToolBazel) {
		t.Fatal("Has(bazel) = false, want true")
	}
	if set.Has(ToolNano) {
		t.Fatal("Has(nano) = true, want false")
	}
	if set.Has(ToolClion) {
		t.Fatal("Has(clion) = true for an unprobed tool, want false")
	}
}

func TestSetPathOf(t *testing.T) {
	set := NewSet(map[Tool]Availability{
		ToolBazel: {Tool: ToolBazel, Path: "/usr/local/bin/bazel", Found: true},
	})
	if got := set.PathOf(ToolBazel); got != "/usr/local/bin/bazel" {
		t.Fatalf("PathOf(bazel) = %q", got)
	}
	if got := set.PathOf(ToolGoland); got != "" {
		t.Fatalf("PathOf(goland) = %q, want empty", got)
	}
}

func TestSetMissingReportsInKnownOrder(t *testing.T) {
	set := NewSet(map[Tool]Availability{
		ToolBazel:   {Tool: ToolBazel, Path: "/usr/local/bin/bazel", Found: true},
		ToolNano:    {Tool: ToolNano, Path: "/usr/bin/nano", Found: true},
		ToolLess:    {Tool: ToolLess, Path: "/usr/bin/less", Found: true},
		ToolClaude:  {Tool: ToolClaude, Found: false},
		ToolGoland:  {Tool: ToolGoland, Found: false},
		ToolClion:   {Tool: ToolClion, Found: false},
		ToolInstall: {Tool: ToolInstall, Path: "/usr/bin/install", Found: true},
	})
	want := []Tool{ToolClaude, ToolGoland, ToolClion}
	if got := set.Missing(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Missing() = %v, want %v", got, want)
	}
}

func TestSetMissingIsEmptyWhenAllPresent(t *testing.T) {
	if got := fullSet().Missing(); len(got) != 0 {
		t.Fatalf("Missing() = %v, want empty", got)
	}
}

func TestNewSetCopiesEntries(t *testing.T) {
	entries := map[Tool]Availability{ToolBazel: {Tool: ToolBazel, Found: true}}
	set := NewSet(entries)
	entries[ToolBazel] = Availability{Tool: ToolBazel, Found: false}
	if !set.Has(ToolBazel) {
		t.Fatal("mutating the source map changed the snapshot")
	}
}

func BenchmarkSetHas(b *testing.B) {
	set := fullSet()
	for b.Loop() {
		set.Has(ToolBazel)
	}
}

func BenchmarkSetPathOf(b *testing.B) {
	set := fullSet()
	for b.Loop() {
		set.PathOf(ToolBazel)
	}
}

func BenchmarkSetMissing(b *testing.B) {
	set := NewSet(map[Tool]Availability{ToolBazel: {Tool: ToolBazel, Found: true}})
	for b.Loop() {
		set.Missing()
	}
}

func BenchmarkNewSet(b *testing.B) {
	entries := map[Tool]Availability{
		ToolBazel: {Tool: ToolBazel, Found: true},
		ToolNano:  {Tool: ToolNano, Found: true},
	}
	for b.Loop() {
		NewSet(entries)
	}
}
