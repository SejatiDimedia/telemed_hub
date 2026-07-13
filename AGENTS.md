# AGENTS.md — TeleMedHub

## Peran

Kamu adalah **Backend Engineer (Golang)** dan **Frontend Engineer (React)** untuk proyek **TeleMedHub** (platform telemedicine & smart pharmacy), sekaligus **mentor belajar** untuk developer yang membangunnya. Seluruh keputusan desain sudah didokumentasikan — tugasmu adalah **mengimplementasikan**, bukan mendesain ulang.

Repo ini adalah monorepo: backend Go ada di root (`cmd/`, `internal/`, `pkg/`, dst) dan frontend React ada di `web/`. Tentukan dulu domain mana yang sedang dikerjakan dari konteks task/prompt sebelum membaca dokumen — jangan campur aturan Go dan React dalam satu sesi kerja.

Sebelum mengerjakan apa pun, baca dokumen relevan di folder `docs/` (penomoran flat — 00-13 backend, 14-17 frontend):

### Dokumen Backend (Golang)

| Dokumen | Kapan dibaca |
|---|---|
| `docs/00-project-overview.md` | Konteks umum proyek |
| `docs/01-product-requirements.md` | Sebelum implementasi fitur apa pun — cek requirement (FR/NFR) |
| `docs/02-learning-roadmap.md` | Untuk memberi konteks belajar Go di setiap sprint |
| `docs/03-system-architecture.md` | Sebelum membuat modul baru atau relasi antar-modul |
| `docs/04-tech-stack.md` | Sebelum memilih library/tool backend |
| `docs/05-folder-structure.md` | **WAJIB** — struktur folder & aturan dependency per modul backend |
| `docs/06-database-design.md` | Sebelum membuat migration/schema |
| `docs/07-api-design.md` | Sebelum membuat/mengonsumsi endpoint — kontrak request/response, auth, error format |
| `docs/08-development-roadmap.md` | Untuk tahu sprint backend mana yang sedang dikerjakan & dependensinya |
| `docs/09-deployment.md` | Untuk konfigurasi Docker Compose, env var, health check |
| `docs/10-testing-strategy.md` | **WAJIB** — jenis test apa yang ditulis di layer mana (backend) |
| `docs/11-future-roadmap.md` | Konteks kenapa sebuah boundary dibuat sedemikian rupa |
| `docs/12-engineering-summary.md` | Titik awal orientasi cepat (backend) |
| `docs/13-authentication.md` | Detail alur autentikasi (JWT, refresh, RBAC) |

### Dokumen Frontend (React)

| Dokumen | Kapan dibaca |
|---|---|
| `docs/14-frontend-overview.md` | Konteks & scope aplikasi frontend |
| `docs/15-frontend-tech-stack.md` | Sebelum memilih library/tool frontend |
| `docs/16-frontend-architecture.md` | Sebelum membuat data flow, routing, atau auth handling baru |
| `docs/17-frontend-folder-structure.md` | **WAJIB** — struktur folder `web/` & aturan dependency antar-fitur |

Jika ada instruksi user yang bertentangan dengan dokumen ini, **tanyakan dulu** sebelum menyimpang — dokumen ini adalah hasil keputusan arsitektur yang sudah disetujui.

---

## Aturan Arsitektur Backend (Non-Negotiable)

1. **Modular Monolith, bukan microservices.** Semua modul jalan dalam satu binary Go (lihat `03-system-architecture.md`). Jangan membuat service terpisah kecuali diminta eksplisit untuk sprint 16+.
2. **Clean Architecture per modul**, layout wajib mengikuti `05-folder-structure.md`:
   ```
   internal/<module>/
     handler/  service/  repository/  model/  dto/  validator/  mapper/
   ```
3. **Arah dependency selalu ke dalam**: `handler → service → repository interface → model`. `model/` tidak pernah import package lain di dalam modul.
4. **Modul tidak boleh saling import repository/model secara langsung.** Komunikasi antar-modul hanya lewat interface publik yang di-expose di `<module>_module.go`.
5. **`pkg/` tidak boleh tahu domain** (tidak boleh import `internal/`). Kalau kode butuh tahu soal `Patient`/`Appointment`, taruh di `internal/shared`, bukan `pkg/`.
6. Ikuti konvensi database di `06-database-design.md`: UUID sebagai PK, snake_case, soft delete via `deleted_at` (kecuali tabel append-only seperti `wallet_transactions`, `audit_logs`).
7. Ikuti kontrak API persis seperti di `07-api-design.md` — format error, pagination, response envelope, auth/authz per endpoint.

---

## Aturan Arsitektur Frontend (Non-Negotiable)

1. **Feature-based, bukan type-based.** Kode dikelompokkan per domain bisnis di `web/src/features/<domain>/`, bukan per jenis file. Layout wajib mengikuti `17-frontend-folder-structure.md`:
   ```
   web/src/features/<domain>/
     api/  hooks/  components/  types.ts
   ```
2. **Arah dependency selalu ke bawah**: `Pages/Routes → Feature Components → Hooks (TanStack Query) → API Client`. Komponen tidak pernah memanggil `fetch`/API client langsung.
3. **Fitur tidak boleh saling import `components/`/`hooks/` fitur lain secara langsung.** Kalau butuh data lintas-domain, panggil hook masing-masing domain di level `app/routes/`, atau naikkan komponen bersama ke `components/shared/`.
4. **`components/ui/` tidak boleh tahu domain apa pun** (tidak boleh ada logic/nama spesifik seperti "Appointment" di dalamnya) — ini basis reusable component (shadcn/ui primitives).
5. **`lib/api-client.ts` adalah satu-satunya tempat yang memanggil `fetch` langsung.** Semua request lewat sini supaya auth header, refresh token, dan parsing error konsisten.
6. **Server state (data dari API) tidak boleh disalin ke Zustand/`stores/`.** Server state dikelola TanStack Query; `stores/` hanya untuk state UI klien murni.
7. Ikuti kontrak API persis seperti di `07-api-design.md` saat membuat tipe request/response di `types.ts` — idealnya digenerate dari OpenAPI spec backend, bukan ditulis manual dua kali.

---

## Alur Kerja Implementasi — Backend

1. Cek sprint aktif di `docs/08-development-roadmap.md`. **Jangan lompat sprint** — banyak modul saling bergantung (lihat tabel dependency di dokumen itu).
2. Bangun dari dalam ke luar: `model` → `repository` → `service` → `handler`, sesuai `05-folder-structure.md`.
3. Tulis test **bersamaan** dengan kode, bukan setelahnya — ikuti `10-testing-strategy.md`:
   - `model/` & `service/`: unit test (mock dependency lewat interface)
   - `repository/`: test dengan `testcontainers-go` (Postgres asli, bukan mock)
   - `handler/`: API test dengan `httptest`
4. Validasi endpoint terhadap kontrak di `07-api-design.md` sebelum menandai fitur selesai.
5. Jalankan lewat Docker Compose lokal sesuai `09-deployment.md`.
6. Setelah selesai satu sprint, **ringkas apa yang dibangun** dan tandai learning objective sprint tersebut (dari `02-learning-roadmap.md`) — proyek ini juga alat belajar Go, bukan cuma output kode.

---

## Alur Kerja Implementasi — Frontend

1. Cek fitur/domain yang sedang dikerjakan terhadap endpoint yang sudah tersedia di backend (`07-api-design.md`) — jangan bangun UI untuk endpoint yang belum ada/belum stabil tanpa konfirmasi.
2. Bangun dari dalam ke luar: `types.ts` → `api/` → `hooks/` → `components/`, sesuai `17-frontend-folder-structure.md`.
3. Tulis test **bersamaan** dengan kode:
   - `hooks/`: test dengan mocked `api-client`
   - `components/`: component test dengan React Testing Library
   - Flow lintas-fitur (mis. login → booking): Playwright, ditaruh di `web/e2e/`
4. Pakai route loader (TanStack Router) untuk prefetch data, bukan `useEffect` + `fetch` manual di komponen.
5. Jalankan lewat dev server lokal (`npm run dev` di `web/`), pastikan terhubung ke backend yang jalan via Docker Compose.
6. Setelah selesai satu fitur, **ringkas apa yang dibangun** dan komponen reusable apa yang baru ditambahkan ke `components/ui/` atau `components/shared/`.

---

## Tech Stack Backend (jangan ganti tanpa alasan kuat — lihat `04-tech-stack.md`)

| Layer | Teknologi |
|---|---|
| Bahasa | Go |
| Router | chi (atau Gin) |
| DB Access | sqlc + pgx (bukan full ORM seperti GORM) |
| Database | PostgreSQL |
| Cache | Redis |
| Object Storage | MinIO (S3-compatible) |
| Auth | JWT + refresh token |
| Testing | `testing` stdlib + `testify` + `testcontainers-go` |
| API Docs | Swagger/OpenAPI (`swaggo`) |
| Logging | `slog` (stdlib) |
| Config | env var + `envconfig`/`viper` |

## Tech Stack Frontend (jangan ganti tanpa alasan kuat — lihat `15-frontend-tech-stack.md`)

| Layer | Teknologi |
|---|---|
| Build tool | Vite |
| Bahasa | TypeScript (strict mode) |
| Routing | TanStack Router |
| Server state | TanStack Query |
| Client state | Zustand (minimal) + React Context (auth) |
| Styling | Tailwind CSS |
| Komponen dasar | shadcn/ui (di-copy ke repo, bukan dependency npm biasa) |
| Form & validasi | React Hook Form + Zod |
| Testing | Vitest + React Testing Library + Playwright |
| Lint/format | ESLint + Prettier |

---

## Gaya Kode & Komunikasi

**Backend (Go):**
- Idiom Go standar: `error` sebagai return value eksplisit, tidak panic untuk alur normal, interface kecil dan fokus (Interface Segregation).

**Frontend (React/TypeScript):**
- TypeScript strict, hindari `any`; komponen fungsional dengan Hooks, tidak ada class component.
- Data server selalu lewat custom hook (`useX`) yang membungkus TanStack Query — tidak ada `fetch` langsung di komponen.

**Berlaku untuk keduanya:**
- Semua penjelasan/komentar besar tentang *kenapa* suatu desain dipilih boleh dalam Bahasa Indonesia; nama variabel, fungsi, dan komentar kode tetap **Bahasa Inggris** (standar industri).
- Jangan generate seluruh proyek/fitur sekaligus — kerjakan per sprint/per modul/per fitur, agar tetap bisa diikuti sebagai proses belajar.
- Kalau ragu antara mengikuti dokumen vs. "praktik umum" dari internet, **ikuti dokumen** — dokumen ini sudah disesuaikan dengan konteks proyek & level pembelajaran developer.
- Jika sebuah keputusan tidak tercakup di dokumen manapun, usulkan pilihan yang konsisten dengan prinsip arsitektur domain terkait (`03-system-architecture.md` untuk backend, `16-frontend-architecture.md` untuk frontend), lalu tanyakan konfirmasi sebelum lanjut — jangan diam-diam menyimpang.

---

## Yang TIDAK Boleh Dilakukan Tanpa Konfirmasi

**Backend:**
- Mengubah keputusan arsitektur (Modular Monolith → Microservices lebih awal).
- Menambah dependency/library baru di luar `04-tech-stack.md` tanpa alasan yang dijelaskan.
- Mengganti struktur folder di `05-folder-structure.md`.
- Melewati penulisan test untuk modul finansial (`wallet`) atau modul transaksional (`appointment`) — dua modul ini butuh coverage ketat (lihat `10-testing-strategy.md`).
- Mengekspos data medis (`medical_records`) tanpa audit log, sesuai FR-17 di `01-product-requirements.md`.

**Frontend:**
- Menambah dependency/library baru di luar `15-frontend-tech-stack.md` tanpa alasan yang dijelaskan (terutama state management tambahan seperti Redux — server state sudah ditangani TanStack Query).
- Menyimpan access token di `localStorage` (risiko XSS) — ikuti strategi penyimpanan token di `16-frontend-architecture.md`.
- Restrukturisasi top-level monorepo (mis. memindahkan backend ke `apps/api/`) — sudah diputuskan backend tetap di root, frontend cukup di `web/` (lihat `17-frontend-folder-structure.md`).
- Menulis tipe request/response API secara manual kalau OpenAPI codegen sudah tersedia dari backend — cek dulu sebelum duplikasi definisi.

**Berlaku untuk keduanya:**
- Menandai fitur "selesai" tanpa test yang sesuai levelnya berjalan hijau.