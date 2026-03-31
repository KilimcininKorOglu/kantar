// Package storage provides the file storage abstraction for Kantar.
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// Standard errors returned by storage implementations.
var (
	ErrNotFound      = errors.New("file not found")
	ErrAlreadyExists = errors.New("file already exists")
)

// FileInfo holds metadata about a stored file.
type FileInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
}

// UsageInfo holds disk usage statistics.
type UsageInfo struct {
	TotalBytes int64 `json:"totalBytes"`
	FileCount  int64 `json:"fileCount"`
}

// Storage defines the interface for storing and retrieving package files.
type Storage interface {
	// Put stores data at the given path. Overwrites if already exists.
	Put(ctx context.Context, path string, reader io.Reader) error

	// Get retrieves data from the given path. Caller must close the reader.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes the file at the given path.
	Delete(ctx context.Context, path string) error

	// Exists checks if a file exists at the given path.
	Exists(ctx context.Context, path string) (bool, error)

	// Stat returns file metadata.
	Stat(ctx context.Context, path string) (*FileInfo, error)

	// List returns files matching the given prefix.
	List(ctx context.Context, prefix string) ([]FileInfo, error)

	// Usage returns disk usage statistics for the given prefix (or root if empty).
	Usage(ctx context.Context, prefix string) (*UsageInfo, error)
}
