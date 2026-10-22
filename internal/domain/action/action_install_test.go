package action

import (
	"errors"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
)

const localBin = "/home/u/.local/bin"

func installContext(kind target.Kind, set toolset.Set, dir string) Context {
	ctx := contextFor(kind, "//src/go/apps/falcon:falcon", set)
	ctx.LocalBin = dir
	return ctx
}

func TestInstallIsOfferedForBinaries(t *testing.T) {
	ctx := installContext(target.KindGoBinary, tools(toolset.KnownTools...), localBin)
	if !has(NewRegistry().For(ctx), IDInstallLocal) {
		t.Fatalf("a go_binary should offer the copy, got %v", ids(NewRegistry().For(ctx)))
	}
}

func TestInstallIsWithheld(t *testing.T) {
	all := tools(toolset.KnownTools...)
	cases := map[string]Context{
		"library":         installContext(target.KindGoLibrary, all, localBin),
		"test":            installContext(target.KindGoTest, all, localBin),
		"no install tool": installContext(target.KindGoBinary, tools(toolset.ToolBazel), localBin),
		"no bazel":        installContext(target.KindGoBinary, tools(toolset.ToolInstall), localBin),
		"no home":         installContext(target.KindGoBinary, all, ""),
		"relative":        installContext(target.KindGoBinary, all, ".local/bin"),
		"quote in path":   installContext(target.KindGoBinary, all, "/home/o'brien/.local/bin"),
	}
	for name, ctx := range cases {
		if has(NewRegistry().For(ctx), IDInstallLocal) {
			t.Errorf("%s: the copy must not be offered", name)
		}
	}
}

func TestInstallCommandIsOneBazelRun(t *testing.T) {
	ctx := installContext(target.KindGoBinary, tools(toolset.KnownTools...), localBin)
	spec := commandOf(t, ctx, IDInstallLocal)
	want := []string{"run", "--run_under='/usr/bin/install' -D -m 0755 -t '/home/u/.local/bin'", "//src/go/apps/falcon:falcon"}
	if spec.Path != "/usr/bin/bazel" || spec.Dir != "/repo" || !equalArgs(spec.Args, want) {
		t.Fatalf("Command() = %+v, want bazel %v from /repo", spec, want)
	}
}

func TestInstallCommandRefusesWhatItWouldNotOffer(t *testing.T) {
	all := tools(toolset.KnownTools...)
	if _, err := newInstallLocal().Command(installContext(target.KindGoBinary, all, "")); !errors.Is(err, ErrNoDestination) {
		t.Fatalf("no destination: error = %v, want ErrNoDestination", err)
	}
	noInstall := installContext(target.KindGoBinary, tools(toolset.ToolBazel), localBin)
	if _, err := newInstallLocal().Command(noInstall); !errors.Is(err, ErrToolUnavailable) {
		t.Fatalf("no install: error = %v, want ErrToolUnavailable", err)
	}
}

func TestInstallSitsAfterBuildAndBeforeTheEditors(t *testing.T) {
	ctx := installContext(target.KindGoBinary, tools(toolset.KnownTools...), localBin)
	ctx.Markdown = []string{"README.md"}
	got := ids(NewRegistry().For(ctx))
	build, install, nano := indexOf(got, IDBazelBuild), indexOf(got, IDInstallLocal), indexOf(got, "nano.README.md")
	if build < 0 || install != build+1 || nano != install+1 {
		t.Fatalf("panel order = %v, want build, copy, then the editors", got)
	}
}

func TestIsQuotableDirectory(t *testing.T) {
	cases := map[string]bool{
		localBin:                   true,
		"/home/a b/.local/bin":     true,
		"":                         false,
		"relative/bin":             false,
		"/home/o'brien/.local/bin": false,
		"/home/x\n/bin":            false,
	}
	for dir, want := range cases {
		if got := IsQuotableDirectory(dir); got != want {
			t.Errorf("IsQuotableDirectory(%q) = %v, want %v", dir, got, want)
		}
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func indexOf(list []string, id string) int {
	for i, candidate := range list {
		if candidate == id {
			return i
		}
	}
	return -1
}

func BenchmarkRunUnderInstall(b *testing.B) {
	for b.Loop() {
		RunUnderInstall("/usr/bin/install", localBin)
	}
}

func BenchmarkIsQuotableDirectory(b *testing.B) {
	for b.Loop() {
		IsQuotableDirectory(localBin)
	}
}
