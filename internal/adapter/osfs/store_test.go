package osfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteThenRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	store := NewStore()
	if err := store.Write(path, []byte("payload"), 0o600); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	got, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("Read() = %q", got)
	}
}

func TestWriteAppliesPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	if err := NewStore().Write(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestWriteReplacesExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	store := NewStore()
	if err := store.Write(path, []byte("old and much longer"), 0o600); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := store.Write(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	got, _ := store.Read(path)
	if string(got) != "new" {
		t.Fatalf("Read() = %q — a shorter rewrite left a tail behind", got)
	}
}

func TestWriteLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	if err := NewStore().Write(filepath.Join(dir, "cache.json"), []byte("x"), 0o600); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	names, err := NewStore().ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames() error = %v", err)
	}
	if len(names) != 1 || names[0] != "cache.json" {
		t.Fatalf("ListNames() = %v, want only cache.json", names)
	}
}

func TestWriteFailsWhenDirectoryMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent", "cache.json")
	if err := NewStore().Write(path, []byte("x"), 0o600); err == nil {
		t.Fatal("Write() error = nil, want a failure")
	}
}

func TestReadReportsMissingFile(t *testing.T) {
	if _, err := NewStore().Read(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("Read() error = nil, want a failure")
	}
}

func TestMkdirAllIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "c")
	store := NewStore()
	if err := store.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := store.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("second MkdirAll() error = %v", err)
	}
	if !NewProbe().IsDirectory(path) {
		t.Fatal("MkdirAll() did not create the directory")
	}
}

func TestListNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"README.md", "BUILD", "main.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	names, err := NewStore().ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames() error = %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("ListNames() = %v, want 3 entries", names)
	}
}

func TestListNamesOnEmptyDirectory(t *testing.T) {
	names, err := NewStore().ListNames(t.TempDir())
	if err != nil {
		t.Fatalf("ListNames() error = %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("ListNames() = %v, want empty", names)
	}
}

func TestListNamesReportsMissingDirectory(t *testing.T) {
	if _, err := NewStore().ListNames(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("ListNames() error = nil, want a failure")
	}
}
