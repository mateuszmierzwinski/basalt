package session

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/action"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
)

// fakeStore is a hand-written double: the FileStore surface the provider uses
// is one method, and a fake reads better here than a generated mock.
type fakeStore struct {
	names map[string][]string
	err   error
}

func (f fakeStore) Read(string) ([]byte, error)             { return nil, fs.ErrNotExist }
func (f fakeStore) Write(string, []byte, fs.FileMode) error { return nil }
func (f fakeStore) MkdirAll(string, fs.FileMode) error      { return nil }
func (f fakeStore) ListNames(dir string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.names[dir], nil
}

func allTools() toolset.Set {
	entries := make(map[toolset.Tool]toolset.Availability, len(toolset.KnownTools))
	for _, tool := range toolset.KnownTools {
		entries[tool] = toolset.Availability{Tool: tool, Path: "/usr/bin/" + string(tool), Found: true}
	}
	return toolset.NewSet(entries)
}

func newProvider(t *testing.T, store fakeStore) Actions {
	t.Helper()
	return NewProvider(ProviderConfig{Registry: action.NewRegistry(), Store: store, Tools: allTools(), Root: "/repo"})
}

func mustTarget(t *testing.T, kind target.Kind, label string) target.Target {
	t.Helper()
	tgt, err := target.New(kind, label)
	if err != nil {
		t.Fatalf("target.New(%q) error = %v", label, err)
	}
	return tgt
}

func TestContextResolvesTheDirectory(t *testing.T) {
	provider := newProvider(t, fakeStore{})
	got, err := provider.Context(mustTarget(t, target.KindGoBinary, "//src/go/tools/basalt:basalt"))
	if err != nil {
		t.Fatalf("Context() error = %v", err)
	}
	if got.Directory != "/repo/src/go/tools/basalt" {
		t.Fatalf("Directory = %q", got.Directory)
	}
	if got.Root != "/repo" {
		t.Fatalf("Root = %q", got.Root)
	}
}

func TestContextFindsMarkdownFiles(t *testing.T) {
	store := fakeStore{names: map[string][]string{
		"/repo/src/go/tools/basalt": {"BUILD", "README.md", "main.go", "CLAUDE.md"},
	}}
	got, err := newProvider(t, store).Context(mustTarget(t, target.KindGoBinary, "//src/go/tools/basalt:basalt"))
	if err != nil {
		t.Fatalf("Context() error = %v", err)
	}
	if len(got.Markdown) != 2 {
		t.Fatalf("Markdown = %v, want two files", got.Markdown)
	}
}

func TestContextToleratesAnUnreadableDirectory(t *testing.T) {
	store := fakeStore{err: fs.ErrPermission}
	got, err := newProvider(t, store).Context(mustTarget(t, target.KindGoBinary, "//src/go/a:b"))
	if err != nil {
		t.Fatalf("Context() error = %v — an unreadable package is not fatal", err)
	}
	if len(got.Markdown) != 0 {
		t.Fatalf("Markdown = %v, want none", got.Markdown)
	}
}

// The last line of defence: even a poisoned entry that somehow reached the UI
// cannot resolve to a path outside the monorepo.
func TestContextRefusesAnEscapingLabel(t *testing.T) {
	poisoned := target.Target{
		Kind: target.KindGoBinary, Package: "//../../etc",
		Name: "passwd", Label: "//../../etc:passwd", Display: "[etc:passwd]",
	}
	if _, err := newProvider(t, fakeStore{}).Context(poisoned); !errors.Is(err, target.ErrLabelEscapesRoot) {
		t.Fatalf("Context() error = %v, want ErrLabelEscapesRoot", err)
	}
}

func TestActionsAreEmptyForAnEscapingLabel(t *testing.T) {
	poisoned := target.Target{
		Kind: target.KindGoBinary, Package: "//../../etc",
		Name: "passwd", Label: "//../../etc:passwd", Display: "[etc:passwd]",
	}
	if got := newProvider(t, fakeStore{}).Actions(poisoned); len(got) != 0 {
		t.Fatalf("Actions() = %d entries, want none for a label outside the monorepo", len(got))
	}
}

func TestActionsReflectTheRuleKind(t *testing.T) {
	provider := newProvider(t, fakeStore{})

	binary := provider.Actions(mustTarget(t, target.KindGoBinary, "//src/go/a:b"))
	library := provider.Actions(mustTarget(t, target.KindGoLibrary, "//src/go/a:lib"))

	if !containsLabel(binary, "Bazel > Run") {
		t.Fatal("a go_binary should offer Run")
	}
	if containsLabel(library, "Bazel > Run") {
		t.Fatal("a go_library must not offer Run")
	}
}

func containsLabel(actions []action.Action, want string) bool {
	for _, entry := range actions {
		if entry.Label() == want {
			return true
		}
	}
	return false
}

func BenchmarkProviderActions(b *testing.B) {
	store := fakeStore{names: map[string][]string{
		"/repo/src/go/a": {"BUILD", "README.md", "main.go"},
	}}
	provider := NewProvider(ProviderConfig{Registry: action.NewRegistry(), Store: store, Tools: allTools(), Root: "/repo"})
	tgt, _ := target.New(target.KindGoBinary, "//src/go/a:b")
	b.ResetTimer()
	for b.Loop() {
		provider.Actions(tgt)
	}
}
