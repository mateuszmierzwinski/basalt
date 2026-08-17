package bazelquery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mateuszmierzwinski/basalt/internal/port"
	"github.com/mateuszmierzwinski/basalt/internal/port/mocks"
	"go.uber.org/mock/gomock"
)

func TestTargetsRunsTheExpectedCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := mocks.NewMockCommandRunner(ctrl)
	runner.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, spec port.CommandSpec) (port.CommandResult, error) {
			if spec.Path != "/usr/local/bin/bazel" {
				t.Errorf("Path = %q", spec.Path)
			}
			if spec.Dir != "/repo" {
				t.Errorf("Dir = %q", spec.Dir)
			}
			want := []string{"query", "kind(rule, //src/...:*)", "--output=label_kind"}
			if len(spec.Args) != len(want) {
				t.Fatalf("Args = %v, want %v", spec.Args, want)
			}
			for i := range want {
				if spec.Args[i] != want[i] {
					t.Errorf("Args[%d] = %q, want %q", i, spec.Args[i], want[i])
				}
			}
			return port.CommandResult{Stdout: realQueryOutput}, nil
		})

	got, err := NewClient(runner, "/usr/local/bin/bazel", "/repo").Targets(context.Background())
	if err != nil {
		t.Fatalf("Targets() error = %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("Targets() returned %d, want 9", len(got))
	}
}

func TestTargetsReportsQueryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := mocks.NewMockCommandRunner(ctrl)
	runner.EXPECT().Run(gomock.Any(), gomock.Any()).Return(
		port.CommandResult{Stderr: "INFO: loading\nERROR: no such package 'src/nope'", ExitCode: 7},
		errors.New("exit status 7"),
	)

	_, err := NewClient(runner, "/usr/local/bin/bazel", "/repo").Targets(context.Background())
	if !errors.Is(err, ErrQueryFailed) {
		t.Fatalf("Targets() error = %v, want ErrQueryFailed", err)
	}
	if got := err.Error(); !strings.Contains(got, "no such package") || !strings.Contains(got, "exit 7") {
		t.Fatalf("error %q should carry bazel's own message and exit code", got)
	}
}

func TestTargetsReportsEmptyOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := mocks.NewMockCommandRunner(ctrl)
	runner.EXPECT().Run(gomock.Any(), gomock.Any()).Return(port.CommandResult{Stdout: ""}, nil)

	if _, err := NewClient(runner, "/bazel", "/repo").Targets(context.Background()); !errors.Is(err, ErrNoTargets) {
		t.Fatalf("Targets() error = %v, want ErrNoTargets", err)
	}
}

func TestTargetsPropagatesCallerCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := mocks.NewMockCommandRunner(ctrl)
	runner.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ port.CommandSpec) (port.CommandResult, error) {
			<-ctx.Done()
			return port.CommandResult{ExitCode: -1}, ctx.Err()
		})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewClient(runner, "/bazel", "/repo").Targets(ctx); err == nil {
		t.Fatal("Targets() error = nil, want cancellation")
	}
}
