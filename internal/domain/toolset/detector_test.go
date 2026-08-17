package toolset

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/port/mocks"
	"go.uber.org/mock/gomock"
)

func TestDetectFindsEveryTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	for _, tool := range KnownTools {
		env.EXPECT().LookPath(string(tool)).Return("/usr/bin/"+string(tool), nil)
	}

	set, err := NewDetector(env).Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got := set.Missing(); len(got) != 0 {
		t.Fatalf("Missing() = %v, want empty", got)
	}
}

// The operator's machine has no GoLand or CLion binary, so this is the ordinary
// case rather than an edge case: the IDE actions are hidden and basalt starts.
func TestDetectToleratesMissingIDEs(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	env.EXPECT().LookPath("bazel").Return("/usr/local/bin/bazel", nil)
	env.EXPECT().LookPath("nano").Return("/usr/bin/nano", nil)
	env.EXPECT().LookPath("less").Return("/usr/bin/less", nil)
	env.EXPECT().LookPath("claude").Return("/home/u/.local/bin/claude", nil)
	env.EXPECT().LookPath("goland").Return("", exec.ErrNotFound)
	env.EXPECT().LookPath("clion").Return("", exec.ErrNotFound)

	set, err := NewDetector(env).Detect()
	if err != nil {
		t.Fatalf("Detect() error = %v, want nil", err)
	}
	if set.Has(ToolGoland) || set.Has(ToolClion) {
		t.Fatal("an absent IDE was reported as available")
	}
	if !set.Has(ToolBazel) {
		t.Fatal("bazel should still be available")
	}
}

func TestDetectFailsWithoutBazel(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	env.EXPECT().LookPath("bazel").Return("", exec.ErrNotFound)
	env.EXPECT().LookPath("nano").Return("/usr/bin/nano", nil)
	env.EXPECT().LookPath("less").Return("/usr/bin/less", nil)
	env.EXPECT().LookPath("claude").Return("", exec.ErrNotFound)
	env.EXPECT().LookPath("goland").Return("", exec.ErrNotFound)
	env.EXPECT().LookPath("clion").Return("", exec.ErrNotFound)

	set, err := NewDetector(env).Detect()
	if !errors.Is(err, ErrBazelMissing) {
		t.Fatalf("Detect() error = %v, want ErrBazelMissing", err)
	}
	if !set.Has(ToolNano) {
		t.Fatal("the populated set should still be returned alongside the error")
	}
}
