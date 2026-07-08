# 08 — Development Roadmap

Sprints are assumed to be **1–2 weeks** each, part-time pace, consistent with `02-learning-roadmap.md`. Each sprint produces a working, demoable increment of the modular monolith.

---

## Sprint 0 — Project Setup (✅ Completed)

| | |
|---|---|
| **Goals** | Establish the foundation everything else builds on |
| **Features** | Repo init, folder structure (`05`), `go.mod`, Docker Compose skeleton (Postgres, Redis, MinIO), config loading, base logger, health-check endpoint |
| **Learning Objectives** | Go modules, project layout conventions, Docker Compose networking |
| **Dependencies** | None |
| **Deliverables** | `docker-compose up` boots an empty API returning `200` on `/healthz` |
| **Estimated Duration** | 3–5 days |

## Sprint 1 — Authentication (✅ Completed)

| | |
|---|---|
| **Goals** | Ship secure registration/login as the entry point for every other module |
| **Features** | `auth` module: register, login, JWT issuance, refresh, logout; password hashing |
| **Learning Objectives** | bcrypt/argon2, JWT libraries, middleware, RBAC foundations |
| **Dependencies** | Sprint 0 |
| **Deliverables** | Working `/auth/*` endpoints per `07-api-design.md`, unit-tested |
| **Estimated Duration** | 1–1.5 weeks |

## Sprint 2 — User Management (✅ Completed)

| | |
|---|---|
| **Goals** | Establish shared identity data used by doctor/patient profiles |
| **Features** | `user` module: profile retrieval/update, role assignment wiring |
| **Learning Objectives** | Repository pattern, DTO/mapper conventions |
| **Dependencies** | Sprint 1 (auth) |
| **Deliverables** | `/users/me` style endpoints, role-based middleware proven end-to-end |
| **Estimated Duration** | 3–5 days |

## Sprint 3 — Doctor (✅ Completed)

| | |
|---|---|
| **Goals** | Enable doctor profiles and availability — prerequisite for booking |
| **Features** | `doctor` module: profile, credential verification (admin), availability slots |
| **Learning Objectives** | One-to-one table relationships, admin-only authorization patterns |
| **Dependencies** | Sprint 2 |
| **Deliverables** | Doctor listing/search, availability CRUD, admin verification flow |
| **Estimated Duration** | 1 week |

## Sprint 4 — Appointment (✅ Completed)

| | |
|---|---|
| **Goals** | Deliver the platform's first core business transaction: booking |
| **Features** | `appointment` module: booking, cancellation, double-booking prevention |
| **Learning Objectives** | PostgreSQL transactions, row-level locking, unique partial constraints |
| **Dependencies** | Sprint 3 (doctor availability) |
| **Deliverables** | Race-condition-safe booking flow, load-tested for concurrent booking attempts |
| **Estimated Duration** | 1.5–2 weeks |

## Sprint 5 — Consultation (✅ Completed)

| | |
|---|---|
| **Goals** | Model the actual consultation session lifecycle |
| **Features** | `consultation` module: start/complete session, notes |
| **Learning Objectives** | State machine design in Go, lifecycle validation |
| **Dependencies** | Sprint 4 |
| **Deliverables** | Full appointment → consultation flow working end-to-end |
| **Estimated Duration** | 1 week |

## Sprint 6 — Prescription (✅ Completed)

| | |
|---|---|
| **Goals** | Enable doctors to issue prescriptions tied to a consultation |
| **Features** | `prescription` module: issuance with line items |
| **Learning Objectives** | Header/line-item schema patterns, nested DTO validation |
| **Dependencies** | Sprint 5 |
| **Deliverables** | Prescription issuance endpoint, viewable by patient/doctor |
| **Estimated Duration** | 3–5 days |

## Sprint 7 — Inventory (✅ Completed)

| | |
|---|---|
| **Goals** | Provide the medicine catalog backing pharmacy orders |
| **Features** | `inventory` module: medicine CRUD, stock tracking |
| **Learning Objectives** | Search/filter query patterns, basic stock-decrement logic |
| **Dependencies** | None (can run parallel to Sprints 3–6 if desired) |
| **Deliverables** | Medicine catalog endpoints per `07-api-design.md` |
| **Estimated Duration** | 3–5 days |

## Sprint 8 — Wallet (✅ Completed)

| | |
|---|---|
| **Goals** | Deliver the platform's financial backbone |
| **Features** | `wallet` module: top-up, balance, ledger; `pharmacy`/`orders` module wired to charge wallet atomically |
| **Learning Objectives** | Multi-table transactions, idempotency keys, ledger reconciliation pattern |
| **Dependencies** | Sprints 6, 7 (prescription → order → payment chain) |
| **Deliverables** | Order creation with atomic wallet deduction, concurrency load-tested |
| **Estimated Duration** | 1.5–2 weeks |

## Sprint 9 — Medical Records (✅ Completed)

| | |
|---|---|
| **Goals** | Deliver the security-sensitive records domain with full auditability |
| **Features** | `medical_records` module: CRUD, `audit_logs` writes on every read |
| **Learning Objectives** | Fine-grained authorization checks, audit-log design |
| **Dependencies** | Sprint 5 (consultations feed records) |
| **Deliverables** | Records API with verified audit trail on every doctor/admin access |
| **Estimated Duration** | 1 week |

## Sprint 10 — Notification (✅ Completed)

| | |
|---|---|
| **Goals** | Introduce asynchronous, decoupled event notification |
| **Features** | `notification` module: email dispatch on booking/order events, worker pool |
| **Learning Objectives** | Goroutines + channels at scale, worker pool pattern, retry/backoff |
| **Dependencies** | Sprints 4, 8 (events to notify on already exist) |
| **Deliverables** | Async notification dispatcher decoupled from request/response cycle |
| **Estimated Duration** | 1–1.5 weeks |

## Sprint 11 — AI Assistant (✅ Completed)

| | |
|---|---|
| **Goals** | Add the platform's differentiating AI-assisted triage feature |
| **Features** | `ai` module: session + suggestion endpoints, external LLM integration |
| **Learning Objectives** | Third-party HTTP client integration, structured/JSON-mode prompting, timeout/circuit-breaker |
| **Dependencies** | Sprint 2 (patient identity) |
| **Deliverables** | Working triage endpoint with disclaimer, response stored for audit |
| **Estimated Duration** | 1 week |

## Sprint 12 — Performance (✅ Completed)

| | |
|---|---|
| **Goals** | Harden the system under realistic load before adding more surface area |
| **Features** | Query optimization pass, N+1 elimination, index verification against `06-database-design.md` |
| **Learning Objectives** | Query profiling (`EXPLAIN ANALYZE`), Go pprof basics |
| **Dependencies** | Sprints 1–11 (there must be something to profile) |
| **Deliverables** | Benchmark report; P95 latency targets from `01-product-requirements.md` met |
| **Estimated Duration** | 3–5 days |

## Sprint 13 — Background Workers

| | |
|---|---|
| **Goals** | Generalize the worker pattern introduced in Sprint 10 into a reusable background-job subsystem |
| **Features** | Scheduled jobs (e.g., appointment reminders), retry queue |
| **Learning Objectives** | Cron-style scheduling in Go, job idempotency |
| **Dependencies** | Sprint 10 |
| **Deliverables** | Reminder jobs running reliably on a schedule |
| **Estimated Duration** | 1 week |

## Sprint 14 — Redis

| | |
|---|---|
| **Goals** | Introduce caching where it measurably matters |
| **Features** | Cache doctor availability lookups and other read-heavy endpoints; cache invalidation on writes |
| **Learning Objectives** | Cache-aside pattern, TTL design, invalidation strategy |
| **Dependencies** | Sprint 3 (availability), Sprint 12 (baseline to compare against) |
| **Deliverables** | Measured latency improvement on cached endpoints |
| **Estimated Duration** | 3–5 days |

## Sprint 15 — Observability

| | |
|---|---|
| **Goals** | Make the running system debuggable and measurable in production terms |
| **Features** | Structured logging audit pass, request tracing IDs, basic metrics (latency, error rate), `/metrics` endpoint |
| **Learning Objectives** | `slog` structured logging patterns, basic Prometheus-style metrics |
| **Dependencies** | All prior sprints (there must be a system to observe) |
| **Deliverables** | Dashboard-ready metrics endpoint, correlated logs via trace ID |
| **Estimated Duration** | 1 week |

## Sprint 16 — Service Extraction (Proof of Concept)

| | |
|---|---|
| **Goals** | Prove the Modular Monolith's extractability promise from `05-folder-structure.md` |
| **Features** | Extract one module (recommended: `notification`, lowest coupling) into a standalone service communicating via gRPC |
| **Learning Objectives** | Protobuf schema design, gRPC server/client in Go |
| **Dependencies** | Sprint 10, Sprint 15 (observability needed across a now-distributed call) |
| **Deliverables** | Two services communicating via gRPC; monolith calls the extracted service through the same interface as before |
| **Estimated Duration** | 1.5–2 weeks |

## Sprint 17 — Microservice Preparation

| | |
|---|---|
| **Goals** | Generalize the extraction pattern and prepare remaining modules for future extraction |
| **Features** | Service discovery approach decision, shared proto contracts, evaluate which modules are next extraction candidates (e.g., `ai`, `wallet`) |
| **Learning Objectives** | Distributed systems tradeoffs (consistency, latency, failure handling) |
| **Dependencies** | Sprint 16 |
| **Deliverables** | `11-future-roadmap.md`-aligned extraction plan; no further modules extracted yet unless justified by real load |
| **Estimated Duration** | 1 week |

## Sprint 18 — Production Deployment

| | |
|---|---|
| **Goals** | Take the platform from local Docker Compose to a real, reachable deployment |
| **Features** | CI/CD pipeline, cloud deployment (single instance or managed container platform), environment/secrets management |
| **Learning Objectives** | CI/CD basics, cloud deployment fundamentals, secrets management |
| **Dependencies** | Sprint 15 (observability needed before going live) |
| **Deliverables** | Publicly accessible TeleMedHub instance with automated deploy on merge to `main` |
| **Estimated Duration** | 1.5–2 weeks |

---

| Sprint | Focus | Status | Depends On |
|---|---|---|---|
| 0 | Project Setup | ✅ Completed | — |
| 1 | Authentication | ✅ Completed | 0 |
| 2 | User Management | ✅ Completed | 1 |
| 3 | Doctor | ✅ Completed | 2 |
| 4 | Appointment | ✅ Completed | 3 |
| 5 | Consultation | ✅ Completed | 4 |
| 6 | Prescription | ✅ Completed | 5 |
| 7 | Inventory | ✅ Completed | — (parallel-safe) |
| 8 | Wallet | ⏳ Next | 6, 7 |
| 9 | Medical Records | ⏳ Pending | 5 |
| 10 | Notification | ⏳ Pending | 4, 8 |
| 11 | AI Assistant | ⏳ Pending | 2 |
| 12 | Performance | ⏳ Pending | 1–11 |
| 13 | Background Workers | ⏳ Pending | 10 |
| 14 | Redis | ⏳ Pending | 3, 12 |
| 15 | Observability | ⏳ Pending | all prior |
| 16 | Service Extraction | ⏳ Pending | 10, 15 |
| 17 | Microservice Preparation | ⏳ Pending | 16 |
| 18 | Production Deployment | ⏳ Pending | 15 |

---

**Next document:** `09-deployment.md` — deployment strategy from local dev to production.
