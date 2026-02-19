package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Filesystem implements Storage using the local file system.
type Filesystem struct {
	basePath string
}

// NewFilesystem creates a new filesystem storage rooted at basePath.
func NewFilesystem(basePath string) (*Filesystem, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage directory %s: %w", basePath, err)
	}
	return &Filesystem{basePath: basePath}, nil
}

func (f *Filesystem) resolve(path string) (string, error) {
	// Prevent path traversal
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected: %s", path)
	}
	return filepath.Join(f.basePath, cleaned), nil
}

func (f *Filesystem) Put(_ context.Context, path string, reader io.Reader) error {
	fullPath, err := f.resolve(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Atomic write: write to temp file, then rename
	tmpFile, err := os.CreateTemp(dir, ".kantar-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing to temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

func (f *Filesystem) Get(_ context.Context, path string) (io.ReadCloser, error) {
	fullPath, err := f.resolve(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("opening file: %w", err)
	}

	return file, nil
}

func (f *Filesystem) Delete(_ context.Context, path string) error {
	fullPath, err := f.resolve(path)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("removing file: %w", err)
	}

	return nil
}

func (f *Filesystem) Exists(_ context.Context, path string) (bool, error) {
	fullPath, err := f.resolve(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat file: %w", err)
	}
	return true, nil
}

func (f *Filesystem) Stat(_ context.Context, path string) (*FileInfo, error) {
	fullPath, err := f.resolve(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("stat file: %w", err)
	}

	return &FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

func (f *Filesystem) List(_ context.Context, prefix string) ([]FileInfo, error) {
	fullPrefix, err := f.resolve(prefix)
	if err != nil {
		return nil, err
	}

	var files []FileInfo

	err = filepath.WalkDir(fullPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		relPath, _ := filepath.Rel(f.basePath, path)
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}

		files = append(files, FileInfo{
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   d.IsDir(),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}

	return files, nil
}

func (f *Filesystem) Usage(_ context.Context, prefix string) (*UsageInfo, error) {
	fullPrefix, err := f.resolve(prefix)
	if err != nil {
		return nil, err
	}

	var totalBytes int64
	var fileCount int64

	err = filepath.WalkDir(fullPrefix, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr == nil {
				totalBytes += info.Size()
				fileCount++
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("calculating usage: %w", err)
	}

	return &UsageInfo{TotalBytes: totalBytes, FileCount: fileCount}, nil
}
