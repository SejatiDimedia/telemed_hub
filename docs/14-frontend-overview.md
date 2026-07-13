# Frontend 00 — Overview

## 1. Relasi dengan Dokumentasi Backend

Dokumen ini adalah kelanjutan dari dokumentasi backend (`docs/backend/00`–`12`) — bukan proyek terpisah. Semua requirement, role, dan kontrak API sudah didefinisikan di sana. Frontend ini adalah **konsumen** dari API yang dirancang di `07-api-design.md`.

| Rujukan Backend | Dipakai Untuk |
|---|---|
| `01-product-requirements.md` | Role, persona, functional requirement yang harus didukung UI |
| `07-api-design.md` | Kontrak setiap request/response yang dikonsumsi frontend |
| `06-database-design.md` | Referensi field/relasi data saat merancang form & tampilan |

## 2. Scope Aplikasi

**Asumsi kerja (state eksplisit — konfirmasi/koreksi kalau perlu):**

Frontend MVP adalah **satu aplikasi web React**, bukan beberapa aplikasi terpisah:

- **Portal Patient & Doctor** — area utama, mayoritas fitur MVP
- **Area Admin** — route terproteksi di aplikasi yang sama (bukan app terpisah), untuk menjaga kompleksitas monorepo tetap rendah di tahap awal
- **Pharmacy Staff** — juga route terproteksi di aplikasi yang sama, mengingat scope-nya kecil (update status order)

Kalau di masa depan admin/pharmacy panel butuh berkembang jadi produk sendiri dengan tim terpisah, itu bisa di-extract ke `apps/admin` — pola ini konsisten dengan filosofi "Modular Monolith dulu, extract kalau perlu" yang sudah dipakai di backend.

## 3. Target Pengguna (mengacu ke persona backend)

| Role | Kebutuhan Utama di UI |
|---|---|
| Patient (Rina) | Booking dokter, lihat resep, pesan obat, top-up wallet, riwayat medis, chat AI triage |
| Doctor (Dr. Amir) | Kelola availability, lihat jadwal, jalankan konsultasi, tulis catatan & resep |
| Pharmacy Staff (Sinta) | Lihat daftar order, update status fulfillment |
| Admin (Bayu) | Verifikasi dokter, kelola user, lihat audit log |

## 4. Prinsip Desain Frontend

Sejalan dengan prinsip backend (`03-system-architecture.md`), frontend juga mengikuti:

- **Feature-based, bukan type-based** — kode dikelompokkan per domain bisnis (appointment, wallet, dll), bukan per jenis file (semua hooks di satu folder, semua komponen di folder lain). Ini konsisten dengan cara backend memisahkan `internal/<module>/`.
- **Reusable component system** — komponen UI generik (Button, Input, Modal) terpisah dari komponen spesifik-domain (AppointmentCard, WalletBalance).
- **Type-safety end-to-end** — TypeScript ketat, idealnya tipe request/response frontend disinkronkan dengan kontrak `07-api-design.md` (lihat catatan OpenAPI codegen di `01-frontend-tech-stack.md`).
- **Server state ≠ client state** — data dari API (appointments, wallet balance) dikelola terpisah dari state UI lokal (form input, modal terbuka/tertutup). Ini mencegah bug klasik "data stale" dan "sinkronisasi manual yang error-prone".

## 5. Out of Scope (MVP Frontend)

- Video/chat real-time (menunggu backend WebRTC/WebSocket — lihat `11-future-roadmap.md` backend)
- Native mobile app (React Native — kandidat masa depan terpisah)
- Multi-bahasa (i18n) — bisa ditambahkan belakangan, arsitektur akan disiapkan agar tidak menyulitkan penambahan i18n nanti
- Dark mode — nice-to-have, tidak diprioritaskan di MVP tapi Tailwind config akan disiapkan agar tidak menyulitkan penambahan nanti

---

**Dokumen berikutnya:** `01-frontend-tech-stack.md` — pilihan teknologi dan alasannya.
