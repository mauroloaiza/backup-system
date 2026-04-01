package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo holds metadata used to detect changed files.
type FileInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	SHA256  string    `json:"sha256,omitempty"` // populated on full scan
	IsDir   bool      `json:"is_dir"`
}

// Cache is the incremental state persisted between backup runs.
type Cache struct {
	CreatedAt time.Time            `json:"created_at"`
	Entries   map[string]*FileInfo `json:"entries"` // key = relative path
}

// NewCache creates an empty cache.
func NewCache() *Cache {
	return &Cache{
		CreatedAt: time.Now().UTC(),
		Entries:   make(map[string]*FileInfo),
	}
}

// LoadCache reads a JSON cache file from disk, or returns an empty cache.
func LoadCache(path string) (*Cache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCache(), nil
		}
		return nil, fmt.Errorf("scanner: load cache: %w", err)
	}
	c := &Cache{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("scanner: parse cache: %w", err)
	}
	if c.Entries == nil {
		c.Entries = make(map[string]*FileInfo)
	}
	return c, nil
}

// Save writes the cache to disk.
func (c *Cache) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("scanner: marshal cache: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("scanner: write cache: %w", err)
	}
	return os.Rename(tmp, path)
}

// ScanResult holds changed and deleted files from a scan.
type ScanResult struct {
	Changed []FileInfo // new or modified files
	Deleted []FileInfo // files present in cache but no longer on disk
}

// Options controls scanning behaviour.
type Options struct {
	// Incremental: if true, compare against cache and only return changed files.
	Incremental bool
	// ComputeHash: compute SHA-256 for each file (slower but more accurate).
	ComputeHash bool
	// ExcludePatterns: glob patterns relative to root to exclude.
	ExcludePatterns []string
	// FollowSymlinks: follow symbolic links (default: false).
	FollowSymlinks bool
}

// Scanner walks a directory tree, building a list of changed files.
type Scanner struct {
	root  string
	cache *Cache
	opts  Options
}

// New creates a Scanner for root with an existing (possibly empty) cache.
func New(root string, cache *Cache, opts Options) *Scanner {
	return &Scanner{root: root, cache: cache, opts: opts}
}

// Scan walks the directory and returns changed + deleted files.
func (s *Scanner) Scan() (*ScanResult, *Cache, error) {
	result := &ScanResult{}
	newCache := NewCache()

	// Track which entries we've seen (for deletion detection)
	seen := make(map[string]bool)

	walkFn := func(path string, de os.DirEntry, err error) error {
		if err != nil {
			// Log and skip inaccessible paths
			return nil
		}

		rel, relErr := filepath.Rel(s.root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		// Skip root itself
		if rel == "." {
			return nil
		}

		// Check exclude patterns
		if s.isExcluded(rel) {
			if de.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := de.Info()
		if err != nil {
			return nil
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 && !s.opts.FollowSymlinks {
			return nil
		}

		fi := FileInfo{
			Path:    rel,
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
			IsDir:   de.IsDir(),
		}

		if !de.IsDir() && s.opts.ComputeHash {
			h, hashErr := hashFile(path)
			if hashErr == nil {
				fi.SHA256 = h
			}
		}

		seen[rel] = true
		newCache.Entries[rel] = &fi

		if !de.IsDir() {
			if s.hasChanged(rel, fi) {
				result.Changed = append(result.Changed, fi)
			}
		}

		return nil
	}

	if err := filepath.WalkDir(s.root, walkFn); err != nil {
		return nil, nil, fmt.Errorf("scanner: walk %q: %w", s.root, err)
	}

	// Detect deletions: entries in old cache not seen in current walk
	if s.opts.Incremental {
		for rel, old := range s.cache.Entries {
			if !seen[rel] {
				result.Deleted = append(result.Deleted, *old)
			}
		}
	}

	return result, newCache, nil
}

// hasChanged returns true if the file is new or differs from the cached entry.
func (s *Scanner) hasChanged(rel string, fi FileInfo) bool {
	if !s.opts.Incremental {
		return true // full backup: treat everything as changed
	}
	old, ok := s.cache.Entries[rel]
	if !ok {
		return true // new file
	}
	if fi.Size != old.Size || !fi.ModTime.Equal(old.ModTime) {
		return true // size or mtime changed
	}
	if s.opts.ComputeHash && fi.SHA256 != "" && old.SHA256 != "" {
		return fi.SHA256 != old.SHA256
	}
	return false
}

func (s *Scanner) isExcluded(rel string) bool {
	for _, pat := range s.opts.ExcludePatterns {
		matched, err := filepath.Match(pat, rel)
		if err == nil && matched {
			return true
		}
		// Also check if any path segment matches
		if strings.Contains(rel, pat) {
			return true
		}
	}
	return false
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
