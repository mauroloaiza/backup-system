package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Destination writes backup objects to a local filesystem directory.
// Writes are atomic: data goes to a .tmp file then renamed on Close.
type Destination struct {
	basePath string
}

// New creates a local Destination rooted at basePath.
// The directory is created if it doesn't exist.
func New(basePath string) (*Destination, error) {
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("local: create base dir %q: %w", basePath, err)
	}
	return &Destination{basePath: basePath}, nil
}

// atomicWriter writes to a .tmp file and renames to the final path on Close.
type atomicWriter struct {
	tmp  *os.File
	dest string
}

func (a *atomicWriter) Write(p []byte) (int, error) { return a.tmp.Write(p) }

func (a *atomicWriter) Close() error {
	if err := a.tmp.Close(); err != nil {
		return fmt.Errorf("local: close tmp: %w", err)
	}
	if err := os.Rename(a.tmp.Name(), a.dest); err != nil {
		_ = os.Remove(a.tmp.Name())
		return fmt.Errorf("local: rename to %q: %w", a.dest, err)
	}
	return nil
}

func (d *Destination) Write(name string) (io.WriteCloser, error) {
	full := filepath.Join(d.basePath, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return nil, fmt.Errorf("local: mkdir for %q: %w", name, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(full), ".tmp-backup-*")
	if err != nil {
		return nil, fmt.Errorf("local: create tmp: %w", err)
	}
	return &atomicWriter{tmp: tmp, dest: full}, nil
}

func (d *Destination) Read(name string) (io.ReadCloser, error) {
	full := filepath.Join(d.basePath, filepath.FromSlash(name))
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("local: open %q: %w", name, err)
	}
	return f, nil
}

func (d *Destination) Delete(name string) error {
	full := filepath.Join(d.basePath, filepath.FromSlash(name))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local: delete %q: %w", name, err)
	}
	return nil
}

func (d *Destination) List(prefix string) ([]string, error) {
	root := filepath.Join(d.basePath, filepath.FromSlash(prefix))
	var names []string
	err := filepath.WalkDir(root, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(d.basePath, path)
		names = append(names, strings.ReplaceAll(rel, string(os.PathSeparator), "/"))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("local: list %q: %w", prefix, err)
	}
	return names, nil
}

func (d *Destination) Close() error { return nil }
