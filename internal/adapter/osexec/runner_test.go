package osexec

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

func lookup(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s unavailable: %v", name, err)
	}
	return path
}

func TestRunCapturesStdout(t *testing.T) {
	spec := port.CommandSpec{Path: lookup(t, "echo"), Args: []string{"hello"}}
	got, err := NewRunner().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(got.Stdout) != "hello" {
		t.Fatalf("Stdout = %q", got.Stdout)
	}
	if got.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", got.ExitCode)
	}
}

func TestRunReportsNonZeroExit(t *testing.T) {
	spec := port.CommandSpec{Path: lookup(t, "false")}
	got, err := NewRunner().Run(context.Background(), spec)
	if err == nil {
		t.Fatal("Run() error = nil, want a failure")
	}
	if got.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", got.ExitCode)
	}
}

func TestRunReportsUnstartableCommand(t *testing.T) {
	spec := port.CommandSpec{Path: "/nonexistent/basalt-test-binary"}
	got, err := NewRunner().Run(context.Background(), spec)
	if err == nil {
		t.Fatal("Run() error = nil, want a failure")
	}
	if got.ExitCode != -1 {
		t.Fatalf("ExitCode = %d, want -1 for an unstartable command", got.ExitCode)
	}
}

// The whole point of argv execution: metacharacters are inert data.
func TestRunDoesNotInterpretShellMetacharacters(t *testing.T) {
	payload := "; touch /tmp/basalt-pwned; echo $(whoami) `id` && rm -rf /"
	spec := port.CommandSpec{Path: lookup(t, "echo"), Args: []string{payload}}
	got, err := NewRunner().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(got.Stdout) != payload {
		t.Fatalf("Stdout = %q, want the payload echoed verbatim", got.Stdout)
	}
}

func TestRunHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	spec := port.CommandSpec{Path: lookup(t, "sleep"), Args: []string{"30"}}

	start := time.Now()
	if _, err := NewRunner().Run(ctx, spec); err == nil {
		t.Fatal("Run() error = nil, want cancellation")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Run() took %v — context was not honoured", elapsed)
	}
}

func TestRunUsesTheGivenDirectory(t *testing.T) {
	dir := t.TempDir()
	spec := port.CommandSpec{Path: lookup(t, "pwd"), Dir: dir}
	got, err := NewRunner().Run(context.Background(), spec)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(got.Stdout, strings.TrimPrefix(dir, "/private")) {
		t.Fatalf("Stdout = %q, want it to contain %q", got.Stdout, dir)
	}
}
