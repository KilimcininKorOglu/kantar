# Feature 007: Cache Layer

**Feature ID:** F007
**Feature Name:** Cache Layer
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 0.5 week
**Status:** NOT_STARTED

## Overview

Upstream metadata ve paket response'larını önbelleğe alma. In-memory cache (varsayılan) ve Redis backend. Konfigürasyondan TTL, max size ayarları. Cache invalidation stratejisi.

## Goals

- Backend-agnostik cache interface
- In-memory LRU cache implementasyonu
- Upstream response cache ile ağ trafiğini azaltma
- Konfigüre edilebilir TTL ve max size

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T030-T032)
- [ ] Cache hit/miss doğru çalışıyor
- [ ] TTL sonrasında entry'ler expire oluyor
- [ ] Max size aşıldığında LRU eviction çalışıyor

## Tasks

### T030: Cache Interface Tanımı

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

Cache soyutlama katmanı. Get, Set, Delete, Exists, Flush, Stats operasyonları.

#### Technical Details

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Flush(ctx context.Context) error
    Stats(ctx context.Context) (*CacheStats, error)
}
```

#### Files to Touch

- `internal/cache/cache.go` (new)
- `internal/cache/types.go` (new)

#### Dependencies

- T001

#### Success Criteria

- [ ] Interface tanımlı
- [ ] CacheStats tipi tanımlı (hits, misses, size, entries)

---

### T031: In-Memory LRU Cache

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Thread-safe in-memory LRU cache. Max size (bytes) ve TTL desteği. Config'den `max_size` ve `ttl` okunacak.

#### Files to Touch

- `internal/cache/memory.go` (new)
- `internal/cache/memory_test.go` (new)

#### Dependencies

- T030, T006

#### Success Criteria

- [ ] LRU eviction çalışıyor
- [ ] TTL expiration çalışıyor
- [ ] Thread-safe (concurrent access)
- [ ] Max size byte bazlı hesaplanıyor
- [ ] Stats doğru raporlanıyor

---

### T032: Redis Cache Backend

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Redis tabanlı cache implementasyonu. Connection pooling, serialization, key prefix.

#### Files to Touch

- `internal/cache/redis.go` (new)
- `internal/cache/redis_test.go` (new)

#### Dependencies

- T030, T006

#### Success Criteria

- [ ] Redis'e yazma/okuma çalışıyor
- [ ] TTL Redis'e delege ediliyor
- [ ] Bağlantı hatalarında graceful degradation
- [ ] Key prefix ile namespace isolation

## Performance Targets

- Cache hit latency (memory): < 1ms
- Cache hit latency (Redis): < 5ms
- Memory cache max size: configurable (default 2GB)

## Risk Assessment

- **Düşük Risk:** LRU cache iyi bilinen bir yapı
- **Dikkat:** Redis bağlantı kaybında fallback stratejisi gerekli
