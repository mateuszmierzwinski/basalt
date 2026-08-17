package workspace

import (
	"errors"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/port/mocks"
	"go.uber.org/mock/gomock"
)

func TestResolveSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	probe := mocks.NewMockDirectoryProbe(ctrl)

	env.EXPECT().Lookup("MONOREPO").Return("/repo/", true)
	probe.EXPECT().IsDirectory("/repo").Return(true)
	probe.EXPECT().IsDirectory("/repo/src").Return(true)

	got, err := NewResolver(env, probe).Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Root != "/repo" || got.SourceRoot != "/repo/src" || got.CacheDir != "/repo/basalt_target_cache" {
		t.Fatalf("Resolve() = %+v", got)
	}
}

func TestResolveRejectsUnsetMonorepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	env.EXPECT().Lookup("MONOREPO").Return("", false)

	_, err := NewResolver(env, mocks.NewMockDirectoryProbe(ctrl)).Resolve()
	if !errors.Is(err, ErrMonorepoUnset) {
		t.Fatalf("Resolve() error = %v, want ErrMonorepoUnset", err)
	}
}

func TestResolveRejectsEmptyMonorepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	env.EXPECT().Lookup("MONOREPO").Return("   ", true)

	_, err := NewResolver(env, mocks.NewMockDirectoryProbe(ctrl)).Resolve()
	if !errors.Is(err, ErrMonorepoUnset) {
		t.Fatalf("Resolve() error = %v, want ErrMonorepoUnset", err)
	}
}

func TestResolveRejectsRelativeMonorepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	env.EXPECT().Lookup("MONOREPO").Return("relative/repo", true)

	_, err := NewResolver(env, mocks.NewMockDirectoryProbe(ctrl)).Resolve()
	if !errors.Is(err, ErrMonorepoNotAbsolute) {
		t.Fatalf("Resolve() error = %v, want ErrMonorepoNotAbsolute", err)
	}
}

func TestResolveRejectsMissingRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	probe := mocks.NewMockDirectoryProbe(ctrl)

	env.EXPECT().Lookup("MONOREPO").Return("/nope", true)
	probe.EXPECT().IsDirectory("/nope").Return(false)

	_, err := NewResolver(env, probe).Resolve()
	if !errors.Is(err, ErrMonorepoNotDirectory) {
		t.Fatalf("Resolve() error = %v, want ErrMonorepoNotDirectory", err)
	}
}

func TestResolveRejectsMissingSourceTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	env := mocks.NewMockEnvironment(ctrl)
	probe := mocks.NewMockDirectoryProbe(ctrl)

	env.EXPECT().Lookup("MONOREPO").Return("/repo", true)
	probe.EXPECT().IsDirectory("/repo").Return(true)
	probe.EXPECT().IsDirectory("/repo/src").Return(false)

	_, err := NewResolver(env, probe).Resolve()
	if !errors.Is(err, ErrSourceRootMissing) {
		t.Fatalf("Resolve() error = %v, want ErrSourceRootMissing", err)
	}
}
