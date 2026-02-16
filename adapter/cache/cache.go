package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store implements the ports.CacheStore interface using the filesystem.
type Store struct{}

// New creates a new file-based cache store.
func New() *Store {
	return &Store{}
}

// AtomicWrite writes data to path atomically via a temp file + rename.
func (s *Store) AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, path)
}

// ReadIfFresh reads a file and returns its contents if it was modified within ttl.
func (s *Store) ReadIfFresh(path string, ttl time.Duration) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}

	if time.Since(info.ModTime()) > ttl {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	return data, true
}

// ReadFile reads the entire contents of a file.
func (s *Store) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file, creating it if necessary.
func (s *Store) WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// FileMTime returns the modification time of a file.
func (s *Store) FileMTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// CleanOld removes files matching pattern in dir, except those matching keep.
func (s *Store) CleanOld(dir, pattern, keep string) error {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	for _, m := range matches {
		if filepath.Base(m) == keep {
			continue
		}
		os.Remove(m)
	}

	return nil
}
