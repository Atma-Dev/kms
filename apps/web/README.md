# web-kms

Nuxt 4 SPA (`ssr: false`) untuk KMS (knowledge management system). API-nya di [`apps/api`](../api).

Konvensi halaman, pola pagination, dan catatan UI ada di [CLAUDE.md](CLAUDE.md).

## Menjalankan lokal

```bash
npm install
cp .env.example .env     # isi NUXT_PUBLIC_API_BASE
npm run dev              # http://localhost:3000
```

`NUXT_PUBLIC_BYPASS_LOGIN=true` + `NUXT_PUBLIC_BYPASS_LOGIN_EMAIL=<email>` melewati
OAuth dan langsung memakai `GET /api/auth/dev-login` di API (hanya jalan saat
API-nya `APP_ENV=local`).

```bash
npm run build     # build produksi
npm run preview   # pratinjau hasil build
npm test          # vitest
```
