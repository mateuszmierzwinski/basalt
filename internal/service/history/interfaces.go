package history

import "github.com/mateuszmierzwinski/basalt/internal/domain/runlog"

// Recorder keeps and reports how each target's last bazel command went.
//
// Nothing here returns an error for a read: a history that could not be loaded
// is reported once at startup and then behaves exactly like an empty one, since
// "this target has not been run" is a perfectly good answer.
type Recorder interface {
	// Record stores an outcome and persists it.
	Record(entry runlog.Entry) error
	// Statuses is the lookup the target list is coloured from.
	Statuses() map[string]runlog.Status
	// LogPathFor returns the recorded log for a target, and whether there is
	// one that still exists.
	LogPathFor(label string) (string, bool)
	// PrepareLogPath is where a target's next run should write its output,
	// with the directory made ready for it.
	PrepareLogPath(label string) string
}
