package workspace

import (
	"path/filepath"
	"testing"
)

func TestNormalizeRoot(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"trailing separator is dropped", "/home/u/Monorepo/", "/home/u/Monorepo"},
		{"already clean is unchanged", "/home/u/Monorepo", "/home/u/Monorepo"},
		{"surrounding whitespace is trimmed", "  /home/u/Monorepo/  ", "/home/u/Monorepo"},
		{"redundant segments collapse", "/home/u/./Monorepo//x/..", "/home/u/Monorepo"},
		{"empty stays empty", "", ""},
		{"whitespace only stays empty", "   ", ""},
		{"relative path is preserved as relative", "Monorepo/", "Monorepo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeRoot(tc.raw); got != tc.want {
				t.Fatalf("NormalizeRoot(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestNormalizeRootPreventsDoubledSeparators(t *testing.T) {
	root := NormalizeRoot("/home/u/Monorepo/")
	if got := SourceDirectory(root); got != filepath.Join("/home/u/Monorepo", "src") {
		t.Fatalf("SourceDirectory = %q", got)
	}
}

func TestSourceDirectory(t *testing.T) {
	if got := SourceDirectory("/repo"); got != "/repo/src" {
		t.Fatalf("SourceDirectory = %q, want /repo/src", got)
	}
}

func TestCacheDirectory(t *testing.T) {
	if got := CacheDirectory("/repo"); got != "/repo/basalt_target_cache" {
		t.Fatalf("CacheDirectory = %q, want /repo/basalt_target_cache", got)
	}
}

func BenchmarkNormalizeRoot(b *testing.B) {
	for b.Loop() {
		NormalizeRoot("  /home/u/Monorepo//./  ")
	}
}

func BenchmarkSourceDirectory(b *testing.B) {
	for b.Loop() {
		SourceDirectory("/home/u/Monorepo")
	}
}

func BenchmarkCacheDirectory(b *testing.B) {
	for b.Loop() {
		CacheDirectory("/home/u/Monorepo")
	}
}
