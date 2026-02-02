# Feature 004: Core HTTP Server

**Feature ID:** F004
**Feature Name:** Core HTTP Server
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

chi router tabanlı HTTP server. Middleware zinciri (logging, recovery, CORS, auth, rate limiting), health endpoint, graceful shutdown, TLS desteği. Tüm plugin route'ları bu sunucu üzerine mount edilecek.

## Goals

- Production-ready HTTP server
- Middleware zinciri ile cross-cutting concern'ler
- Graceful shutdown ile veri kaybı önleme
- Plugin route'larını dinamik mount etme

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T015-T019)
- [ ] Server başlıyor ve isteklere yanıt veriyor
- [ ] Middleware zinciri çalışıyor
- [ ] Graceful shutdown sinyalleri yakalanıyor
- [ ] Health check endpoint'i cevap veriyor

## Tasks

### T015: HTTP Server Altyapısı

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

chi router ile HTTP server kurulumu, konfigürasyona göre host/port/TLS ayarları, graceful shutdown.

#### Technical Details

```go
type Server struct {
    router chi.Router
    config *config.ServerConfig
    srv    *http.Server
}

func (s *Server) Start(ctx context.Context) error
func (s *Server) Shutdown(ctx context.Context) error
```

Graceful shutdown: SIGINT/SIGTERM yakalama, mevcut isteklerin tamamlanmasını bekleme.

#### Files to Touch

- `internal/server/server.go` (new)
- `internal/server/server_test.go` (new)

#### Dependencies

- T006 (config struct'ları)

#### Success Criteria

- [ ] Server konfigürasyona göre başlıyor
- [ ] Graceful shutdown çalışıyor
- [ ] TLS desteği mevcut
- [ ] Port already in use durumunda anlaşılır hata

---

### T016: Middleware Zinciri

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Standart middleware'ler: request logging, panic recovery, CORS, request ID, timeout, gzip compression.

#### Technical Details

Middleware sırası:
1. RequestID
2. RealIP
3. Logger (structured JSON)
4. Recoverer
5. Timeout
6. CORS
7. Compress

#### Files to Touch

- `internal/server/middleware/logging.go` (new)
- `internal/server/middleware/recovery.go` (new)
- `internal/server/middleware/cors.go` (new)
- `internal/server/middleware/requestid.go` (new)

#### Dependencies

- T015

#### Success Criteria

- [ ] Her istek loglanıyor (method, path, status, duration)
- [ ] Panic'ler yakalanıp 500 dönüyor
- [ ] CORS header'ları yapılandırılabilir
- [ ] Request ID her istekte üretiliyor

---

### T017: Health & Status Endpoint'leri

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

`/api/v1/system/status` endpoint'i: server durumu, uptime, veritabanı bağlantısı, registry durumları, bellek kullanımı.

#### Files to Touch

- `internal/server/handlers/system.go` (new)

#### Dependencies

- T015, T010 (DB health check)

#### Success Criteria

- [ ] Health endpoint JSON response dönüyor
- [ ] DB bağlantı durumu raporlanıyor
- [ ] Uptime ve versiyon bilgisi mevcut

---

### T018: Rate Limiting

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 1 day

#### Description

Per-user ve per-IP rate limiting. Token bucket veya sliding window algoritması. Konfigürasyondan limitleri okuma.

#### Files to Touch

- `internal/server/middleware/ratelimit.go` (new)
- `internal/server/middleware/ratelimit_test.go` (new)

#### Dependencies

- T016

#### Success Criteria

- [ ] Per-IP rate limiting çalışıyor
- [ ] Per-user rate limiting çalışıyor (auth sonrası)
- [ ] 429 Too Many Requests doğru dönüyor
- [ ] Rate limit header'ları mevcut (X-RateLimit-*)

---

### T019: Route Mounting Altyapısı

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Plugin'lerin route'larını ana router'a mount etme mekanizması. Her plugin kendi sub-router'ını döndürür, core engine bunu namespace altında mount eder.

#### Technical Details

```go
// Plugin route'ları mount edilecek:
// /v2/*          → Docker plugin
// /npm/*         → npm plugin
// /pypi/*        → PyPI plugin
// /go/*          → Go Modules plugin
// /cargo/*       → Cargo plugin
// /maven/*       → Maven plugin
// /nuget/*       → NuGet plugin
// /helm/*        → Helm plugin
// /api/v1/*      → Management REST API
```

#### Files to Touch

- `internal/server/router.go` (new)
- `internal/server/router_test.go` (new)

#### Dependencies

- T015, F008 (Plugin Architecture — interface tanımı)

#### Success Criteria

- [ ] Plugin route'ları dinamik olarak mount ediliyor
- [ ] Her plugin kendi namespace'inde izole
- [ ] Management API ayrı prefix altında
- [ ] Route conflict'leri tespit ediliyor

## Performance Targets

- Request latency overhead (middleware): < 1ms
- Concurrent connections: 10,000+
- Graceful shutdown timeout: configurable (default 30s)

## Risk Assessment

- **Düşük Risk:** chi router çok olgun bir kütüphane
- **Dikkat:** Rate limiting state'i single-node'da in-memory olacak; multi-node'da Redis gerekecek (v2)
