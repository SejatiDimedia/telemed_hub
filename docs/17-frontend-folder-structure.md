# Frontend 03 — Folder Structure

## 1. Struktur Monorepo (Top-Level)

Karena backend sudah diimplementasikan dan berjalan dengan struktur root-level (`cmd/`, `internal/`, `pkg/`, dst sesuai `05-folder-structure.md`), frontend **tidak memaksakan restrukturisasi `apps/api` + `apps/web`**. Backend tetap di root apa adanya; frontend ditambahkan sebagai satu folder baru (`web/`) sejajar dengannya:

```
telemed_hub/
├── cmd/                          # backend, tidak berubah
├── internal/                     # backend, tidak berubah
├── pkg/                          # backend, tidak berubah
├── configs/                      # backend, tidak berubah
├── migrations/                   # backend, tidak berubah
├── scripts/                      # backend, tidak berubah
├── deployments/                  # backend, tidak berubah (docke r-compose.yml dsb, cukup ditambah service web)
├── test/                         # backend, tidak berubah
├── web/                          # frontend React (dokumen ini)
│   ├── src/
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   └── package.json
├── docs/
│   ├── backend/                  # 00-12 (dokumentasi backend yang sudah ada)
│   └── frontend/                 # dokumen ini (00-03)
├── AGENTS.md
├── GEMINI.md
└── .antigravity/
    └── agents/
```

**Kenapa bukan `apps/api` + `apps/web`:** backend sudah jadi dan jalan — memindahkan seluruh strukturnya ke `apps/api/` akan menyentuh `docker-compose.yml`, `scripts/*.sh`, CI config, dan `Dockerfile` sekaligus, hanya demi kerapian, tanpa manfaat fungsional. Pola "backend di root + `web/` sebagai folder tambahan" adalah pola monorepo yang sama validnya, dan jauh lebih murah untuk proyek yang backend-nya sudah established.

**Kapan restrukturisasi `apps/*` baru masuk akal:** kalau nanti backend benar-benar dipecah jadi beberapa service (Sprint 16+, service extraction di `08-development-roadmap.md`) — di titik itu akan ada beberapa binary/service sekaligus, sehingga penataan ulang root folder jadi wajar dan hanya dilakukan sekali, bukan dua kali (sekarang demi frontend, nanti lagi demi microservices).

**Penyesuaian yang dibutuhkan untuk menambahkan `web/`:**
- `deployments/docker-compose.yml`: tambah service `web` baru (dev server Vite atau build statis di belakang Nginx untuk produksi), tidak perlu mengubah path/konfigurasi service `api` yang sudah ada
- `.gitignore`/`.dockerignore`: tambah entri untuk `web/node_modules`, `web/dist`
- CI: tambah job terpisah untuk lint/test/build frontend, berjalan independen dari job backend (tidak saling blocking)

## 2. Struktur `web/src/`

```
web/src/
├── app/
│   ├── routes/                  # TanStack Router file-based routes
│   │   ├── __root.tsx
│   │   ├── login.tsx
│   │   ├── patient/
│   │   │   ├── route.tsx        # layout + beforeLoad (auth guard role: patient)
│   │   │   ├── appointments.tsx
│   │   │   └── wallet.tsx
│   │   ├── doctor/
│   │   │   ├── route.tsx        # layout + beforeLoad (role: doctor)
│   │   │   └── schedule.tsx
│   │   └── admin/
│   │       └── route.tsx        # layout + beforeLoad (role: admin)
│   ├── router.tsx                # instance TanStack Router
│   └── providers.tsx             # QueryClientProvider, AuthProvider, dll — dikumpulkan satu tempat
│
├── features/                     # satu folder per domain, 1:1 dengan modul backend
│   ├── auth/
│   │   ├── api/                  # panggilan API spesifik auth (login, register, refresh)
│   │   ├── hooks/                # useLogin(), useCurrentUser()
│   │   ├── components/           # LoginForm, RegisterForm
│   │   └── types.ts              # tipe request/response (idealnya digenerate dari OpenAPI)
│   ├── appointment/
│   │   ├── api/
│   │   ├── hooks/                # useAppointments(), useBookAppointment()
│   │   ├── components/           # AppointmentCard, BookingCalendar
│   │   └── types.ts
│   ├── wallet/
│   ├── consultation/
│   ├── prescription/
│   ├── pharmacy/
│   ├── medical-records/
│   ├── ai-assistant/
│   └── notification/
│
├── components/
│   ├── ui/                       # shadcn/ui primitives (Button, Dialog, Input, dll)
│   └── shared/                   # komponen lintas-fitur (StatusBadge, PatientAvatar, EmptyState)
│
├── lib/
│   ├── api-client.ts             # fetch wrapper, interceptor auth/refresh, error parsing
│   ├── query-client.ts           # konfigurasi TanStack Query (default staleTime, retry, dll)
│   └── utils.ts                  # helper generik (formatCurrency, formatDate) — tanpa domain knowledge
│
├── stores/
│   └── ui-store.ts               # Zustand, hanya untuk state UI murni lintas-komponen
│
├── context/
│   └── auth-context.tsx          # AuthProvider — current user, token, login/logout function
│
├── types/
│   └── api.generated.ts          # (opsional) hasil openapi-typescript dari spec backend
│
├── styles/
│   └── globals.css               # Tailwind base + custom CSS variable (design tokens)
│
└── main.tsx                      # entry point
```

## 3. Tanggung Jawab Setiap Folder

| Folder | Tanggung Jawab | Boleh Import Dari |
|---|---|---|
| `app/routes/` | Definisi route, layout per role, auth guard (`beforeLoad`) | `features/*`, `components/*` |
| `features/<domain>/api/` | Fungsi pemanggil endpoint spesifik domain (memanggil `lib/api-client.ts`) | `lib/api-client.ts`, `types.ts` domain sendiri |
| `features/<domain>/hooks/` | Custom hook TanStack Query (`useQuery`/`useMutation`) yang membungkus `api/` | `api/` domain sendiri |
| `features/<domain>/components/` | Komponen UI spesifik domain, memakai `hooks/` domain sendiri | `hooks/` & `types.ts` domain sendiri, `components/ui/`, `components/shared/` |
| `components/ui/` | Primitive UI generik, tanpa pengetahuan domain apa pun | Tidak boleh import dari `features/*` |
| `components/shared/` | Komponen lintas-domain (dipakai ≥2 fitur) | `components/ui/` saja |
| `lib/` | Utilitas teknis murni (API client, query client, formatter generik) | Tidak boleh import dari `features/*` |
| `stores/` | State UI klien murni (bukan data server) | Tidak boleh menyimpan data yang berasal dari API |
| `context/` | Cross-cutting concern (auth) yang dipakai di banyak tempat tapi bukan "UI store" biasa | `lib/api-client.ts` |

## 4. Aturan yang Tidak Boleh Dilanggar

| Aturan | Alasan |
|---|---|
| `features/<domain-a>/` tidak boleh import langsung dari `features/<domain-b>/components` atau `hooks` | Sama seperti backend: domain saling lepas. Kalau butuh data lintas-domain (mis. appointment butuh nama dokter), lakukan lewat hook masing-masing domain yang dipanggil di level komponen gabungan (`app/routes/`), bukan import silang |
| `components/ui/` tidak boleh tahu apa pun soal domain | Supaya tetap benar-benar reusable dan bisa dites terisolasi |
| `lib/api-client.ts` adalah **satu-satunya** tempat yang memanggil `fetch` langsung | Semua request lewat sini supaya auth header, refresh logic, dan error parsing konsisten di seluruh app |
| Data dari server tidak boleh disalin ke `stores/` (Zustand) | Mencegah duplikasi sumber kebenaran dengan cache TanStack Query — risiko data stale |
| Setiap folder `features/<domain>/` sebaiknya punya `types.ts` yang selaras dengan kontrak di `07-api-design.md` | Kalau OpenAPI codegen dipakai, `types.ts` cukup re-export dari `types/api.generated.ts` — jangan duplikasi definisi manual |

## 5. Konvensi Penamaan

| Jenis | Konvensi | Contoh |
|---|---|---|
| Komponen | PascalCase, satu file satu komponen utama | `AppointmentCard.tsx` |
| Hook | camelCase, prefix `use` | `useBookAppointment.ts` |
| Folder domain | kebab-case, singular atau sesuai nama modul backend | `appointment/`, `medical-records/` |
| Tipe/interface | PascalCase, suffix jelas kalau perlu | `Appointment`, `AppointmentStatus`, `BookAppointmentRequest` |
| File route (TanStack Router) | mengikuti path URL | `patient/appointments.tsx` → `/patient/appointments` |

## 6. Testing (co-located, mengikuti pola backend)

```
features/appointment/
├── components/
│   ├── AppointmentCard.tsx
│   └── AppointmentCard.test.tsx      # component test, React Testing Library
├── hooks/
│   ├── useBookAppointment.ts
│   └── useBookAppointment.test.ts    # hook test, mocked api-client
└── api/
    └── appointment-api.ts

e2e/
└── booking-flow.spec.ts               # Playwright, flow lintas-fitur (login -> booking -> bayar)
```

Sejalan dengan `10-testing-strategy.md` backend: unit/component test menempel di sebelah kode yang diuji, hanya flow lintas-fitur yang naik ke folder top-level (`e2e/`).

---

Dengan ini, dokumentasi frontend (`00`–`03`) sudah lengkap untuk fase perencanaan. Struktur ini siap dipakai sebagai rujukan `AGENTS.md`/`GEMINI.md` versi frontend, kalau kamu ingin lanjut ke situ berikutnya.
