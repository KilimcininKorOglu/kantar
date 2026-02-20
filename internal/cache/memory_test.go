package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemorySetAndGet(t *testing.T) {
	c := NewMemory(0, 1*time.Hour)
	ctx := context.Background()

	if err := c.Set(ctx, "key1", []byte("value1"), 0); err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := c.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("got %q, want %q", val, "value1")
	}
}

func TestMemoryGetMiss(t *testing.T) {
	c := NewMemory(0, 1*time.Hour)
	ctx := context.Background()

	val, err := c.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil, got %q", val)
	}
}

func TestMemoryExpiration(t *testing.T) {
	c := NewMemory(0, 1*time.Hour)
	ctx := context.Background()

	if err := c.Set(ctx, "expire", []byte("data"), 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}

	val, _ := c.Get(ctx, "expire")
	if val == nil {
		t.Fatal("expected value before expiry")
	}

	time.Sleep(60 * time.Millisecond)

	val, _ = c.Get(ctx, "expire")
	if val != nil {
		t.Error("expected nil after expiry")
	}
}

func TestMemoryMaxBytes(t *testing.T) {
	c := NewMemory(10, 1*time.Hour) // 10 bytes max
	ctx := context.Background()

	c.Set(ctx, "a", []byte("12345"), 0) // 5 bytes
	c.Set(ctx, "b", []byte("12345"), 0) // 5 bytes, total = 10

	stats, _ := c.Stats(ctx)
	if stats.Entries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.Entries)
	}

	c.Set(ctx, "c", []byte("12345"), 0) // should evict one

	stats, _ = c.Stats(ctx)
	if stats.Entries != 2 {
		t.Errorf("expected 2 entries after eviction, got %d", stats.Entries)
	}
	if stats.Bytes > 10 {
		t.Errorf("expected <= 10 bytes, got %d", stats.Bytes)
	}
}

func TestMemoryDelete(t *testing.T) {
	c := NewMemory(0, 1*time.Hour)
	ctx := context.Background()

	c.Set(ctx, "del", []byte("data"), 0)
	c.Delete(ctx, "del")

	exists, _ := c.Exists(ctx, "del")
	if exists {
		t.Error("expected not exists after delete")
	}
}

func TestMemoryFlush(t *testing.T) {
	c := NewMemory(0, 1*time.Hour)
	ctx := context.Background()

	c.Set(ctx, "a", []byte("1"), 0)
	c.Set(ctx, "b", []byte("2"), 0)

	c.Flush(ctx)

	stats, _ := c.Stats(ctx)
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries after flush, got %d", stats.Entries)
	}
}

func TestMemoryStats(t *testing.T) {
	c := NewMemory(0, 1*time.Hour)
	ctx := context.Background()

	c.Set(ctx, "k", []byte("val"), 0)
	c.Get(ctx, "k")     // hit
	c.Get(ctx, "miss")  // miss

	stats, _ := c.Stats(ctx)
	if stats.Hits != 1 {
		t.Errorf("hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("misses = %d, want 1", stats.Misses)
	}
	if stats.Entries != 1 {
		t.Errorf("entries = %d, want 1", stats.Entries)
	}
}
