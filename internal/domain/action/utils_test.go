package action

import (
	"reflect"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

func TestIsRunnable(t *testing.T) {
	cases := []struct {
		kind target.Kind
		want bool
	}{
		{target.KindGoBinary, true},
		{target.KindCcBinary, true},
		{"sh_binary", true},
		{target.KindGoLibrary, false},
		{target.KindGoTest, false},
		{"genrule", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsRunnable(tc.kind); got != tc.want {
			t.Fatalf("IsRunnable(%q) = %t, want %t", tc.kind, got, tc.want)
		}
	}
}

func TestIsTestable(t *testing.T) {
	cases := []struct {
		kind target.Kind
		want bool
	}{
		{target.KindGoTest, true},
		{target.KindCcTest, true},
		{"sh_test", true},
		{target.KindGoBinary, false},
		{target.KindGoLibrary, false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsTestable(tc.kind); got != tc.want {
			t.Fatalf("IsTestable(%q) = %t, want %t", tc.kind, got, tc.want)
		}
	}
}

func TestIsGo(t *testing.T) {
	for kind, want := range map[target.Kind]bool{
		target.KindGoBinary:  true,
		target.KindGoLibrary: true,
		target.KindGoTest:    true,
		target.KindCcBinary:  false,
		"genrule":            false,
	} {
		if got := IsGo(kind); got != want {
			t.Fatalf("IsGo(%q) = %t, want %t", kind, got, want)
		}
	}
}

func TestIsNative(t *testing.T) {
	for kind, want := range map[target.Kind]bool{
		target.KindCcBinary:  true,
		target.KindCcLibrary: true,
		target.KindCcTest:    true,
		target.KindGoBinary:  false,
		"proto_library":      false,
	} {
		if got := IsNative(kind); got != want {
			t.Fatalf("IsNative(%q) = %t, want %t", kind, got, want)
		}
	}
}

func TestIsMarkdown(t *testing.T) {
	for name, want := range map[string]bool{
		"README.md":  true,
		"claude.MD":  true,
		"PLAN.Md":    true,
		"BUILD":      false,
		"main.go":    false,
		"notes.txt":  false,
		"md":         false,
		"README.mdx": false,
	} {
		if got := IsMarkdown(name); got != want {
			t.Fatalf("IsMarkdown(%q) = %t, want %t", name, got, want)
		}
	}
}

func TestSelectMarkdown(t *testing.T) {
	got := SelectMarkdown([]string{"BUILD", "README.md", "main.go", "CLAUDE.md", "coverage.out"})
	want := []string{"README.md", "CLAUDE.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectMarkdown() = %v, want %v", got, want)
	}
}

func TestSelectMarkdownOnEmptyInput(t *testing.T) {
	if got := SelectMarkdown(nil); len(got) != 0 {
		t.Fatalf("SelectMarkdown(nil) = %v, want empty", got)
	}
}

func BenchmarkIsRunnable(b *testing.B) {
	for b.Loop() {
		IsRunnable(target.KindGoBinary)
	}
}

func BenchmarkIsTestable(b *testing.B) {
	for b.Loop() {
		IsTestable(target.KindGoTest)
	}
}

func BenchmarkIsGo(b *testing.B) {
	for b.Loop() {
		IsGo(target.KindGoLibrary)
	}
}

func BenchmarkIsNative(b *testing.B) {
	for b.Loop() {
		IsNative(target.KindCcLibrary)
	}
}

func BenchmarkIsMarkdown(b *testing.B) {
	for b.Loop() {
		IsMarkdown("README.md")
	}
}

func BenchmarkSelectMarkdown(b *testing.B) {
	names := []string{"BUILD", "README.md", "main.go", "CLAUDE.md", "PLAN.md", "coverage.out"}
	for b.Loop() {
		SelectMarkdown(names)
	}
}
