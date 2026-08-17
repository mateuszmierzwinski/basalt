package fswatch

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// buildFileNames are the files whose contents define targets. A change to any
// other file cannot change the target list, so it is ignored.
var buildFileNames = map[string]bool{
	"BUILD":       true,
	"BUILD.bazel": true,
}

// skippedDirNames are never watched: they churn constantly and contain no
// hand-maintained BUILD files.
var skippedDirNames = map[string]bool{
	".git":         true,
	".scannerwork": true,
	"node_modules": true,
}

// IsRelevant reports whether a filesystem event could change the target list.
// Only BUILD files matter for content changes; directory creation and removal
// matter because they add or drop packages wholesale.
func IsRelevant(path string, op fsnotify.Op) bool {
	if buildFileNames[filepath.Base(path)] {
		return true
	}
	return op.Has(fsnotify.Create) || op.Has(fsnotify.Remove) || op.Has(fsnotify.Rename)
}

// IsSkippedDir reports whether a directory should be left unwatched.
func IsSkippedDir(name string) bool {
	return skippedDirNames[name] || strings.HasPrefix(name, ".bazel")
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// addRecursive watches a directory and every descendant. Individual failures
// are tolerated — a directory removed between the walk and the watch is normal
// in a tree being synced — because a partially watched tree is far better than
// no watch at all.
func addRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if IsSkippedDir(entry.Name()) {
			return filepath.SkipDir
		}
		w.Add(path)
		return nil
	})
}
