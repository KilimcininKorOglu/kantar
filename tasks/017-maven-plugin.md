# Feature 017: Maven/Gradle Plugin

**Feature ID:** F017
**Feature Name:** Maven/Gradle Plugin
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Maven Central Repository uyumlu plugin. Maven ve Gradle build tool'ları ile uyumlu artifact depolama ve serve etme. GroupId/ArtifactId/Version (GAV) koordinat sistemi.

## Goals

- Maven repository layout uyumluluğu
- `mvn` ve `gradle` komutlarıyla uyumluluk
- Maven Central'dan sync
- POM dosyası yönetimi

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T076-T079)
- [ ] Maven dependency resolution çalışıyor
- [ ] Gradle dependency resolution çalışıyor
- [ ] Artifact deploy çalışıyor

## Tasks

### T076: Maven Repository Layout

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Maven repository dizin yapısı ve metadata endpoint'leri. GAV koordinat sistemi ile artifact serve etme.

#### Technical Details

```
GET /maven/{groupId-as-path}/{artifactId}/{version}/{artifactId}-{version}.{ext}
GET /maven/{groupId-as-path}/{artifactId}/{version}/{artifactId}-{version}.pom
GET /maven/{groupId-as-path}/{artifactId}/maven-metadata.xml
GET /maven/{groupId-as-path}/{artifactId}/{version}/maven-metadata.xml
```

GroupId path: `com.example` → `com/example/`
Checksum files: `.sha1`, `.sha256`, `.md5`

#### Files to Touch

- `internal/plugins/maven/plugin.go` (new)
- `internal/plugins/maven/routes.go` (new)
- `internal/plugins/maven/layout.go` (new)

#### Dependencies

- T033, T019

#### Success Criteria

- [ ] GAV koordinat sistemi doğru çözülüyor
- [ ] maven-metadata.xml serve ediliyor
- [ ] Checksum dosyaları serve ediliyor
- [ ] POM dosyası serve ediliyor

---

### T077: Artifact Deploy ve Storage

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Artifact upload (PUT), snapshot vs release ayrımı, POM parsing.

#### Technical Details

```
PUT /maven/{groupId-as-path}/{artifactId}/{version}/{filename}  — artifact deploy
```

Snapshot versiyonlar: `1.0-SNAPSHOT` → timestamped versiyonlar
Release versiyonlar: immutable, tekrar deploy edilemez

#### Files to Touch

- `internal/plugins/maven/deploy.go` (new)
- `internal/plugins/maven/pom.go` (new)
- `internal/plugins/maven/deploy_test.go` (new)

#### Dependencies

- T076, T027, T022

#### Success Criteria

- [ ] Artifact deploy çalışıyor
- [ ] Snapshot vs Release ayrımı çalışıyor
- [ ] Checksum dosyaları otomatik üretiliyor
- [ ] maven-metadata.xml otomatik güncelleniyor

---

### T078: Maven Central Upstream Sync

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Maven Central (repo1.maven.org) üzerinden artifact sync.

#### Files to Touch

- `internal/plugins/maven/upstream.go` (new)

#### Dependencies

- T076, T040

#### Success Criteria

- [ ] Maven Central'dan artifact çekiliyor
- [ ] POM bağımlılık ağacı çözülebiliyor
- [ ] Checksum doğrulaması yapılıyor

---

### T079: Maven Plugin Config ve Entegrasyon

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 day

#### Description

Maven plugin tam entegrasyonu, Gradle uyumluluğu doğrulaması.

#### Files to Touch

- `internal/plugins/maven/config.go` (new)
- `internal/plugins/maven/plugin.go` (update)

#### Dependencies

- T076-T078

#### Success Criteria

- [ ] RegistryPlugin interface tam implement edildi
- [ ] Maven ve Gradle ile end-to-end çalışıyor

## Risk Assessment

- **Orta Risk:** Maven repository layout karmaşık (snapshot timestamping, metadata merge)
- **Dikkat:** Gradle ve Maven farklı metadata çözümleme stratejileri kullanabilir
