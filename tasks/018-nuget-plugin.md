# Feature 018: NuGet Plugin

**Feature ID:** F018
**Feature Name:** NuGet Plugin
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 0.5 week
**Status:** NOT_STARTED

## Overview

NuGet V3 Service Index API uyumlu plugin. `dotnet nuget push`, `dotnet add package` komutlarıyla uyumlu. nuget.org upstream sync.

## Goals

- NuGet V3 API uyumluluk
- dotnet CLI ile uyumluluk
- nuget.org'dan sync

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T080-T082)
- [ ] `dotnet add package` çalışıyor
- [ ] `dotnet nuget push` çalışıyor
- [ ] Upstream sync çalışıyor

## Tasks

### T080: NuGet V3 Service Index ve Package Content

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

NuGet V3 API endpoint'leri: service index, package content, search, registration.

#### Technical Details

```
GET  /nuget/v3/index.json                          — service index
GET  /nuget/v3/registration/{id}/index.json         — package registration
GET  /nuget/v3/flatcontainer/{id}/index.json        — version list
GET  /nuget/v3/flatcontainer/{id}/{ver}/{id}.{ver}.nupkg — package download
GET  /nuget/v3/search?q=                            — search
PUT  /nuget/v3/package                              — push
```

#### Files to Touch

- `internal/plugins/nuget/plugin.go` (new)
- `internal/plugins/nuget/routes.go` (new)
- `internal/plugins/nuget/service.go` (new)
- `internal/plugins/nuget/content.go` (new)

#### Dependencies

- T033, T019

#### Success Criteria

- [ ] Service index doğru JSON döndürüyor
- [ ] Package download çalışıyor
- [ ] Search endpoint çalışıyor
- [ ] Registration page'ler çalışıyor

---

### T081: NuGet Push ve Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

NuGet package push (multipart) ve nuget.org upstream sync.

#### Files to Touch

- `internal/plugins/nuget/push.go` (new)
- `internal/plugins/nuget/upstream.go` (new)

#### Dependencies

- T080, T027, T040

#### Success Criteria

- [ ] `dotnet nuget push` çalışıyor
- [ ] Upstream sync çalışıyor
- [ ] .nupkg dosyaları doğru depolanıyor

---

### T082: NuGet Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

NuGet plugin tam entegrasyonu.

#### Files to Touch

- `internal/plugins/nuget/config.go` (new)
- `internal/plugins/nuget/plugin.go` (update)

#### Dependencies

- T080-T081

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] End-to-end dotnet add package ve push çalışıyor

## Risk Assessment

- **Orta Risk:** NuGet V3 API birden fazla service endpoint kullanıyor (service index pattern)
- **Çözüm:** nuget.org V3 index.json referans alınacak
