# Frontend 01 — Tech Stack

Format sama seperti `04-tech-stack.md` di backend: setiap teknologi dijelaskan alasan, alternatif, pro/kontra.

---

## Build Tool: Vite

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Dev server cepat (native ESM, HMR instan), konfigurasi minimal, standar de-facto untuk React modern |
| Alternatif | Create React App (sudah deprecated/tidak direkomendasikan lagi), Next.js (bagus tapi bawa kompleksitas SSR/routing yang tidak dibutuhkan — TeleMedHub adalah SPA di belakang API, bukan aplikasi yang butuh SSR/SEO) |
| Kontra | Tidak ada SSR built-in (memang tidak dibutuhkan untuk aplikasi internal seperti ini) |

## Bahasa: TypeScript (strict mode)

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Type-safety mencegah bug integrasi dengan API backend; kontrak di `07-api-design.md` bisa direpresentasikan sebagai TypeScript types/interfaces |
| Konfigurasi | `strict: true`, `noUncheckedIndexedAccess: true` — ketat sejak awal, lebih murah daripada mengetatkan belakangan |

## Routing: TanStack Router

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Type-safe routing penuh (route params, search params, loader data semuanya ter-infer otomatis oleh TypeScript) — cocok dengan prinsip type-safety end-to-end. File-based routing opsional tapi direkomendasikan untuk konsistensi struktur folder |
| Alternatif | React Router (lebih populer, tapi type-safety untuk params/search params tidak sekuat TanStack Router), Next.js App Router (bawa SSR yang tidak dibutuhkan) |
| Kontra | Ekosistem/komunitas masih lebih kecil dibanding React Router; sedikit lebih curam untuk pemula |
| Fitur kunci yang dipakai | **Route loaders** untuk prefetch data sebelum render (dipadukan dengan TanStack Query), **protected route** pattern untuk RBAC (patient/doctor/admin) |

## Server State: TanStack Query

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Standar industri untuk mengelola data dari server: caching, refetch otomatis, invalidation, loading/error state — semua tanpa reinventing the wheel di setiap komponen |
| Alternatif | SWR (lebih ringan tapi fitur lebih sedikit), fetch manual + useState/useEffect (rawan bug, tidak scalable) |
| Prinsip pemakaian | Setiap panggilan ke endpoint `07-api-design.md` dibungkus custom hook (`useAppointments()`, `useWalletBalance()`), bukan dipanggil langsung di komponen — supaya caching key dan invalidation konsisten |

## Client State: Zustand (minimal) + React Context (untuk auth)

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Kebanyakan state di TeleMedHub adalah **server state** (dikelola TanStack Query) — client state murni (UI toggle, wizard step) jarang butuh solusi berat. Zustand dipakai hanya kalau ada state lintas-komponen yang genuinely tidak berasal dari server |
| Auth state | Context sederhana (`AuthProvider`) menyimpan current user + token, bukan Zustand — auth adalah cross-cutting concern yang sifatnya lebih dekat ke "identity provider" daripada UI state biasa |
| Anti-pattern yang dihindari | Menaruh data server (appointments, wallet) ke dalam Zustand/Redux — ini duplikasi tanggung jawab dengan TanStack Query dan sumber bug klasik data stale |

## Styling: Tailwind CSS

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Utility-first, tidak perlu context-switch ke file CSS terpisah, mudah menjaga konsistensi lewat `tailwind.config` (design tokens: warna, spacing, radius) |
| Alternatif | CSS Modules (lebih verbose), styled-components/Emotion (runtime cost, kurang cocok dipadukan dengan Tailwind) |
| Konvensi | Design tokens (warna brand, spacing scale) didefinisikan di `tailwind.config.ts`, **bukan** hardcode hex/px di komponen — supaya rebranding/theming di masa depan tidak perlu cari-ganti manual |

## Komponen Dasar: shadcn/ui (sebagai starting point, bukan dependency langsung)

| Aspek | Detail |
|---|---|
| Kenapa dipilih | shadcn/ui bukan library npm biasa — komponennya di-copy ke repo kamu sendiri (`components/ui/`), jadi kamu punya kontrol penuh dan tidak terikat versi eksternal. Cocok dengan prinsip "reusable components" yang kamu inginkan, karena kamu langsung memilikinya |
| Alternatif | Chakra UI / MUI (bawa desain sistem sendiri yang harus di-override), Radix UI langsung tanpa styling (lebih banyak kerja manual) | 
| Cara pakai | Dipakai sebagai **base primitives** (Button, Dialog, Select, dll) di `components/ui/`, lalu komponen domain-spesifik (`AppointmentCard`) dibangun di atasnya |

## Form & Validasi: React Hook Form + Zod

| Aspek | Detail |
|---|---|
| Kenapa dipilih | React Hook Form minim re-render (performant), Zod memberi validasi type-safe yang skemanya **bisa dipakai ulang** untuk memvalidasi response API juga |
| Keuntungan tambahan | Skema Zod untuk request body bisa dicocokkan langsung dengan validasi di `07-api-design.md` (mis. `password min 8 karakter`) — satu sumber kebenaran untuk validasi di sisi client |

## API Client Layer: fetch wrapper tipis + (opsional) OpenAPI codegen

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Backend sudah punya rencana menghasilkan OpenAPI spec lewat `swaggo` (lihat `04-tech-stack.md` backend). Kalau spec itu tersedia, tipe request/response frontend bisa **digenerate otomatis** (mis. pakai `openapi-typescript`) — menghindari drift antara frontend dan kontrak backend |
| Fallback (kalau spec belum siap) | Fetch wrapper tipis (`apiClient.ts`) dengan interceptor untuk attach JWT, handle refresh token, dan parse error envelope sesuai format di `07-api-design.md` |

## Testing: Vitest + React Testing Library + Playwright

| Aspek | Detail |
|---|---|
| Unit/component test | Vitest (selaras dengan Vite, cepat) + React Testing Library (test perilaku, bukan detail implementasi) |
| E2E test | Playwright — untuk flow kritikal (login → booking → bayar), mirror filosofi `10-testing-strategy.md` backend yang fokus ke flow bernilai tinggi, bukan coverage % semata |

## Linting & Formatting: ESLint + Prettier

| Aspek | Detail |
|---|---|
| Kenapa dipilih | Standar industri, terintegrasi baik dengan TypeScript & React | 
| Tambahan | `eslint-plugin-react-hooks` (wajib, mencegah bug dependency array), `eslint-plugin-jsx-a11y` (accessibility check dasar) |

## Ringkasan Stack

| Layer | Teknologi |
|---|---|
| Build tool | Vite |
| Bahasa | TypeScript (strict) |
| Routing | TanStack Router |
| Server state | TanStack Query |
| Client state | Zustand (minim) + Context (auth) |
| Styling | Tailwind CSS |
| Komponen dasar | shadcn/ui (Radix-based, di-copy ke repo) |
| Form & validasi | React Hook Form + Zod |
| API client | fetch wrapper + opsional OpenAPI codegen |
| Testing | Vitest + React Testing Library + Playwright |
| Lint/format | ESLint + Prettier |

---

**Dokumen berikutnya:** `02-frontend-architecture.md` — bagaimana semua ini disusun jadi satu arsitektur.
