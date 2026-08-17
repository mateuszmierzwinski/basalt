// Package runlog records how each target's last bazel command went, and where
// its output was kept.
//
// It is the domain half: the models, their validation and the rules for where a
// log may live. Reading and writing the files is the history service's job.
package runlog

import "time"

// Status is the outcome of a target's most recent bazel command.
type Status int

const (
	// StatusUnknown means the target has not been run since the history was
	// last cleared — most targets, most of the time.
	StatusUnknown Status = iota
	StatusPassed
	StatusFailed
)

// StatusOf turns a command error into a status.
func StatusOf(runErr error) Status {
	if runErr != nil {
		return StatusFailed
	}
	return StatusPassed
}

// Entry is one target's last run.
//
// LogFile is the log's name, not its path: it is stored rather than re-derived
// so that a history written by a later version, which might name logs
// differently, still points at the right file, and it is resolved against the
// running cache directory on load — because the file it names is opened in a
// pager, and because the same checkout sits at a different path on each machine
// that syncs it.
type Entry struct {
	Label    string    `json:"label"`
	Action   string    `json:"action"`
	Status   Status    `json:"status"`
	RanAt    time.Time `json:"ran_at"`
	LogFile  string    `json:"log_file"`
	ExitNote string    `json:"exit_note"`
}

// History is the persisted record, keyed by target label.
//
// Like the target cache it holds no absolute path: entries are keyed by label
// and LogFile is a bare file name resolved against the running cache directory,
// so a checkout synced to a second machine keeps its history rather than
// starting blank.
type History struct {
	SchemaVersion int              `json:"schema_version"`
	Entries       map[string]Entry `json:"entries"`
}

// Statuses projects the history into the lookup the target list renders from.
func (h History) Statuses() map[string]Status {
	statuses := make(map[string]Status, len(h.Entries))
	for label, entry := range h.Entries {
		statuses[label] = entry.Status
	}
	return statuses
}
