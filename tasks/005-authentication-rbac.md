# Feature 005: Authentication & RBAC

**Feature ID:** F005
**Feature Name:** Authentication & RBAC
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

JWT tabanlı kimlik doğrulama, bcrypt ile parola hash'leme, 5 kademeli RBAC (Super Admin, Registry Admin, Publisher, Consumer, Viewer), API token yönetimi. v1.0'da yalnızca local authentication; LDAP/OIDC v1.1'de eklenecek.

## Goals

- Güvenli kimlik doğrulama (JWT + bcrypt)
- Namespace tabanlı RBAC
- CI/CD için API token desteği
- İlk admin hesabı oluşturma (bootstrap)

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T020-T025)
- [ ] Login/logout çalışıyor
- [ ] JWT token'ları geçerli ve süresi doluyor
- [ ] RBAC yetkilendirme doğru çalışıyor
- [ ] API token'lar ile erişim sağlanıyor

## Tasks

### T020: Kullanıcı Modeli ve Parola Hash'leme

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Kullanıcı veri modeli, bcrypt ile parola hash/verify, güvenli parola kuralları.

#### Files to Touch

- `internal/auth/user.go` (new)
- `internal/auth/password.go` (new)
- `internal/auth/password_test.go` (new)

#### Dependencies

- T011 (DB şeması — users tablosu)

#### Success Criteria

- [ ] Parola bcrypt ile hash'leniyor
- [ ] Parola doğrulama çalışıyor
- [ ] Minimum parola gereksinimleri uygulanıyor
- [ ] Plaintext parola hiçbir yerde saklanmıyor

---

### T021: JWT Token Üretimi ve Doğrulama

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

JWT token oluşturma, imzalama (HMAC-SHA256), doğrulama, yenileme. Token payload'ında kullanıcı ID, roller, expiry.

#### Files to Touch

- `internal/auth/jwt.go` (new)
- `internal/auth/jwt_test.go` (new)

#### Dependencies

- T020

#### Success Criteria

- [ ] JWT oluşturma ve imzalama çalışıyor
- [ ] Token doğrulama (signature + expiry) çalışıyor
- [ ] Süresi dolmuş token reddediliyor
- [ ] Token payload'ı kullanıcı bilgilerini içeriyor

---

### T022: Auth Middleware

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

HTTP middleware olarak JWT doğrulama. Authorization header'dan veya cookie'den token okuma. Context'e kullanıcı bilgilerini ekleme.

#### Files to Touch

- `internal/auth/middleware.go` (new)
- `internal/auth/middleware_test.go` (new)

#### Dependencies

- T021, T016 (middleware zinciri)

#### Success Criteria

- [ ] Bearer token'dan kullanıcı doğrulanıyor
- [ ] Geçersiz token'da 401 dönüyor
- [ ] Kullanıcı bilgileri context'e ekleniyor
- [ ] Public endpoint'ler auth'suz erişilebilir

---

### T023: RBAC Yetkilendirme

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

5 kademeli rol sistemi implementasyonu. Namespace bazlı yetkilendirme (örn: "Ali, @frontend-team/* npm paketleri için Publisher"). Middleware olarak route bazlı yetki kontrolü.

#### Technical Details

```go
type Role string
const (
    RoleSuperAdmin   Role = "super_admin"
    RoleRegistryAdmin Role = "registry_admin"
    RolePublisher    Role = "publisher"
    RoleConsumer     Role = "consumer"
    RoleViewer       Role = "viewer"
)

// Namespace scope: registry_type + package_pattern
type RoleAssignment struct {
    UserID       int
    Role         Role
    RegistryType string   // "*" for all
    Namespace    string   // glob pattern, e.g., "@frontend-team/*"
}
```

#### Files to Touch

- `internal/auth/rbac.go` (new)
- `internal/auth/rbac_test.go` (new)
- `internal/auth/middleware_authz.go` (new)

#### Dependencies

- T022, T011 (DB şeması — roles/user_roles tabloları)

#### Success Criteria

- [ ] Roller doğru hiyerarşide çalışıyor
- [ ] Namespace bazlı yetki kontrolü çalışıyor
- [ ] Yetkisiz erişimde 403 dönüyor
- [ ] Super Admin tüm kaynaklara erişebiliyor
- [ ] Glob pattern matching çalışıyor

---

### T024: API Token Yönetimi

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

CI/CD ve otomasyon için uzun ömürlü API token'ları. Token oluşturma, listeleme, silme, son kullanma tarihi.

#### Files to Touch

- `internal/auth/apitoken.go` (new)
- `internal/auth/apitoken_test.go` (new)

#### Dependencies

- T020, T021

#### Success Criteria

- [ ] API token oluşturma çalışıyor
- [ ] Token hash'lenerek saklanıyor (plaintext yok)
- [ ] Süresi dolan token reddediliyor
- [ ] Token listesinde hash gösterilmiyor (prefix only)

---

### T025: Bootstrap ve Login API

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

İlk çalıştırmada admin hesabı oluşturma (bootstrap), login/logout API endpoint'leri, token yenileme.

#### Technical Details

```
POST /api/v1/auth/login      — { username, password } → { token, expires_at }
POST /api/v1/auth/logout     — token invalidation
POST /api/v1/auth/refresh    — token yenileme
POST /api/v1/auth/bootstrap  — ilk admin oluşturma (yalnızca boş DB'de)
```

#### Files to Touch

- `internal/server/handlers/auth.go` (new)
- `internal/auth/bootstrap.go` (new)

#### Dependencies

- T020, T021, T022, T015

#### Success Criteria

- [ ] Login başarılı JWT dönüyor
- [ ] İlk çalıştırmada admin hesabı oluşturuluyor
- [ ] Bootstrap yalnızca boş DB'de çalışıyor
- [ ] Logout token'ı invalidate ediyor

## Risk Assessment

- **Orta Risk:** RBAC namespace pattern matching karmaşıklığı
- **Çözüm:** filepath.Match veya custom glob matcher kullanılacak
- **Dikkat:** JWT secret'ı config'de olmalı, varsayılan random üretilmeli
