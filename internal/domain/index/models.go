package index

import (
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// Index is the persisted target cache. It is JSON for readability — the brief
// asks for a cache a human can open and check.
//
// SchemaVersion is checked on load so a format change invalidates the cache
// rather than being misread as valid.
//
// Nothing here is an absolute path. Every entry is a Bazel label, which is
// workspace-relative by construction, so the file is portable between checkouts
// — the same tree synced to two machines under different $MONOREPO paths shares
// one cache instead of invalidating it on every switch. Each label is still
// re-resolved against the running root on load, so portability costs nothing in
// validation.
type Index struct {
	SchemaVersion int             `json:"schema_version"`
	GeneratedAt   time.Time       `json:"generated_at"`
	Targets       []target.Target `json:"targets"`
}
