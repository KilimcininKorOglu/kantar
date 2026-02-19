package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestFilesystemPutAndGet(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(dir)
	if err != nil {
		t.Fatalf("create filesystem: %v", err)
	}

	ctx := context.Background()
	data := []byte("hello kantar")

	// Put
	if err := fs.Put(ctx, "npm/express/4.18.2.tgz", bytes.NewReader(data)); err != nil {
		t.Fatalf("put: %v", err)
	}

	// Get
	reader, err := fs.Get(ctx, "npm/express/4.18.2.tgz")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer reader.Close()

	got, _ := io.ReadAll(reader)
	if !bytes.Equal(got, data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestFilesystemExists(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()

	exists, _ := fs.Exists(ctx, "nonexistent")
	if exists {
		t.Error("expected not exists")
	}

	fs.Put(ctx, "test.txt", bytes.NewReader([]byte("data")))
	exists, _ = fs.Exists(ctx, "test.txt")
	if !exists {
		t.Error("expected exists")
	}
}

func TestFilesystemDelete(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()

	fs.Put(ctx, "deleteme.txt", bytes.NewReader([]byte("data")))

	if err := fs.Delete(ctx, "deleteme.txt"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	exists, _ := fs.Exists(ctx, "deleteme.txt")
	if exists {
		t.Error("expected not exists after delete")
	}
}

func TestFilesystemStat(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()

	data := []byte("some content")
	fs.Put(ctx, "stat.txt", bytes.NewReader(data))

	info, err := fs.Stat(ctx, "stat.txt")
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size != int64(len(data)) {
		t.Errorf("size = %d, want %d", info.Size, len(data))
	}
}

func TestFilesystemUsage(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()

	fs.Put(ctx, "a.txt", bytes.NewReader([]byte("aaaa")))
	fs.Put(ctx, "b.txt", bytes.NewReader([]byte("bbbbbb")))

	usage, err := fs.Usage(ctx, "")
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if usage.FileCount != 2 {
		t.Errorf("fileCount = %d, want 2", usage.FileCount)
	}
	if usage.TotalBytes != 10 {
		t.Errorf("totalBytes = %d, want 10", usage.TotalBytes)
	}
}

func TestFilesystemPathTraversal(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()

	err := fs.Put(ctx, "../../../etc/passwd", bytes.NewReader([]byte("bad")))
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestFilesystemGetNotFound(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()

	_, err := fs.Get(ctx, "nonexistent.txt")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
