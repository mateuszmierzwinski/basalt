package osfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

type store struct{}

// NewStore builds the real file store.
func NewStore() port.FileStore {
	return &store{}
}

// Read returns a file's contents.
func (s *store) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Write replaces a file atomically: the content is written to a temporary file
// in the same directory and renamed over the target, so a crash mid-write
// leaves the previous cache intact rather than a truncated one.
func (s *store) Write(path string, data []byte, perm fs.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if err := writeAndClose(temporary, data, perm); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}

func writeAndClose(file *os.File, data []byte, perm fs.FileMode) error {
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Chmod(perm); err != nil {
		return err
	}
	return file.Sync()
}

// MkdirAll creates a directory and any missing parents.
func (s *store) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// ListNames returns the entry names of a directory, sorted, as os.ReadDir
// guarantees. A missing directory yields an error; an empty one yields no
// names.
func (s *store) ListNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}
