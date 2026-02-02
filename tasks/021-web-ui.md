# Feature 021: Web UI (Admin Dashboard)

**Feature ID:** F021
**Feature Name:** Web UI (Admin Dashboard)
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 3 weeks
**Status:** NOT_STARTED

## Overview

React + Vite ile admin dashboard SPA. Go `embed` ile binary'ye gömülecek. Dashboard, paket yönetimi, registry durumu, kullanıcı yönetimi, audit log, ayarlar. PRD Section 5.4'teki wireframe'lere uygun.

## Goals

- Tüm admin işlevlerini web üzerinden yönetme
- Responsive ve hızlı (< 1s sayfa yükleme)
- REST API'yi client olarak kullanma
- Go binary'ye embed edilme

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T093-T100)
- [ ] Dashboard sayfası çalışıyor
- [ ] Paket yönetimi sayfaları çalışıyor
- [ ] SPA Go binary'ye embed ediliyor

## Tasks

### T093: React + Vite Proje Kurulumu

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

React + Vite + TypeScript proje kurulumu. Router, HTTP client (fetch/axios), temel layout.

#### Files to Touch

- `web/package.json` (new)
- `web/vite.config.ts` (new)
- `web/tsconfig.json` (new)
- `web/src/main.tsx` (new)
- `web/src/App.tsx` (new)

#### Dependencies

- T001

#### Success Criteria

- [ ] `npm run dev` ile dev server çalışıyor
- [ ] `npm run build` ile production build üretilebiliyor
- [ ] TypeScript derleniyor
- [ ] Temel routing çalışıyor

---

### T094: Go Embed Entegrasyonu

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

React build çıktısını Go `embed` ile binary'ye gömme. SPA routing için fallback handler.

#### Technical Details

```go
//go:embed web/dist
var webUI embed.FS

// SPA fallback: bilinmeyen path'ler → index.html
```

#### Files to Touch

- `internal/server/web.go` (new)
- `web.go` (new — embed directive, project root)

#### Dependencies

- T093, T015

#### Success Criteria

- [ ] Build sonrası tek binary web UI'ı serve ediyor
- [ ] SPA routing çalışıyor (deep link'ler)
- [ ] Static asset'ler doğru serve ediliyor

---

### T095: Auth UI (Login/Logout)

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Login sayfası, token yönetimi, protected route'lar, logout.

#### Files to Touch

- `web/src/pages/Login.tsx` (new)
- `web/src/contexts/AuthContext.tsx` (new)
- `web/src/hooks/useAuth.ts` (new)
- `web/src/components/ProtectedRoute.tsx` (new)

#### Dependencies

- T093, T025 (auth API)

#### Success Criteria

- [ ] Login formu çalışıyor
- [ ] JWT token localStorage'da saklanıyor
- [ ] Yetkisiz erişimde login'e yönlendirme
- [ ] Logout çalışıyor

---

### T096: Dashboard Sayfası

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

PRD Section 5.4.1'deki dashboard wireframe'ine uygun: istatistikler, registry health, pending approvals, recent activity.

#### Files to Touch

- `web/src/pages/Dashboard.tsx` (new)
- `web/src/components/StatsCard.tsx` (new)
- `web/src/components/RegistryHealth.tsx` (new)
- `web/src/components/PendingApprovals.tsx` (new)
- `web/src/components/RecentActivity.tsx` (new)

#### Dependencies

- T095, T017 (system status API), T042 (package API)

#### Success Criteria

- [ ] İstatistikler gösteriliyor (paket sayısı, storage, kullanıcı)
- [ ] Registry health durumları gösteriliyor
- [ ] Pending approvals listesi çalışıyor
- [ ] Recent activity akışı çalışıyor

---

### T097: Package Management Sayfaları

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Paket listeleme, arama, detay görüntüleme, onaylama/engelleme. PRD Section 5.4.2'deki package detail view.

#### Files to Touch

- `web/src/pages/PackageList.tsx` (new)
- `web/src/pages/PackageDetail.tsx` (new)
- `web/src/components/PackageTable.tsx` (new)
- `web/src/components/VersionList.tsx` (new)
- `web/src/components/ApprovalActions.tsx` (new)

#### Dependencies

- T095, T042

#### Success Criteria

- [ ] Paket listesi pagination ve arama ile çalışıyor
- [ ] Paket detay sayfası versiyon listesi gösteriyor
- [ ] Approve/block butonları çalışıyor
- [ ] Dependency listesi gösteriliyor

---

### T098: User Management Sayfası

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Kullanıcı listeleme, oluşturma, düzenleme, rol ataması, token yönetimi.

#### Files to Touch

- `web/src/pages/Users.tsx` (new)
- `web/src/components/UserForm.tsx` (new)
- `web/src/components/TokenManager.tsx` (new)

#### Dependencies

- T095, T025

#### Success Criteria

- [ ] Kullanıcı listesi çalışıyor
- [ ] Kullanıcı oluşturma formu çalışıyor
- [ ] Rol ataması çalışıyor
- [ ] Token oluşturma/listeleme/silme çalışıyor

---

### T099: Audit Log Sayfası

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Audit log listesi, filtreleme, CSV/JSON export.

#### Files to Touch

- `web/src/pages/AuditLog.tsx` (new)
- `web/src/components/AuditFilter.tsx` (new)

#### Dependencies

- T095, T053 (audit API)

#### Success Criteria

- [ ] Audit log listesi çalışıyor
- [ ] Filtreleme (tarih, actor, event) çalışıyor
- [ ] Export butonları çalışıyor

---

### T100: Settings ve Dependency Graph Sayfaları

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 2 days

#### Description

Sistem ayarları sayfası, registry yapılandırma, policy görüntüleme. Dependency graph visualization.

#### Files to Touch

- `web/src/pages/Settings.tsx` (new)
- `web/src/pages/DependencyGraph.tsx` (new)
- `web/src/components/RegistryConfig.tsx` (new)

#### Dependencies

- T095

#### Success Criteria

- [ ] Registry ayarları görüntülenebiliyor
- [ ] Dependency graph görselleştirmesi çalışıyor
- [ ] Policy listesi görüntülenebiliyor

## Performance Targets

- Initial load: < 1s
- Page transitions: < 200ms
- Bundle size: < 500KB gzipped

## Risk Assessment

- **Orta Risk:** Frontend framework seçimi ve component library
- **Çözüm:** Minimal bağımlılık, Tailwind CSS veya headless UI tercih edilebilir
- **Dikkat:** Go embed ve dev mode arasındaki geçiş (dev'de Vite proxy, prod'da embed)
