package fswatch

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestIsRelevant(t *testing.T) {
	cases := []struct {
		name string
		path string
		op   fsnotify.Op
		want bool
	}{
		{"BUILD written", "/repo/src/go/a/BUILD", fsnotify.Write, true},
		{"BUILD.bazel written", "/repo/src/go/a/BUILD.bazel", fsnotify.Write, true},
		{"directory created", "/repo/src/go/newpkg", fsnotify.Create, true},
		{"directory removed", "/repo/src/go/oldpkg", fsnotify.Remove, true},
		{"path renamed", "/repo/src/go/a", fsnotify.Rename, true},
		{"go source written", "/repo/src/go/a/main.go", fsnotify.Write, false},
		{"markdown written", "/repo/src/go/a/README.md", fsnotify.Write, false},
		{"chmod on a source file", "/repo/src/go/a/main.go", fsnotify.Chmod, false},
		{"BUILD chmod still counts", "/repo/src/go/a/BUILD", fsnotify.Chmod, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRelevant(tc.path, tc.op); got != tc.want {
				t.Fatalf("IsRelevant(%q, %v) = %t, want %t", tc.path, tc.op, got, tc.want)
			}
		})
	}
}

// A Dropbox sync rewriting Go sources must not trigger a re-index.
func TestIsRelevantIgnoresSourceChurn(t *testing.T) {
	churn := []string{"main.go", "README.md", "coverage.out", ".DS_Store", "notes.txt"}
	for _, name := range churn {
		if IsRelevant("/repo/src/go/a/"+name, fsnotify.Write) {
			t.Fatalf("IsRelevant(%q, Write) = true, want false", name)
		}
	}
}

func TestIsSkippedDir(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{".git", true},
		{".scannerwork", true},
		{"node_modules", true},
		{".bazel", true},
		{".bazelbsp", true},
		{"src", false},
		{"internal", false},
		{"go", false},
	}
	for _, tc := range cases {
		if got := IsSkippedDir(tc.name); got != tc.want {
			t.Fatalf("IsSkippedDir(%q) = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestIsDirectory(t *testing.T) {
	if !isDirectory(t.TempDir()) {
		t.Fatal("isDirectory(tempdir) = false")
	}
	if isDirectory("/nonexistent/basalt/path") {
		t.Fatal("isDirectory(absent) = true")
	}
}

func BenchmarkIsRelevant(b *testing.B) {
	for b.Loop() {
		IsRelevant("/repo/src/go/a/main.go", fsnotify.Write)
	}
}

func BenchmarkIsSkippedDir(b *testing.B) {
	for b.Loop() {
		IsSkippedDir("internal")
	}
}
