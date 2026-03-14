# Feature 023: Complete Web UI (React + Vite)

**Feature ID:** F023
**Feature Name:** Complete Web UI — React Admin Dashboard
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 4 weeks
**Status:** NOT_STARTED

## Overview

PRD Section 5.4'teki wireframe'lere tam uyumlu, production-grade React + Vite + TypeScript admin dashboard. Mevcut placeholder HTML'yi tam bir SPA ile değiştirir. Tailwind CSS ile styling, React Router ile navigation, fetch API ile backend iletişimi.

Dashboard, paket yönetimi (listeleme, arama, detay, onay/engelleme), registry durumu, kullanıcı yönetimi, audit log, policy görüntüleme ve dependency graph sayfalarını içerir.

## Goals

- PRD wireframe'lerine tam uyumlu admin dashboard
- Responsive ve hızlı (< 1s sayfa yükleme, < 200ms geçişler)
- REST API client ile tüm backend işlemleri
- Gerçek zamanlı durum gösterimi (registry health, pending approvals)
- Go `embed` ile binary'ye gömülme (mevcut altyapı kullanılacak)

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T107-T118)
- [ ] Dashboard sayfası PRD wireframe'ine uygun
- [ ] Paket yönetimi tam çalışıyor (CRUD + approval)
- [ ] Audit log filtreleme ve export çalışıyor
- [ ] `npm run build` sonrası Go binary embed ediyor
- [ ] Bundle size < 500KB gzipped
- [ ] Lighthouse performance score > 90

## Tasks

### T107: React + Vite + TypeScript Proje Kurulumu

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Vite ile React + TypeScript projesi. Tailwind CSS 4, React Router v7, temel proje yapısı. ESLint + Prettier yapılandırması. Vite proxy ile dev modda backend'e yönlendirme.

#### Technical Details

```
web/
  src/
    main.tsx              # Entry point
    App.tsx               # Root component + Router
    api/                  # API client layer
      client.ts           # fetch wrapper with auth
      types.ts            # API response types
    components/           # Shared components
      layout/             # Layout components
    pages/                # Page components
    hooks/                # Custom hooks
    contexts/             # React contexts
    lib/                  # Utility functions
  index.html
  vite.config.ts
  tsconfig.json
  tailwind.config.ts
  package.json
```

Bağımlılıklar:
- `react`, `react-dom`, `react-router` (v7)
- `tailwindcss` (v4), `@tailwindcss/vite`
- `typescript`, `@types/react`, `@types/react-dom`
- `eslint`, `prettier`

#### Files to Touch

- `web/package.json` (new)
- `web/vite.config.ts` (new)
- `web/tsconfig.json` (new)
- `web/index.html` (new)
- `web/tailwind.config.ts` (new)
- `web/src/main.tsx` (new)
- `web/src/App.tsx` (new)

#### Dependencies

- None (mevcut `web/web.go` embed altyapısı zaten var)

#### Success Criteria

- [ ] `npm run dev` ile dev server çalışıyor
- [ ] `npm run build` ile production build `web/dist/` altına üretiliyor
- [ ] TypeScript derleniyor, ESLint hatasız
- [ ] Tailwind CSS sınıfları çalışıyor
- [ ] Vite proxy ile `/api/*` backend'e yönleniyor

---

### T108: API Client Layer ve TypeScript Types

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Backend REST API ile iletişim katmanı. fetch wrapper ile JWT auth, error handling, response typing. Tüm API endpoint'leri için type-safe fonksiyonlar.

#### Technical Details

```typescript
// api/client.ts
class ApiClient {
  async get<T>(path: string): Promise<T>
  async post<T>(path: string, body: unknown): Promise<T>
  async put<T>(path: string, body: unknown): Promise<T>
  async delete(path: string): Promise<void>
}

// api/types.ts
interface Package { id: number; name: string; status: string; ... }
interface User { id: number; username: string; role: string; ... }
interface AuditLog { id: number; event: string; actor: Actor; ... }
interface SystemStatus { status: string; uptime: string; ... }
```

#### Files to Touch

- `web/src/api/client.ts` (new)
- `web/src/api/types.ts` (new)
- `web/src/api/packages.ts` (new)
- `web/src/api/users.ts` (new)
- `web/src/api/registries.ts` (new)
- `web/src/api/audit.ts` (new)
- `web/src/api/system.ts` (new)

#### Dependencies

- T107

#### Success Criteria

- [ ] Tüm API endpoint'leri için type-safe fonksiyonlar
- [ ] JWT token otomatik ekleniyor (Authorization header)
- [ ] 401 response'da otomatik logout
- [ ] Error handling tutarlı
- [ ] TypeScript types tüm response şemalarını kapsıyor

---

### T109: Layout, Navigation ve Auth UI

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

PRD wireframe'ine uygun sidebar navigation layout. Auth context, login sayfası, JWT token yönetimi, protected route'lar. Sidebar: Overview, Packages, Registries, Users, Policies, Audit Log, Settings.

#### Technical Details

- Sidebar: fixed, 240px genişlik, collapsible
- Header: breadcrumb, user info, logout butonu
- AuthContext: login/logout, token storage (localStorage), user bilgisi
- ProtectedRoute: auth kontrolü, yetkisiz → login redirect
- Login sayfası: username + password form, error handling

#### Files to Touch

- `web/src/components/layout/Sidebar.tsx` (new)
- `web/src/components/layout/Header.tsx` (new)
- `web/src/components/layout/MainLayout.tsx` (new)
- `web/src/contexts/AuthContext.tsx` (new)
- `web/src/hooks/useAuth.ts` (new)
- `web/src/pages/Login.tsx` (new)
- `web/src/components/ProtectedRoute.tsx` (new)

#### Dependencies

- T107, T108

#### Success Criteria

- [ ] Sidebar navigation tüm menü öğelerini içeriyor
- [ ] Login/logout flow çalışıyor
- [ ] JWT token localStorage'da persist ediliyor
- [ ] Yetkisiz erişimde login'e yönlendirme
- [ ] Responsive: mobilde hamburger menü
- [ ] Aktif sayfa sidebar'da vurgulu

---

### T110: Dashboard Sayfası

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

PRD Section 5.4.1 wireframe'ine tam uyumlu dashboard. 4 istatistik kartı (paket sayısı, bekleyen, indirme, kullanıcı), registry health grid, pending approvals tablosu, recent activity akışı.

#### Technical Details

Dashboard bileşenleri:
1. **Stats Row:** 4 kart — Packages, Pending, Downloads (today), Users
2. **Registry Health Grid:** Her ekosistem için durum kartı (status, paket sayısı, boyut)
3. **Pending Approvals:** Tablo — Package, Version, Registry, Requested By, [Approve/Block] butonları
4. **Recent Activity:** Zaman akışı — event icon + description + timestamp

Auto-refresh: 30 saniyede bir API poll

#### Files to Touch

- `web/src/pages/Dashboard.tsx` (new)
- `web/src/components/dashboard/StatsCard.tsx` (new)
- `web/src/components/dashboard/RegistryHealthGrid.tsx` (new)
- `web/src/components/dashboard/PendingApprovals.tsx` (new)
- `web/src/components/dashboard/RecentActivity.tsx` (new)

#### Dependencies

- T109

#### Success Criteria

- [ ] 4 istatistik kartı API'den veri çekiyor
- [ ] Registry health grid 8 ekosistemi gösteriyor
- [ ] Pending approvals tablosu çalışıyor, approve/block butonları fonksiyonel
- [ ] Recent activity real-time güncelleneiyor (30s poll)
- [ ] Layout PRD wireframe'ine uygun

---

### T111: Package List Sayfası

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Paket listeleme sayfası: registry filtreleme, arama, status filtreleme, pagination, toplu onay/engelleme. Tablo ve kart görünümleri.

#### Technical Details

- Registry tab'ları (Docker, npm, PyPI, Go, Cargo, Maven, NuGet, Helm)
- Arama: debounced input (300ms)
- Status filtre: All, Pending, Approved, Blocked
- Pagination: sayfa numaraları + sonraki/önceki
- Bulk actions: checkbox selection + toplu approve/block
- Her satır: paket adı, son versiyon, status badge, boyut, tarih

#### Files to Touch

- `web/src/pages/PackageList.tsx` (new)
- `web/src/components/packages/PackageTable.tsx` (new)
- `web/src/components/packages/PackageFilters.tsx` (new)
- `web/src/components/packages/StatusBadge.tsx` (new)
- `web/src/components/common/Pagination.tsx` (new)
- `web/src/components/common/SearchInput.tsx` (new)
- `web/src/hooks/usePackages.ts` (new)

#### Dependencies

- T109, T108

#### Success Criteria

- [ ] Tüm 8 registry için paket listesi gösteriliyor
- [ ] Arama gerçek zamanlı çalışıyor (debounced)
- [ ] Status filtreleme çalışıyor
- [ ] Pagination doğru çalışıyor
- [ ] Toplu approve/block çalışıyor
- [ ] Boş durum (empty state) gösterimi

---

### T112: Package Detail Sayfası

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

PRD Section 5.4.2 wireframe'ine uygun paket detay sayfası. Versiyon listesi, paket bilgileri, indirme istatistikleri, bağımlılıklar, güvenlik açığı raporu, sync/approve/block aksiyonları.

#### Technical Details

Layout:
1. **Header:** Paket adı, description, status badge, [Approve]/[Block]/[Sync] butonları
2. **3-Column Grid:**
   - Versions: scrollable liste, current marker, beta/deprecated tag'ler
   - Info: License, Size, Deps count, Vulns, Last sync
   - Stats: Downloads (today/week/total)
3. **Dependencies:** Grid — her dep bir kart (ad + versiyon + status)
4. **Vulnerability Report:** Tablo veya "No known vulnerabilities" + [Scan] butonu

#### Files to Touch

- `web/src/pages/PackageDetail.tsx` (new)
- `web/src/components/packages/VersionList.tsx` (new)
- `web/src/components/packages/PackageInfo.tsx` (new)
- `web/src/components/packages/DependencyGrid.tsx` (new)
- `web/src/components/packages/VulnerabilityReport.tsx` (new)
- `web/src/components/packages/ApprovalActions.tsx` (new)

#### Dependencies

- T111

#### Success Criteria

- [ ] Versiyon listesi scrollable ve seçilebilir
- [ ] Paket metadata (license, size, deps) gösteriliyor
- [ ] Download stats gösteriliyor
- [ ] Approve/block/sync butonları çalışıyor
- [ ] Bağımlılıklar grid olarak gösteriliyor
- [ ] Layout PRD wireframe'ine uygun

---

### T113: Registry Management Sayfası

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Registry'leri listeleme ve yapılandırma sayfası. Her registry için durum, mod (allowlist/mirror), upstream URL, paket sayısı, son sync zamanı. Sync tetikleme butonu.

#### Files to Touch

- `web/src/pages/Registries.tsx` (new)
- `web/src/components/registries/RegistryCard.tsx` (new)
- `web/src/components/registries/SyncButton.tsx` (new)

#### Dependencies

- T109

#### Success Criteria

- [ ] 8 registry kartı gösteriliyor
- [ ] Her kartta durum, mod, upstream, paket sayısı
- [ ] Sync butonu çalışıyor
- [ ] Registry yapılandırma detayları görünüyor

---

### T114: User Management Sayfası

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1.5 days

#### Description

Kullanıcı CRUD, rol ataması, API token yönetimi. Kullanıcı listesi tablosu, oluşturma/düzenleme modal, token oluşturma dialog.

#### Files to Touch

- `web/src/pages/Users.tsx` (new)
- `web/src/components/users/UserTable.tsx` (new)
- `web/src/components/users/UserForm.tsx` (new)
- `web/src/components/users/TokenManager.tsx` (new)
- `web/src/components/common/Modal.tsx` (new)
- `web/src/components/common/ConfirmDialog.tsx` (new)

#### Dependencies

- T109

#### Success Criteria

- [ ] Kullanıcı listesi tablo olarak gösteriliyor
- [ ] Kullanıcı oluşturma/düzenleme modal çalışıyor
- [ ] Rol ataması dropdown ile çalışıyor
- [ ] Token oluşturma dialog (token yalnızca bir kez gösteriliyor)
- [ ] Kullanıcı silme onay dialog'u

---

### T115: Audit Log Sayfası

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1.5 days

#### Description

Audit log tablosu, gelişmiş filtreleme (tarih aralığı, event türü, actor, registry), CSV/JSON export, hash-chain doğrulama butonu.

#### Files to Touch

- `web/src/pages/AuditLog.tsx` (new)
- `web/src/components/audit/AuditTable.tsx` (new)
- `web/src/components/audit/AuditFilters.tsx` (new)
- `web/src/components/common/DateRangePicker.tsx` (new)
- `web/src/components/common/ExportButton.tsx` (new)

#### Dependencies

- T109

#### Success Criteria

- [ ] Audit log tablosu pagination ile çalışıyor
- [ ] Tarih aralığı filtreleme çalışıyor
- [ ] Event türü, actor, registry filtreleme çalışıyor
- [ ] CSV export çalışıyor
- [ ] JSON export çalışıyor
- [ ] Hash-chain verify butonu sonucu gösteriyor

---

### T116: Policy ve Settings Sayfaları

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 1 day

#### Description

Policy listesi görüntüleme (salt-okunur), sistem ayarları sayfası (salt-okunur config görüntüleme). Policy ihlal geçmişi.

#### Files to Touch

- `web/src/pages/Policies.tsx` (new)
- `web/src/pages/Settings.tsx` (new)
- `web/src/components/policies/PolicyCard.tsx` (new)
- `web/src/components/settings/ConfigView.tsx` (new)

#### Dependencies

- T109

#### Success Criteria

- [ ] Policy listesi kart olarak gösteriliyor
- [ ] Her policy'nin kuralları okunabilir formatta
- [ ] Sistem durumu ve config bilgileri gösteriliyor
- [ ] Policy ihlal geçmişi listeleniyor

---

### T117: Dependency Graph Görselleştirme

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 2 days

#### Description

Paket bağımlılık ağacını interaktif graf olarak görselleştirme. D3.js veya react-force-graph ile force-directed layout. Tıklanabilir node'lar (paket detayına link), status renklendirme.

#### Files to Touch

- `web/src/pages/DependencyGraph.tsx` (new)
- `web/src/components/graph/ForceGraph.tsx` (new)
- `web/src/components/graph/GraphControls.tsx` (new)
- `web/src/hooks/useDependencyGraph.ts` (new)

#### Dependencies

- T112

#### Success Criteria

- [ ] Bağımlılık grafı görselleştiriliyor
- [ ] Node'lar tıklanabilir (paket detayına navigation)
- [ ] Onaylı/bekleyen/engelli durumlar renk kodlu
- [ ] Zoom ve pan kontrolleri
- [ ] Performanslı (100+ node'da akıcı)

---

### T118: Shared UI Components ve Tema

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1.5 days

#### Description

Ortak UI bileşenleri: Button, Input, Select, Table, Card, Badge, Toast notification, Loading spinner, Empty state, Error boundary. Dark/light tema desteği.

#### Files to Touch

- `web/src/components/common/Button.tsx` (new)
- `web/src/components/common/Input.tsx` (new)
- `web/src/components/common/Select.tsx` (new)
- `web/src/components/common/Table.tsx` (new)
- `web/src/components/common/Card.tsx` (new)
- `web/src/components/common/Badge.tsx` (new)
- `web/src/components/common/Toast.tsx` (new)
- `web/src/components/common/Spinner.tsx` (new)
- `web/src/components/common/EmptyState.tsx` (new)
- `web/src/components/common/ErrorBoundary.tsx` (new)
- `web/src/lib/theme.ts` (new)

#### Dependencies

- T107

#### Success Criteria

- [ ] Tüm ortak bileşenler tutarlı tasarım dili
- [ ] Dark tema varsayılan (PRD wireframe'e uygun)
- [ ] Toast notifications çalışıyor
- [ ] Loading spinner tüm API çağrılarında gösteriliyor
- [ ] Error boundary beklenmeyen hataları yakalıyor
- [ ] Empty state bileşeni "veri yok" durumlarında gösteriliyor

## Performance Targets

- Initial load: < 1s
- Page transitions: < 200ms
- Bundle size: < 500KB gzipped
- Lighthouse performance: > 90

## Risk Assessment

- **Orta Risk:** React 19 + Vite 6 + Tailwind v4 yeni versiyonlar, breaking change riski
- **Çözüm:** Stable versiyonlar tercih edilecek, lock file ile sabitlenecek
- **Dikkat:** Bundle size — tree-shaking ve code splitting ile kontrol altında tutulmalı
- **Dikkat:** D3.js veya graf kütüphanesi büyük olabilir — dynamic import ile lazy load

## Notes

- Mevcut `web/web.go` embed altyapısı kullanılacak — sadece `web/dist/` içeriği değişecek
- Dev modda Vite proxy ile backend'e yönlendirme (`vite.config.ts` → proxy: `/api` → `http://localhost:8080`)
- Mevcut placeholder `web/dist/index.html` React build ile değiştirilecek
