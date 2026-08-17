// Package history persists which targets passed or failed, and where their
// output was kept.
package history

import (
	"io/fs"
	"sync"

	"github.com/mateuszmierzwinski/basalt/internal/domain/runlog"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

const (
	historyDirPerm  fs.FileMode = 0o755
	historyFilePerm fs.FileMode = 0o644
	logFilePerm     fs.FileMode = 0o644
)

// Config is the store's construction parameters.
type Config struct {
	Store    port.FileStore
	Probe    port.FileProbe
	CacheDir string
}

// store holds the history in memory and writes it through on every change.
//
// The mutex guards entries, which the session mutates while the renderer reads
// the projection — the whole point of keeping this behind an interface.
type store struct {
	files    port.FileStore
	probe    port.FileProbe
	cacheDir string

	mu      sync.RWMutex
	entries map[string]runlog.Entry
}

// New builds the recorder and loads whatever history is on disk.
//
// A history that cannot be read or does not validate is reported and then
// discarded: it degrades to "nothing has been run yet", which is the same state
// as a fresh checkout and never worth refusing to start over.
func New(config Config) (Recorder, error) {
	s := &store{
		files:    config.Store,
		probe:    config.Probe,
		cacheDir: config.CacheDir,
		entries:  map[string]runlog.Entry{},
	}
	return s, s.load()
}

func (s *store) load() error {
	data, err := s.files.Read(runlog.FilePath(s.cacheDir))
	if err != nil {
		return nil // No history yet is the ordinary case, not a failure.
	}
	loaded, err := runlog.Decode(data, s.cacheDir)
	if err != nil {
		return err
	}
	s.entries = loaded.Entries
	if s.entries == nil {
		s.entries = map[string]runlog.Entry{}
	}
	return nil
}

// Record stores an outcome and writes the history through immediately, so a
// basalt killed between runs does not lose what it already knew.
func (s *store) Record(entry runlog.Entry) error {
	entry.LogFile = runlog.LogNameOf(entry.LogFile)
	s.mu.Lock()
	s.entries[entry.Label] = entry
	snapshot := s.snapshotLocked()
	s.mu.Unlock()
	return s.persist(snapshot)
}

func (s *store) snapshotLocked() runlog.History {
	copied := make(map[string]runlog.Entry, len(s.entries))
	for label, entry := range s.entries {
		copied[label] = entry
	}
	return runlog.History{SchemaVersion: runlog.SchemaVersion, Entries: copied}
}

func (s *store) persist(snapshot runlog.History) error {
	data, err := runlog.Encode(snapshot)
	if err != nil {
		return err
	}
	if err := s.files.MkdirAll(s.cacheDir, historyDirPerm); err != nil {
		return err
	}
	return s.files.Write(runlog.FilePath(s.cacheDir), data, historyFilePerm)
}

// Statuses projects the history for the target list.
func (s *store) Statuses() map[string]runlog.Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked().Statuses()
}

// LogPathFor returns a target's recorded log, if it was recorded and is still
// on disk and still inside the cache directory.
//
// The containment check is repeated here rather than trusted from load time: it
// is the guard between a JSON file anyone can edit and a path handed to a pager,
// and it costs nothing to re-assert at the point of use.
func (s *store) LogPathFor(label string) (string, bool) {
	s.mu.RLock()
	entry, known := s.entries[label]
	s.mu.RUnlock()

	if !known || entry.LogFile == "" {
		return "", false
	}
	path, inside := runlog.ResolveLog(s.cacheDir, entry.LogFile)
	if !inside || !s.probe.IsFile(path) {
		return "", false
	}
	return path, true
}

// PrepareLogPath is where the next run of a target writes its output, with the
// directory created so the runner can open the file.
//
// A failure to create it is not reported: the runner falls back to running
// unlogged, which is strictly better than refusing to run the command at all.
func (s *store) PrepareLogPath(label string) string {
	s.files.MkdirAll(runlog.LogsDirectory(s.cacheDir), historyDirPerm)
	return runlog.LogPath(s.cacheDir, label)
}
