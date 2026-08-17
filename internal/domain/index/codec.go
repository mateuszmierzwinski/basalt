package index

import (
	"encoding/json"
	"fmt"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// reject wraps a sentinel with the underlying cause. Every cache rejection in
// this package reports the same way, so the shape is named once.
func reject(sentinel error, cause error) error {
	return fmt.Errorf("%w: %v", sentinel, cause)
}

// Encode serializes an index for storage. It is indented because the brief
// specifies JSON "for readability".
func Encode(idx Index) ([]byte, error) {
	return json.MarshalIndent(idx, "", "  ")
}

// Decode parses and validates a stored index. The cache is a plain file in the
// repository root, so its contents are treated as untrusted input: a caller
// gets either a fully-validated index or an error telling it to re-index.
func Decode(data []byte, root string) (Index, error) {
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, reject(ErrCorruptCache, err)
	}
	if err := validateProvenance(idx); err != nil {
		return Index{}, err
	}
	if err := validateTargets(idx, root); err != nil {
		return Index{}, err
	}
	return idx, nil
}

func validateProvenance(idx Index) error {
	if idx.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: found %d, want %d", ErrSchemaMismatch, idx.SchemaVersion, SchemaVersion)
	}
	return nil
}

// validateTargets re-derives every entry from its label, so a hand-edited cache
// cannot smuggle in a display string, a package or a path that the parser would
// never have produced.
func validateTargets(idx Index, root string) error {
	for _, entry := range idx.Targets {
		rebuilt, err := target.New(entry.Kind, entry.Label)
		if err != nil {
			return reject(ErrPoisonedEntry, err)
		}
		if rebuilt != entry {
			return fmt.Errorf("%w: %q does not match its label", ErrPoisonedEntry, entry.Display)
		}
		if _, err := target.Directory(root, entry); err != nil {
			return reject(ErrPoisonedEntry, err)
		}
	}
	return nil
}
