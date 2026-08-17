package osfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDirectoryAcceptsADirectory(t *testing.T) {
	if !NewProbe().IsDirectory(t.TempDir()) {
		t.Fatal("IsDirectory(tempdir) = false, want true")
	}
}

func TestIsDirectoryRejectsARegularFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if NewProbe().IsDirectory(file) {
		t.Fatal("IsDirectory(file) = true, want false")
	}
}

func TestIsDirectoryRejectsAMissingPath(t *testing.T) {
	if NewProbe().IsDirectory(filepath.Join(t.TempDir(), "absent")) {
		t.Fatal("IsDirectory(absent) = true, want false")
	}
}

func TestIsDirectoryRejectsABrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "absent"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if NewProbe().IsDirectory(link) {
		t.Fatal("IsDirectory(broken symlink) = true, want false")
	}
}
