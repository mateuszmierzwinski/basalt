package filter

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

func targets(t *testing.T, labels ...string) []target.Target {
	t.Helper()
	built := make([]target.Target, 0, len(labels))
	for _, label := range labels {
		tgt, err := target.New(target.KindGoBinary, label)
		if err != nil {
			t.Fatalf("target.New(%q) error = %v", label, err)
		}
		built = append(built, tgt)
	}
	return built
}

func sample(t *testing.T) []target.Target {
	t.Helper()
	return targets(t,
		"//src/go/apps/falcon:linux",
		"//src/go/apps/falcon:macos",
		"//src/go/apps/falcon_web/cmd/falcon_web:falcon_web",
		"//src/go/tools/basalt/cmd/basalt:basalt",
		"//src/c/mixer:engine",
	)
}

func TestFilterEmptyQueryMatchesEverything(t *testing.T) {
	all := sample(t)
	got := NewMatcher().Filter(all, "", ModeLiteral)
	if got.Err != nil {
		t.Fatalf("Err = %v", got.Err)
	}
	if len(got.Matches) != len(all) {
		t.Fatalf("matched %d, want %d", len(got.Matches), len(all))
	}
	if got.Matches[0].Start != -1 {
		t.Fatal("an unfiltered match should carry no highlight span")
	}
}

func TestFilterLiteral(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "falcon", ModeLiteral)
	if got.Err != nil {
		t.Fatalf("Err = %v", got.Err)
	}
	if len(got.Matches) != 3 {
		t.Fatalf("matched %d, want 3", len(got.Matches))
	}
}

func TestFilterLiteralIsCaseInsensitive(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "FALCON", ModeLiteral)
	if len(got.Matches) != 3 {
		t.Fatalf("matched %d, want 3", len(got.Matches))
	}
}

func TestFilterLiteralReportsSpan(t *testing.T) {
	got := NewMatcher().Filter(targets(t, "//src/go/tools/basalt:basalt"), "tools", ModeLiteral)
	if len(got.Matches) != 1 {
		t.Fatalf("matched %d, want 1", len(got.Matches))
	}
	m := got.Matches[0]
	if m.Target.Label[m.Start:m.End] != "tools" {
		t.Fatalf("span [%d:%d] = %q, want \"tools\"", m.Start, m.End, m.Target.Label[m.Start:m.End])
	}
}

func TestFilterLiteralTreatsRegexAsText(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "(.*)falcon(.*)", ModeLiteral)
	if len(got.Matches) != 0 {
		t.Fatalf("matched %d, want 0 — literal mode must not interpret the pattern", len(got.Matches))
	}
}

// The brief's own example.
func TestFilterRegex(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "(.*)falcon(.*)", ModeRegex)
	if got.Err != nil {
		t.Fatalf("Err = %v", got.Err)
	}
	if len(got.Matches) != 3 {
		t.Fatalf("matched %d, want 3", len(got.Matches))
	}
}

func TestFilterRegexAnchors(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "engine$", ModeRegex)
	if len(got.Matches) != 1 {
		t.Fatalf("matched %d, want 1", len(got.Matches))
	}
}

func TestFilterRegexIsCaseInsensitive(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "FALCON_WEB", ModeRegex)
	if len(got.Matches) != 1 {
		t.Fatalf("matched %d, want 1", len(got.Matches))
	}
}

func TestFilterRegexReportsInvalidPattern(t *testing.T) {
	for _, query := range []string{"(unclosed", "[z-a]", "*", "(?P<", "a{2,1}"} {
		got := NewMatcher().Filter(sample(t), query, ModeRegex)
		if !errors.Is(got.Err, ErrInvalidRegex) {
			t.Fatalf("Filter(%q) Err = %v, want ErrInvalidRegex", query, got.Err)
		}
		if len(got.Matches) != 0 {
			t.Fatalf("Filter(%q) returned matches alongside an error", query)
		}
	}
}

func TestFilterRejectsOversizedQuery(t *testing.T) {
	huge := strings.Repeat("a", MaxQueryLength+1)
	got := NewMatcher().Filter(sample(t), huge, ModeRegex)
	if !errors.Is(got.Err, ErrQueryTooLong) {
		t.Fatalf("Err = %v, want ErrQueryTooLong", got.Err)
	}
}

func TestFilterNoMatches(t *testing.T) {
	got := NewMatcher().Filter(sample(t), "nothing-matches-this", ModeLiteral)
	if got.Err != nil || len(got.Matches) != 0 {
		t.Fatalf("Result = %+v", got)
	}
}

func TestFilterEmptyTargetList(t *testing.T) {
	got := NewMatcher().Filter(nil, "falcon", ModeLiteral)
	if got.Err != nil || len(got.Matches) != 0 {
		t.Fatalf("Result = %+v", got)
	}
}

// RE2 has no catastrophic backtracking, but the guard should hold regardless:
// the classic exponential pattern must complete promptly.
func TestFilterResistsPathologicalPattern(t *testing.T) {
	many := make([]target.Target, 0, 500)
	for i := 0; i < 500; i++ {
		many = append(many, target.Target{Label: "//src/" + strings.Repeat("a", 200) + ":x"})
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		NewMatcher().Filter(many, "(a+)+$", ModeRegex)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a pathological pattern did not complete in 10s")
	}
}

func TestFilterCapsResultSet(t *testing.T) {
	many := make([]target.Target, 0, MaxMatches+100)
	for i := 0; i < MaxMatches+100; i++ {
		many = append(many, target.Target{Label: "//src/go:thing"})
	}
	got := NewMatcher().Filter(many, "thing", ModeLiteral)
	if len(got.Matches) > MaxMatches {
		t.Fatalf("matched %d, want at most %d", len(got.Matches), MaxMatches)
	}
}

func TestCompileQueryLiteralReturnsNoPattern(t *testing.T) {
	pattern, err := CompileQuery("anything", ModeLiteral)
	if err != nil || pattern != nil {
		t.Fatalf("CompileQuery() = %v, %v", pattern, err)
	}
}

func TestLiteralSpan(t *testing.T) {
	if start, end := LiteralSpan("//src/go:falcon", "falcon"); start != 9 || end != 15 {
		t.Fatalf("LiteralSpan() = %d, %d", start, end)
	}
	if start, end := LiteralSpan("//src/go:x", "absent"); start != -1 || end != -1 {
		t.Fatalf("LiteralSpan() = %d, %d, want -1, -1", start, end)
	}
}

func TestRegexSpan(t *testing.T) {
	pattern, err := CompileQuery("fal.on", ModeRegex)
	if err != nil {
		t.Fatalf("CompileQuery() error = %v", err)
	}
	if start, end := RegexSpan("//src/go:falcon", pattern); start != 9 || end != 15 {
		t.Fatalf("RegexSpan() = %d, %d", start, end)
	}
	if start, _ := RegexSpan("//src/go:x", pattern); start != -1 {
		t.Fatalf("RegexSpan() = %d, want -1", start)
	}
}

func benchTargets() []target.Target {
	built := make([]target.Target, 0, 1300)
	for i := 0; i < 1300; i++ {
		built = append(built, target.Target{Label: "//src/go/tools/basalt/internal/domain/filter:matcher"})
	}
	return built
}

func BenchmarkCompileQuery(b *testing.B) {
	for b.Loop() {
		CompileQuery("(.*)falcon(.*)", ModeRegex)
	}
}

func BenchmarkLiteralSpan(b *testing.B) {
	for b.Loop() {
		LiteralSpan("//src/go/apps/falcon:linux", "falcon")
	}
}

func BenchmarkRegexSpan(b *testing.B) {
	pattern, _ := CompileQuery("falcon", ModeRegex)
	for b.Loop() {
		RegexSpan("//src/go/apps/falcon:linux", pattern)
	}
}

// Sized to the real monorepo: this runs on every keystroke.
func BenchmarkFilterLiteralAtScale(b *testing.B) {
	all := benchTargets()
	matcher := NewMatcher()
	b.ResetTimer()
	for b.Loop() {
		matcher.Filter(all, "filter", ModeLiteral)
	}
}

func BenchmarkFilterRegexAtScale(b *testing.B) {
	all := benchTargets()
	matcher := NewMatcher()
	b.ResetTimer()
	for b.Loop() {
		matcher.Filter(all, "(.*)filter(.*)", ModeRegex)
	}
}
