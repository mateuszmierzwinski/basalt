package runlog

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePathAndLogsDirectory(t *testing.T) {
	if got := FilePath("/repo/basalt_target_cache"); got != "/repo/basalt_target_cache/history.json" {
		t.Fatalf("FilePath() = %q", got)
	}
	if got := LogsDirectory("/repo/basalt_target_cache"); got != "/repo/basalt_target_cache/logs" {
		t.Fatalf("LogsDirectory() = %q", got)
	}
}

func TestLogFileNameIsDerivedFromTheLabel(t *testing.T) {
	name := LogFileName("//src/go/tools/basalt/cmd/basalt:basalt")
	if !strings.HasSuffix(name, ".log") {
		t.Fatalf("LogFileName() = %q, want a .log file", name)
	}
	if !strings.Contains(name, "src") || !strings.Contains(name, "basalt") {
		t.Fatalf("LogFileName() = %q, want it to resemble the label", name)
	}
}

// The naming scheme is pinned, not merely self-consistent.
//
// Comparing LogFileName(x) to LogFileName(x) would be a tautology for a pure
// function and could never fail. What actually matters is that the scheme does
// not drift between releases: a changed one orphans every log already on disk,
// so each target would silently appear to have never been run.
func TestLogFileNameSchemeIsPinned(t *testing.T) {
	cases := map[string]string{
		"//src/go/demo:passes":                    "src_go_demo_passes-4da4700ceec8.log",
		"//src/go/tools/basalt/cmd/basalt:basalt": "src_go_tools_basalt_cmd_basalt_basalt-16edaceee212.log",
	}
	for label, want := range cases {
		if got := LogFileName(label); got != want {
			t.Fatalf("LogFileName(%q) = %q, want %q — the naming scheme changed, which "+
				"orphans every log already on disk", label, got, want)
		}
	}
}

// Two labels that flatten to the same characters must still get their own file,
// or one target's log would be shown under another's name.
func TestLogFileNameSeparatesLabelsThatFlattenAlike(t *testing.T) {
	first := LogFileName("//src/go/a:b")
	second := LogFileName("//src/go/a_b")
	third := LogFileName("//src:go/a:b")
	if first == second || first == third || second == third {
		t.Fatalf("collisions: %q, %q, %q", first, second, third)
	}
}

// A label is validated before it ever reaches here, but the file name is a path
// component — so it must be inert regardless of what it is handed.
func TestLogFileNameNeverEscapesItsDirectory(t *testing.T) {
	hostile := []string{
		"//../../etc:passwd",
		"//src/go:../../../../etc/shadow",
		"//src/go:a/../../b",
		"//src/go:$(id)",
		"//src/go:a;rm -rf /",
		"//src/go:\x1b[2J",
		"//src/go:..",
	}
	for _, label := range hostile {
		name := LogFileName(label)
		if strings.ContainsAny(name, "/\\") {
			t.Fatalf("LogFileName(%q) = %q contains a separator", label, name)
		}
		if strings.Contains(name, "..") {
			t.Fatalf("LogFileName(%q) = %q contains a traversal", label, name)
		}
		resolved := filepath.Clean(filepath.Join("/cache/logs", name))
		if !strings.HasPrefix(resolved, "/cache/logs/") {
			t.Fatalf("LogFileName(%q) resolved to %q", label, resolved)
		}
	}
}

func TestLogPathIsInsideTheLogsDirectory(t *testing.T) {
	path := LogPath("/repo/cache", "//src/go/a:b")
	if !IsInside(LogsDirectory("/repo/cache"), path) {
		t.Fatalf("LogPath() = %q, outside the logs directory", path)
	}
}

func TestIsInside(t *testing.T) {
	cases := []struct {
		name      string
		dir, path string
		want      bool
	}{
		{"direct child", "/cache/logs", "/cache/logs/a.log", true},
		{"nested child", "/cache/logs", "/cache/logs/x/a.log", true},
		{"the directory itself", "/cache/logs", "/cache/logs", false},
		{"parent", "/cache/logs", "/cache", false},
		{"sibling", "/cache/logs", "/cache/other/a.log", false},
		{"traversal", "/cache/logs", "/cache/logs/../../etc/passwd", false},
		{"absolute elsewhere", "/cache/logs", "/etc/shadow", false},
		{"prefix but not a child", "/cache/logs", "/cache/logs-evil/a.log", false},
		{"unclean but inside", "/cache/logs", "/cache/logs/./a.log", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsInside(tc.dir, tc.path); got != tc.want {
				t.Fatalf("IsInside(%q, %q) = %t, want %t", tc.dir, tc.path, got, tc.want)
			}
		})
	}
}

func TestStatusOf(t *testing.T) {
	if got := StatusOf(nil); got != StatusPassed {
		t.Fatalf("StatusOf(nil) = %v, want passed", got)
	}
	if got := StatusOf(errSample); got != StatusFailed {
		t.Fatalf("StatusOf(err) = %v, want failed", got)
	}
}

var errSample = sampleError{}

type sampleError struct{}

func (sampleError) Error() string { return "boom" }

func TestHistoryStatuses(t *testing.T) {
	history := History{Entries: map[string]Entry{
		"//a:b": {Label: "//a:b", Status: StatusPassed},
		"//c:d": {Label: "//c:d", Status: StatusFailed},
	}}
	statuses := history.Statuses()
	if statuses["//a:b"] != StatusPassed || statuses["//c:d"] != StatusFailed {
		t.Fatalf("Statuses() = %v", statuses)
	}
	if statuses["//never:run"] != StatusUnknown {
		t.Fatal("an absent target should read as unknown")
	}
}

func BenchmarkLogFileName(b *testing.B) {
	for b.Loop() {
		LogFileName("//src/go/tools/basalt/cmd/basalt:basalt")
	}
}

func BenchmarkIsInside(b *testing.B) {
	for b.Loop() {
		IsInside("/repo/cache/logs", "/repo/cache/logs/a.log")
	}
}

func BenchmarkLogPath(b *testing.B) {
	for b.Loop() {
		LogPath("/repo/cache", "//src/go/a:b")
	}
}

func BenchmarkStatuses(b *testing.B) {
	history := History{Entries: map[string]Entry{"//a:b": {Label: "//a:b", Status: StatusPassed}}}
	for b.Loop() {
		history.Statuses()
	}
}

func BenchmarkStatusOf(b *testing.B) {
	for b.Loop() {
		StatusOf(nil)
	}
}
