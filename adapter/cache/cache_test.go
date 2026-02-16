package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store := New()
	path := filepath.Join(dir, "test.txt")

	data := []byte("hello world")
	if err := store.AtomicWrite(path, data); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestAtomicWriteCreatesDir(t *testing.T) {
	dir := t.TempDir()
	store := New()
	path := filepath.Join(dir, "sub", "dir", "test.txt")

	if err := store.AtomicWrite(path, []byte("data")); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("got %q, want %q", got, "data")
	}
}

func TestReadIfFresh(t *testing.T) {
	dir := t.TempDir()
	store := New()
	path := filepath.Join(dir, "fresh.txt")

	// Non-existent file
	_, fresh := store.ReadIfFresh(path, time.Minute)
	if fresh {
		t.Error("non-existent file should not be fresh")
	}

	// Fresh file
	os.WriteFile(path, []byte("data"), 0o644)
	data, fresh := store.ReadIfFresh(path, time.Minute)
	if !fresh {
		t.Error("just-written file should be fresh")
	}
	if string(data) != "data" {
		t.Errorf("got %q, want %q", data, "data")
	}

	// Expired file (set mtime to past)
	past := time.Now().Add(-2 * time.Minute)
	os.Chtimes(path, past, past)
	_, fresh = store.ReadIfFresh(path, time.Minute)
	if fresh {
		t.Error("old file should not be fresh")
	}
}

func TestFileMTime(t *testing.T) {
	dir := t.TempDir()
	store := New()
	path := filepath.Join(dir, "mtime.txt")

	os.WriteFile(path, []byte("data"), 0o644)

	mtime, err := store.FileMTime(path)
	if err != nil {
		t.Fatalf("FileMTime: %v", err)
	}

	if time.Since(mtime) > time.Second {
		t.Error("mtime should be recent")
	}
}

func TestCleanOld(t *testing.T) {
	dir := t.TempDir()
	store := New()

	// Create test files
	os.WriteFile(filepath.Join(dir, "data_2024-01-01.txt"), []byte("old"), 0o644)
	os.WriteFile(filepath.Join(dir, "data_2024-01-02.txt"), []byte("keep"), 0o644)
	os.WriteFile(filepath.Join(dir, "data_2024-01-03.txt"), []byte("old"), 0o644)

	err := store.CleanOld(dir, "data_*.txt", "data_2024-01-02.txt")
	if err != nil {
		t.Fatalf("CleanOld: %v", err)
	}

	// Only the "keep" file should remain
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file, got %d", len(entries))
	}
	if entries[0].Name() != "data_2024-01-02.txt" {
		t.Errorf("wrong file kept: %s", entries[0].Name())
	}
}

func TestReadWriteFile(t *testing.T) {
	dir := t.TempDir()
	store := New()
	path := filepath.Join(dir, "rw.txt")

	if err := store.WriteFile(path, []byte("hello")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := store.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", data, "hello")
	}
}
