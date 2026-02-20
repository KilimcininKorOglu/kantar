package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
	size      int64
}

func (e *memoryEntry) expired() bool {
	return time.Now().After(e.expiresAt)
}

// Memory implements an in-memory LRU-like cache with TTL support.
type Memory struct {
	mu         sync.RWMutex
	entries    map[string]*memoryEntry
	maxBytes   int64
	currentBytes int64
	defaultTTL time.Duration
	hits       atomic.Int64
	misses     atomic.Int64
}

// NewMemory creates a new in-memory cache.
// maxBytes limits total cache size; 0 means unlimited.
// defaultTTL is used when Set is called with ttl=0.
func NewMemory(maxBytes int64, defaultTTL time.Duration) *Memory {
	m := &Memory{
		entries:    make(map[string]*memoryEntry),
		maxBytes:   maxBytes,
		defaultTTL: defaultTTL,
	}

	// Start background eviction goroutine
	go m.evictLoop()

	return m
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if !ok || entry.expired() {
		m.misses.Add(1)
		if ok && entry.expired() {
			m.mu.Lock()
			m.removeEntry(key)
			m.mu.Unlock()
		}
		return nil, nil
	}

	m.hits.Add(1)
	// Return a copy to prevent mutation
	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	return result, nil
}

func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = m.defaultTTL
	}

	// Copy value to prevent external mutation
	stored := make([]byte, len(value))
	copy(stored, value)

	entrySize := int64(len(stored))

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove old entry if exists
	if old, ok := m.entries[key]; ok {
		m.currentBytes -= old.size
	}

	// Evict if over capacity
	if m.maxBytes > 0 {
		for m.currentBytes+entrySize > m.maxBytes && len(m.entries) > 0 {
			m.evictOne()
		}
	}

	m.entries[key] = &memoryEntry{
		value:     stored,
		expiresAt: time.Now().Add(ttl),
		size:      entrySize,
	}
	m.currentBytes += entrySize

	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeEntry(key)
	return nil
}

func (m *Memory) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()

	if !ok || entry.expired() {
		return false, nil
	}
	return true, nil
}

func (m *Memory) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]*memoryEntry)
	m.currentBytes = 0
	return nil
}

func (m *Memory) Stats(_ context.Context) (*CacheStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &CacheStats{
		Hits:    m.hits.Load(),
		Misses:  m.misses.Load(),
		Entries: int64(len(m.entries)),
		Bytes:   m.currentBytes,
	}, nil
}

func (m *Memory) removeEntry(key string) {
	if entry, ok := m.entries[key]; ok {
		m.currentBytes -= entry.size
		delete(m.entries, key)
	}
}

// evictOne removes the oldest expired entry, or if none expired, the first entry found.
func (m *Memory) evictOne() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range m.entries {
		if entry.expired() {
			m.removeEntry(key)
			return
		}
		if first || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
			first = false
		}
	}

	if oldestKey != "" {
		m.removeEntry(oldestKey)
	}
}

func (m *Memory) evictLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		for key, entry := range m.entries {
			if entry.expired() {
				m.removeEntry(key)
			}
		}
		m.mu.Unlock()
	}
}
