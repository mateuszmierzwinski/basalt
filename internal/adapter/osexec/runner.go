// Package osexec adapts os/exec to basalt's process ports.
package osexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

type runner struct{}

// NewRunner builds the capturing command runner.
func NewRunner() port.CommandRunner {
	return &runner{}
}

// Run executes a command and captures its output. exec.CommandContext takes
// argv directly — no shell is involved at any point, so nothing in a target
// name or path can be interpreted as a command.
func (r *runner) Run(ctx context.Context, spec port.CommandSpec) (port.CommandResult, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return port.CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCodeOf(err),
	}, err
}

// exitCodeOf extracts a process exit status, distinguishing "the command ran
// and failed" from "the command could not be started at all".
func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

type interactiveRunner struct{}

// NewInteractiveRunner builds the runner that hands the terminal to a child.
func NewInteractiveRunner() port.InteractiveRunner {
	return &interactiveRunner{}
}

// RunInteractive runs a child process attached to basalt's own stdio. The
// caller is responsible for having already restored the terminal out of raw
// mode — nano and claude are unusable otherwise.
//
// When the spec names a log, the child's output is copied into it as well as
// shown. That makes stdout a pipe rather than a terminal, so bazel drops its
// curses progress display in favour of plain lines — which is the better trade:
// the operator still sees everything as it happens, and the saved log is
// readable afterwards instead of being a screenful of cursor movements.
func (r *interactiveRunner) RunInteractive(ctx context.Context, spec port.CommandSpec) error {
	if spec.LogPath == "" {
		return r.run(ctx, spec, os.Stdout, os.Stderr)
	}
	return r.runLogged(ctx, spec)
}

// runLogged captures the child's output to a file while mirroring it to the
// terminal. A log that cannot be opened is not fatal: the command still runs
// and is still watched, it just leaves nothing to review afterwards.
func (r *interactiveRunner) runLogged(ctx context.Context, spec port.CommandSpec) error {
	log, err := os.OpenFile(spec.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, logFilePerm)
	if err != nil {
		return r.run(ctx, spec, os.Stdout, os.Stderr)
	}
	defer log.Close()
	return r.run(ctx, spec,
		io.MultiWriter(os.Stdout, log),
		io.MultiWriter(os.Stderr, log),
	)
}

func (r *interactiveRunner) run(ctx context.Context, spec port.CommandSpec, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// logFilePerm keeps captured output readable by its owner and nobody else's to
// write — build output can carry paths and environment detail.
const logFilePerm = 0o644
