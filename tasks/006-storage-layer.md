# Feature 006: Storage Layer

**Feature ID:** F006
**Feature Name:** Storage Layer
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Paket dosyalarının depolanması için soyutlama katmanı. v1.0'da filesystem backend; S3/MinIO v1.1'de eklenecek. Her ekosistem kendi dizin yapısını kullanır (Docker OCI layout, npm tarballs, PyPI wheels, vb.).

## Goals

- Backend-agnostik storage interface
- Filesystem implementasyonu
- Ekosistem bazlı dizin yapısı
- Disk kullanımı takibi ve garbage collection desteği

## Success Criteria

- [ ] Tüm task'lar tamamlandı (T026-T029)
- [ ] Dosya yazma/okuma/silme çalışıyor
- [ ] Ekosistem bazlı dizin yapısı doğru
- [ ] Disk kullanımı raporlanabiliyor

## Tasks

### T026: Storage Interface Tanımı

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 day

#### Description

Storage soyutlama katmanı interface'i. Put, Get, Delete, Exists, List, Stat operasyonları.

#### Technical Details

```go
type Storage interface {
    Put(ctx context.Context, path string, reader io.Reader) error
    Get(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    List(ctx context.Context, prefix string) ([]FileInfo, error)
    Stat(ctx context.Context, path string) (*FileInfo, error)
    Usage(ctx context.Context) (*UsageInfo, error)
}
```

#### Files to Touch

- `internal/storage/storage.go` (new)
- `internal/storage/types.go` (new)

#### Dependencies

- T001

#### Success Criteria

- [ ] Interface tanımlı ve belgelenmiş
- [ ] FileInfo ve UsageInfo tipleri tanımlı
- [ ] Hata tipleri tanımlı (ErrNotFound, ErrAlreadyExists)

---

### T027: Filesystem Backend

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description

Yerel dosya sistemi üzerinde Storage interface implementasyonu. Atomic write (temp file + rename), dizin oluşturma, dosya kilitleme.

#### Technical Details

- Atomic write: temp dosyaya yaz → rename (veri bozulması önleme)
- Dizin yapısı: `{base}/{ecosystem}/{subpath}`
- File permissions: 0644 dosyalar, 0755 dizinler

#### Files to Touch

- `internal/storage/filesystem.go` (new)
- `internal/storage/filesystem_test.go` (new)

#### Dependencies

- T026, T006 (config — storage path)

#### Success Criteria

- [ ] Dosya yazma/okuma/silme çalışıyor
- [ ] Atomic write ile veri tutarlılığı sağlanıyor
- [ ] Olmayan dizinler otomatik oluşturuluyor
- [ ] Concurrent erişimde veri bozulması yok
- [ ] Unit test'ler yazıldı

---

### T028: Ekosistem Dizin Yapısı

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Her ekosistem için standart dizin yapısını tanımlayan path resolver. Docker OCI layout, npm tarball yapısı, vb.

#### Technical Details

```
data/docker/registry/     — OCI layout
data/docker/blobs/        — blob storage
data/npm/packages/        — metadata
data/npm/tarballs/        — tarball dosyaları
data/pypi/packages/       — metadata
data/pypi/wheels/         — wheel/sdist dosyaları
data/gomod/modules/       — module zip'leri
data/gomod/cache/         — mod file cache
data/cargo/crates/        — crate dosyaları
data/cargo/index/         — sparse index
```

#### Files to Touch

- `internal/storage/paths.go` (new)
- `internal/storage/paths_test.go` (new)

#### Dependencies

- T027

#### Success Criteria

- [ ] Her ekosistem için path resolver tanımlı
- [ ] Path'ler güvenli (path traversal önleme)
- [ ] Ekosistem bazlı disk kullanımı raporlanabiliyor

---

### T029: Garbage Collection

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 1.5 days

#### Description

Kullanılmayan paket versiyonlarını temizleme. Soft delete (işaretle) → hard delete (sil) iki aşamalı GC. DB referansı olmayan orphan dosyaları tespit etme.

#### Files to Touch

- `internal/storage/gc.go` (new)
- `internal/storage/gc_test.go` (new)

#### Dependencies

- T027, T013 (DB query'leri)

#### Success Criteria

- [ ] Orphan dosyalar tespit ediliyor
- [ ] Soft delete/hard delete iki aşamalı çalışıyor
- [ ] GC çalıştırıldığında disk alanı geri kazanılıyor
- [ ] Aktif dosyalar silinmiyor (safety check)
- [ ] GC raporu döndürülüyor (freed bytes, deleted files)

## Performance Targets

- File read latency: < 5ms (local SSD)
- Write throughput: limited by disk I/O
- GC: incremental, configurable batch size

## Risk Assessment

- **Düşük Risk:** Filesystem operasyonları iyi bilinen
- **Dikkat:** Atomic write POSIX rename semantiği gerektirir (cross-filesystem rename sorunlu)
- **Dikkat:** Path traversal attack'larına karşı input sanitization zorunlu
