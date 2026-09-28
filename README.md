# api-kms

Go REST API untuk KMS (knowledge management system). Punya akunnya sendiri (email + kata sandi) dan
menerbitkan JWT-nya sendiri; login Google tersedia sebagai alternatif.

Berawal dari salinan `lms-api` dengan domain LMS dibuang — yang tersisa
auth/RBAC/user plus infrastruktur operasional (notifikasi, activity log, audit
trail, job log, queue, scheduler, m2m client, lookup). Modul domain KMS belum ada.

Arsitektur, pola error handling, dan catatan skema ada di [CLAUDE.md](CLAUDE.md).

## Menjalankan lokal

```bash
cp .env.example .env          # isi DB_*, JWT_*
gartisan migrate              # buat skemanya
go run main.go                # http://localhost:8000
```

Perintah `gartisan` lain yang berguna:

```bash
gartisan migrate:status               # versi sekarang + status dirty
gartisan migrate:up force <version>   # tandai versi sebagai sudah diterapkan
gartisan migrate:down [n]
gartisan migrate:drop                 # hapus semua tabel, minta konfirmasi
gartisan db:seed                      # sama dengan go run ./cmd/seed
```

`gartisan` hanya untuk lokal — image produksi tidak memuatnya, dan memakai CLI
`golang-migrate` langsung. Keduanya membaca sumber migration dan baris
`schema_migrations` yang sama, jadi saling melihat keadaan yang ditinggalkan.

Ambil token tanpa lewat browser (hanya jalan saat `APP_ENV=local`):

```
GET http://localhost:8000/api/auth/dev-login?email=user@example.com
```

## Mengisi database

```bash
go run ./cmd/seed
```

Menulis permission, role beserta hibahnya ke Super-Admin, dan akun QA fixture
(`qa.super-admin@kms.test` / `qa.admin@kms.test`, kata sandi `password`).
Aman dijalankan ulang — semuanya upsert pada kunci alaminya. Sebelum dipakai di
produksi, buang `&User{}` dari registry di `database/seeders/seeder.go`.

### Sync permission

`enums/permission.go` adalah sumber kebenaran permission, dan tabel
`permissions` dibuat sama persis dengannya: baris baru ditambahkan, deskripsi
diperbarui, dan baris yang sudah tidak ada di enum **dihapus** — hibahnya ikut
lewat `ON DELETE CASCADE`.

```bash
go run ./cmd/sync-permission     # di image produksi: ./sync-permission
```

`cmd/seed` melakukan hal yang sama sebagai bagian dari seluruh data acuan.

Penghapusannya yang penting. `RolePermission` menghibahkan apa pun yang ada **di
tabel**, bukan apa yang ada di enum — jadi permission yang dihapus dari kode tapi
tertinggal di DB akan terus aktif selamanya, tanpa jejak di diff mana pun.
Jalankan setiap kali menambah atau menghapus permission.
