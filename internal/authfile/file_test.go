package authfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicCreatesFileWithOwnerOnlyPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	raw := []byte(sampleAuth)

	if err := WriteAtomic(path, raw); err != nil {
		t.Fatalf("WriteAtomic() error = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(raw) {
		t.Errorf("content = %q, want %q", got, raw)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestWriteAtomicReplacesExistingFileAndTightensPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0o644); err != nil {
		t.Fatalf("setup WriteFile() error = %v", err)
	}

	if err := WriteAtomic(path, []byte(sampleAuth)); err != nil {
		t.Fatalf("WriteAtomic() error = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != sampleAuth {
		t.Errorf("content was not replaced, got %q", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestWriteAtomicCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "grok", "auth.json")

	if err := WriteAtomic(path, []byte(sampleAuth)); err != nil {
		t.Fatalf("WriteAtomic() error = %v, want nil", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file was not created: %v", err)
	}
}

func TestWriteAtomicLeavesNoTemporaryFilesBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")

	if err := WriteAtomic(path, []byte(sampleAuth)); err != nil {
		t.Fatalf("WriteAtomic() error = %v, want nil", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "auth.json" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("dir entries = %v, want only auth.json", names)
	}
}

func TestReadReturnsRawBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(sampleAuth), 0o600); err != nil {
		t.Fatalf("setup WriteFile() error = %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if string(got) != sampleAuth {
		t.Errorf("Read() = %q, want file contents", got)
	}
}

func TestReadErrorsWhenFileIsMissing(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "auth.json"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read() error = %v, want fs.ErrNotExist", err)
	}
}
