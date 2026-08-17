// Package port declares the interfaces basalt's domain depends on. Adapters
// implement them; the domain never imports an adapter (CLAUDE.md Rule 1.2).
package port

import (
	"context"
	"io/fs"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/layout"
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Terminal is the controlling tty basalt draws on and reads from.
type Terminal interface {
	// MakeRaw disables echo and line buffering; Restore puts back whatever was
	// there before. Both are idempotent, because a terminal left in raw mode
	// is a denial of service on the operator's shell, not a cosmetic defect.
	MakeRaw() error
	Restore() error
	Size() (width, height int, err error)
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	// Resized fires on SIGWINCH, coalescing bursts into a single wakeup.
	Resized() <-chan struct{}
	Close() error
}

// Screen paints frames onto a terminal.
type Screen interface {
	// Enter switches to the alternate screen buffer, so the operator's
	// scrollback is returned untouched on exit; Leave restores it.
	Enter() error
	Leave() error
	Draw(frame layout.Frame) error
}

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

// Environment abstracts the process environment and executable lookup so the
// domain can be exercised without a real $PATH or real environment variables.
type Environment interface {
	Lookup(key string) (string, bool)
	LookPath(file string) (string, error)
}

// DirectoryProbe reports whether a filesystem path exists and is a directory.
// It exists so workspace validation is testable without touching a real disk.
type DirectoryProbe interface {
	IsDirectory(path string) bool
}

// FileProbe reports whether a path exists and is a regular file. A recorded log
// can be deleted between runs, so its presence is checked before the action to
// open it is offered.
type FileProbe interface {
	IsFile(path string) bool
}

// FileStore is the subset of filesystem access basalt needs to persist its
// target cache.
type FileStore interface {
	Read(path string) ([]byte, error)
	Write(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	ListNames(dir string) ([]string, error)
}

// CommandRunner executes an external program and captures its output.
// Arguments are passed as argv and are never interpolated into a shell string,
// so a hostile target name or path cannot become a command (test/cve).
type CommandRunner interface {
	Run(ctx context.Context, spec CommandSpec) (CommandResult, error)
}

// InteractiveRunner hands the terminal to a child process and waits for it.
// Unlike CommandRunner it captures nothing: stdin, stdout and stderr are the
// caller's own tty, which is what makes nano, claude and bazel usable from
// inside basalt.
type InteractiveRunner interface {
	RunInteractive(ctx context.Context, spec CommandSpec) error
}

// TargetSource enumerates the monorepo's Bazel targets. The production
// implementation shells out to `bazel query`; tests supply a fixture.
type TargetSource interface {
	Targets(ctx context.Context) ([]target.Target, error)
}

// Clock reports the current time. Cache staleness is a policy decision, so it
// has to be testable without sleeping.
type Clock interface {
	Now() time.Time
}

// TreeWatcher signals that something under the watched root changed in a way
// that could invalidate the target index. It coalesces bursts internally: this
// tree is synced by Dropbox and fires storms unrelated to real BUILD edits.
type TreeWatcher interface {
	Changes() <-chan struct{}
	Close() error
}
