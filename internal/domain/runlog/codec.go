package runlog

import (
	"encoding/json"
	"fmt"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Encode serializes the history for storage, indented to stay readable — it
// lives beside the target cache and is meant to be openable.
func Encode(history History) ([]byte, error) {
	return json.MarshalIndent(history, "", "  ")
}

// Decode parses and validates a stored history.
//
// Like the target cache, this is a plain file in the repository and is treated
// as untrusted input. Every entry's label must parse, and every log path must
// resolve inside the cache directory — that file is handed to a pager, so a
// forged path would be an arbitrary-file-read.
func Decode(data []byte, cacheDir string) (History, error) {
	var history History
	if err := json.Unmarshal(data, &history); err != nil {
		return History{}, fmt.Errorf("%w: %v", ErrCorruptHistory, err)
	}
	if err := validateProvenance(history); err != nil {
		return History{}, err
	}
	if err := validateEntries(history, cacheDir); err != nil {
		return History{}, err
	}
	return history, nil
}

func validateProvenance(history History) error {
	if history.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: found %d, want %d", ErrSchemaMismatch, history.SchemaVersion, SchemaVersion)
	}
	return nil
}

func validateEntries(history History, cacheDir string) error {
	for label, entry := range history.Entries {
		if err := validateEntry(label, entry, cacheDir); err != nil {
			return err
		}
	}
	return nil
}

func validateEntry(label string, entry Entry, cacheDir string) error {
	if entry.Label != label {
		return fmt.Errorf("%w: entry %q is filed under %q", ErrPoisonedEntry, entry.Label, label)
	}
	if _, _, err := target.SplitLabel(label); err != nil {
		return fmt.Errorf("%w: %v", ErrPoisonedEntry, err)
	}
	if _, ok := ResolveLog(cacheDir, entry.LogFile); entry.LogFile != "" && !ok {
		return fmt.Errorf("%w: %q", ErrLogEscapesDir, entry.LogFile)
	}
	return nil
}
