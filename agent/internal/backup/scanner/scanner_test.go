package scanner_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smcsoluciones/backup-system/agent/internal/backup/scanner"
)

func makeTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"file_a.txt":        "content of file A",
		"file_b.log":        "content of file B",
		"subdir/file_c.txt": "content of file C",
		"subdir/file_d.tmp": "temp file",
	}
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(full), 0o755)
		_ = os.WriteFile(full, []byte(content), 0o644)
	}
	return dir
}

func TestFullScanDetectsAllFiles(t *testing.T) {
	dir := makeTestDir(t)
	cache := scanner.NewCache()
	opts := scanner.Options{Incremental: false}
	sc := scanner.New(dir, cache, opts)

	result, _, err := sc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	// 4 files (not counting dirs)
	if len(result.Changed) != 4 {
		t.Fatalf("expected 4 changed files, got %d", len(result.Changed))
	}
}

func TestIncrementalNoChangeReturnsZero(t *testing.T) {
	dir := makeTestDir(t)
	cache := scanner.NewCache()
	opts := scanner.Options{Incremental: false}

	// Full scan → populate cache
	_, newCache, err := scanner.New(dir, cache, opts).Scan()
	if err != nil {
		t.Fatal(err)
	}

	// Incremental scan with same cache → nothing changed
	opts.Incremental = true
	result, _, err := scanner.New(dir, newCache, opts).Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Changed) != 0 {
		t.Fatalf("expected 0 changed files, got %d", len(result.Changed))
	}
}

func TestIncrementalDetectsModifiedFile(t *testing.T) {
	dir := makeTestDir(t)

	// Full scan
	_, newCache, _ := scanner.New(dir, scanner.NewCache(), scanner.Options{}).Scan()

	// Modify one file (change mtime by writing)
	time.Sleep(10 * time.Millisecond)
	target := filepath.Join(dir, "file_a.txt")
	_ = os.WriteFile(target, []byte("modified content"), 0o644)

	// Incremental scan
	result, _, err := scanner.New(dir, newCache, scanner.Options{Incremental: true}).Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Changed) != 1 {
		t.Fatalf("expected 1 changed file, got %d: %+v", len(result.Changed), result.Changed)
	}
	if result.Changed[0].Path != "file_a.txt" {
		t.Fatalf("expected file_a.txt changed, got %q", result.Changed[0].Path)
	}
}

func TestExcludePatternsSkipFiles(t *testing.T) {
	dir := makeTestDir(t)
	opts := scanner.Options{
		Incremental:     false,
		ExcludePatterns: []string{"*.tmp", "*.log"},
	}
	result, _, err := scanner.New(dir, scanner.NewCache(), opts).Scan()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range result.Changed {
		if filepath.Ext(f.Path) == ".tmp" || filepath.Ext(f.Path) == ".log" {
			t.Fatalf("excluded file %q appeared in results", f.Path)
		}
	}

	// Only .txt files should remain (2)
	if len(result.Changed) != 2 {
		t.Fatalf("expected 2 files after exclusions, got %d", len(result.Changed))
	}
}

func TestDetectsDeletion(t *testing.T) {
	dir := makeTestDir(t)

	// Full scan
	_, newCache, _ := scanner.New(dir, scanner.NewCache(), scanner.Options{}).Scan()

	// Delete a file
	_ = os.Remove(filepath.Join(dir, "file_b.log"))

	// Incremental scan
	result, _, err := scanner.New(dir, newCache, scanner.Options{Incremental: true}).Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Deleted) != 1 {
		t.Fatalf("expected 1 deleted file, got %d", len(result.Deleted))
	}
	if result.Deleted[0].Path != "file_b.log" {
		t.Fatalf("expected file_b.log deleted, got %q", result.Deleted[0].Path)
	}
}

func TestCacheSaveLoad(t *testing.T) {
	dir := makeTestDir(t)
	_, newCache, _ := scanner.New(dir, scanner.NewCache(), scanner.Options{}).Scan()

	cacheFile := filepath.Join(t.TempDir(), "cache.json")
	if err := newCache.Save(cacheFile); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := scanner.LoadCache(cacheFile)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}

	if len(loaded.Entries) != len(newCache.Entries) {
		t.Fatalf("cache entry count mismatch: got %d, want %d",
			len(loaded.Entries), len(newCache.Entries))
	}
}

func TestLoadCacheMissingFileReturnsEmpty(t *testing.T) {
	c, err := scanner.LoadCache("/nonexistent/path/cache.json")
	if err != nil {
		t.Fatalf("expected no error for missing cache, got: %v", err)
	}
	if len(c.Entries) != 0 {
		t.Fatalf("expected empty cache, got %d entries", len(c.Entries))
	}
}
