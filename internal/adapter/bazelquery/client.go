// Package bazelquery enumerates the monorepo's Bazel targets by shelling out
// to `bazel query`.
//
// This is the authority on what targets exist — the alternative, walking the
// tree and parsing BUILD files by hand, cannot see macros or aliases
// (CLAUDE.md, Resolved design decisions).
package bazelquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

const (
	// queryExpression is scoped to //src/... because that is the tree the
	// operator navigates; //tools and external repos are noise here.
	queryExpression = "kind(rule, //src/...:*)"

	// DefaultTimeout is generous because a cold bazel server takes far longer
	// than the sub-second warm case.
	DefaultTimeout = 3 * time.Minute
)

type client struct {
	runner    port.CommandRunner
	bazelPath string
	root      string
	timeout   time.Duration
}

// NewClient builds the query client. bazelPath is the resolved executable
// found at startup, not the bare name, so a $PATH change cannot redirect it.
func NewClient(runner port.CommandRunner, bazelPath, root string) port.TargetSource {
	return &client{runner: runner, bazelPath: bazelPath, root: root, timeout: DefaultTimeout}
}

// Targets runs the query and parses its output. The command is built as argv,
// so neither the root path nor the query expression can be re-interpreted by a
// shell — there is no shell.
func (c *client) Targets(ctx context.Context) ([]target.Target, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	result, err := c.runner.Run(ctx, c.spec())
	if err != nil {
		return nil, c.queryError(ctx, result, err)
	}
	return ParseQueryOutput(result.Stdout)
}

func (c *client) spec() port.CommandSpec {
	return port.CommandSpec{
		Path: c.bazelPath,
		Args: []string{"query", queryExpression, "--output=label_kind"},
		Dir:  c.root,
	}
}

func (c *client) queryError(ctx context.Context, result port.CommandResult, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w after %s", ErrQueryTimeout, c.timeout)
	}
	return fmt.Errorf("%w (exit %d): %s", ErrQueryFailed, result.ExitCode, lastLine(result.Stderr))
}
