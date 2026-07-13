# GEMINI.md — Antigravity-specific overrides

> File ini melengkapi `AGENTS.md` (aturan utama, lintas-tool). Taruh `GEMINI.md` dan `AGENTS.md` di root project yang sama. Antigravity membaca keduanya; kalau ada aturan yang bentrok, `GEMINI.md` menang untuk perilaku spesifik Antigravity.

## Mode Kerja di Antigravity

- **Tentukan dulu domain task ini** (backend Go di root, atau frontend React di `web/`) dari konteks prompt sebelum membaca dokumen — lihat `AGENTS.md` untuk daftar dokumen per domain.
- **Backend:** mulai sesi dengan membaca `docs/12-engineering-summary.md`, lalu `docs/08-development-roadmap.md` untuk tahu sprint mana yang sedang berjalan.
- **Frontend:** mulai sesi dengan membaca `docs/14-frontend-overview.md`, lalu `docs/17-frontend-folder-structure.md` untuk struktur `web/` yang berlaku.
- Sebelum mengedit/generate kode, tampilkan dulu rencana singkat (file apa yang akan dibuat/diubah, sesuai layer di `05-folder-structure.md` untuk backend atau `17-frontend-folder-structure.md` untuk frontend) dan tunggu konfirmasi jika sedang dalam mode yang meminta approval per langkah.
- Gunakan tool "run tests" bawaan Antigravity setelah setiap perubahan pada `service/`/`repository/` (backend) atau `hooks/`/`components/` (frontend) — jangan tandai task selesai sebelum test lokal hijau.
- Kalau Antigravity mendeteksi kebutuhan menjalankan migration, jalankan lewat `scripts/migrate.sh` (lihat `09-deployment.md`), jangan menulis SQL manual langsung ke database yang jalan.
- Kalau agent perlu tipe request/response frontend dan OpenAPI spec backend sudah tersedia, jalankan codegen (`openapi-typescript`) alih-alih menulis tipe manual di `web/src/types/`.

## Skill Project

Kalau kamu (developer) menambahkan skill kustom untuk Antigravity (misalnya generator boilerplate modul baru mengikuti struktur `05-folder-structure.md`), taruh di:

```
.agents/skills/
```

bukan `.gemini/skills/` (path lama, sudah deprecated di versi Antigravity terbaru).

## Precedence

1. System rules Antigravity (bawaan, tidak bisa diubah)
2. `AGENTS.md` (aturan utama proyek — lihat file itu untuk aturan arsitektur & workflow)
3. `GEMINI.md` (file ini — override perilaku spesifik Antigravity)
4. Instruksi ad-hoc di chat (berlaku untuk sesi itu saja, tidak menimpa dua file di atas kecuali user eksplisit bilang "ubah aturan project")

## Catatan

Dokumentasi lengkap proyek ada di folder `docs/` — `00` sampai `13` untuk backend, `14` sampai `17` untuk frontend (penomoran flat, tidak dipisah subfolder). Jangan menebak keputusan desain yang sudah ada di sana — cari dulu di dokumen sebelum mengasumsikan.