# KMS

Monorepo KMS (knowledge management system), berisi API Go dan web Nuxt. Dependensi dan deployment kedua aplikasi tetap terpisah.

```text
apps/api/       API Go (Gin + GORM/MySQL), migrasi database, seeder
apps/web/       Web Nuxt 4 SPA (tema Midone)
.gitlab-ci.yml  Deployment dev per aplikasi
compose.yaml    Container lokal API dan web
Makefile        Perintah bersama
```

Dokumentasi tiap aplikasi tetap di `apps/api/CLAUDE.md` dan `apps/web/CLAUDE.md`.

## Persiapan

Gunakan Go sesuai `apps/api/go.mod` (1.26), Node.js 24, npm, dan GNU Make. Docker Compose diperlukan hanya untuk menjalankan container.

```sh
cp apps/api/.env.example apps/api/.env
cp apps/web/.env.example apps/web/.env
make install
```

Isi konfigurasi MySQL, Redis, serta kredensial lain di `.env` masing-masing. Jangan commit `.env`.

API memakai port dari `APP_PORT` (default 8019). Web memakai port 3019 (`devServer.port` di `nuxt.config.ts`) dengan `NUXT_PUBLIC_API_BASE=http://localhost:8019`. CORS API hanya menerima `FRONTEND_URL`, jadi kalau port diganti, ganti ketiganya.

## Perintah

| Perintah | Fungsi |
| --- | --- |
| `make install` | Unduh dependensi Go dan install npm dari lockfile |
| `make dev` | Jalankan API dan web bersamaan; Ctrl+C untuk berhenti |
| `make dev-api` / `make dev-web` | Jalankan satu aplikasi |
| `make test` | Tes API, lalu web |
| `make build` | Build API ke `apps/api/tmp/kms-api` dan web ke `apps/web/.output` |
| `make compose-up` | Build dan jalankan container lokal |
| `make compose-down` | Hentikan container lokal |

Compose root untuk lokal, dengan MySQL dan Redis di host (`host.docker.internal`). Compose di masing-masing folder aplikasi untuk server.

## Git dan deployment

Riwayat repo `api-kms` dan `web-kms` diimpor dengan `git subtree` tanpa squash, jadi commit dan hash aslinya tetap ada di `git log`.

`.gitlab-ci.yml` men-deploy ke server dev saat push ke `develop` (atau `devops-cicd`), per aplikasi sesuai folder yang berubah. Server memakai satu checkout di `/deployment/kms`, lalu menjalankan compose dari `apps/api` atau `apps/web`. Nama project Compose dikunci (`name: api-kms` / `name: web-kms`), jadi container dan volume `storage` API tetap sama dengan sebelum monorepo. Deploy prod masih dikomentari sampai servernya siap.

### Pindah dari dua checkout lama

1. Push monorepo ke remote baru, dan pastikan runner `cid-stg` serta variabel `KEY_DEV_1` / `IP_DEV_1` tersedia untuk project itu.
2. Di server, clone ke `/deployment/kms` dan checkout `develop`.
3. Salin `.env`: `/deployment/api-kms/.env` → `/deployment/kms/apps/api/.env`, `/deployment/web-kms/.env` → `/deployment/kms/apps/web/.env`.
4. Matikan pipeline di repo lama, lalu jalankan pipeline monorepo. Volume `api-kms_storage` dipakai ulang karena nama project sama.
5. Setelah stabil, hapus `/deployment/api-kms` dan `/deployment/web-kms`, lalu archive repo lama.
