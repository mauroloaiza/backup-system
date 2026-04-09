package files

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Result holds the outcome of a file backup.
type Result struct {
	Archive    string
	FilesTotal int
	BytesTotal int64
	Duration   time.Duration
}

// Backup creates a tar.gz archive of the given paths in destDir.
func Backup(paths []string, excludes []string, destDir string) (Result, error) {
	start := time.Now()

	archiveName := fmt.Sprintf("files_%s.tar.gz", time.Now().Format("20060102_150405"))
	archivePath := filepath.Join(destDir, archiveName)

	f, err := os.Create(archivePath)
	if err != nil {
		return Result{}, fmt.Errorf("crear archivo: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	var filesTotal int
	var bytesTotal int64

	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsPermission(walkErr) {
					return nil
				}
				return walkErr
			}
			if shouldExclude(path, excludes) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			if d.IsDir() {
				return tw.WriteHeader(&tar.Header{
					Name:     rel + "/",
					Typeflag: tar.TypeDir,
					Mode:     int64(info.Mode()),
					ModTime:  info.ModTime(),
				})
			}

			if d.Type()&fs.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return nil
				}
				return tw.WriteHeader(&tar.Header{
					Name:     rel,
					Typeflag: tar.TypeSymlink,
					Linkname: target,
					ModTime:  info.ModTime(),
				})
			}

			if err := tw.WriteHeader(&tar.Header{
				Name:    rel,
				Size:    info.Size(),
				Mode:    int64(info.Mode()),
				ModTime: info.ModTime(),
			}); err != nil {
				return err
			}

			src, err := os.Open(path)
			if err != nil {
				if os.IsPermission(err) {
					return nil
				}
				return err
			}
			defer src.Close()

			n, err := io.Copy(tw, src)
			if err != nil {
				return err
			}
			filesTotal++
			bytesTotal += n
			return nil
		})
		if err != nil {
			return Result{}, fmt.Errorf("walk %s: %w", root, err)
		}
	}

	return Result{
		Archive:    archivePath,
		FilesTotal: filesTotal,
		BytesTotal: bytesTotal,
		Duration:   time.Since(start),
	}, nil
}

func shouldExclude(path string, excludes []string) bool {
	for _, ex := range excludes {
		if strings.HasPrefix(path, ex) {
			return true
		}
		if matched, _ := filepath.Match(ex, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}
