# Requirements Backlog — Implementation Order

> Recommended implementation order for BilimBaga, following phase dependencies from the roadmap.
> Each phase builds on the previous. Do not implement a later phase before its dependencies are complete.

---

## Phase 1 — Foundation

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB11 | Project scaffold (Docker, Go module, Vite, Nginx, Makefile) | — | 1 |
| FR-BB12 | Database bootstrap (migrations runner, connection pool) | FR-BB11 | 1 |
| FR-BB13 | Tenant configuration (app_name, logo, colors, locales) | FR-BB12 | 1 |
| FR-BB14 | Authentication (login, refresh, logout, change-password) | FR-BB12 | 1 |
| FR-BB15 | JWT middleware (auth, context injection) | FR-BB14 | 1 |
| FR-BB16 | RBAC (roles, permissions, role_permissions seed) | FR-BB15 | 1 |
| FR-BB17 | Department management | FR-BB16 | 2 |
| FR-BB18 | User management API | FR-BB16 | 2 |
| FR-BB19 | Audit log | FR-BB16 | 2 |
| FR-BB110 | Frontend: auth screens | FR-BB14, FR-BB13 | 2 |
| FR-BB111 | Frontend: admin shell | FR-BB110, FR-BB17, FR-BB18 | 3 |
| FR-BB112 | Frontend: branding settings | FR-BB13, FR-BB111 | 3 |

## Phase 2 — Content Management

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB21 | Categories and tags | FR-BB16 | 1 |
| FR-BB22 | Question model (tables, translations, options) | FR-BB21 | 1 |
| FR-BB23 | Question CRUD API | FR-BB22 | 1 |
| FR-BB24 | Translation API | FR-BB23 | 2 |
| FR-BB25 | Bulk import / export | FR-BB23 | 2 |
| FR-BB26 | Frontend: question editor | FR-BB23, FR-BB24 | 2 |
| FR-BB27 | Frontend: question bank list | FR-BB23, FR-BB25 | 3 |

## Phase 3 — Exam Engine

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB31 | Exam configuration model | FR-BB22 | 1 |
| FR-BB32 | Exam configuration API | FR-BB31 | 1 |
| FR-BB33 | Exam assignment | FR-BB32 | 1 |
| FR-BB34 | Employee exam portal API | FR-BB33 | 1 |
| FR-BB35 | Session creation | FR-BB34 | 1 |
| FR-BB36 | Session tables | FR-BB35 | 1 |
| FR-BB37 | Answer saving | FR-BB36 | 1 |
| FR-BB38 | Tab-switch events | FR-BB37 | 2 |
| FR-BB39 | Session submission | FR-BB37 | 1 |
| FR-BB310 | Auto-submit background job | FR-BB39 | 2 |
| FR-BB311 | Grading engine | FR-BB39 | 1 |
| FR-BB312 | Frontend: exam configuration UI | FR-BB32, FR-BB33 | 2 |
| FR-BB313 | Frontend: employee portal | FR-BB34 | 2 |
| FR-BB314 | Frontend: exam taking screen | FR-BB37, FR-BB38, FR-BB39 | 2 |

## Phase 4 — Results & Certificates

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB41 | Result retrieval API | FR-BB311 | 1 |
| FR-BB42 | Manual grading queue | FR-BB39 | 1 |
| FR-BB43 | Certificate generation | FR-BB41 | 1 |
| FR-BB44 | PDF generation | FR-BB43 | 1 |
| FR-BB45 | Frontend: result screen | FR-BB41, FR-BB43 | 2 |
| FR-BB46 | Frontend: employee history | FR-BB41 | 2 |
| FR-BB47 | Frontend: manual grading UI | FR-BB42 | 2 |

## Phase 5 — Analytics & Reporting

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB51 | Dashboard metrics API | FR-BB311, FR-BB33 | 1 |
| FR-BB52 | Per-exam analytics API | FR-BB311 | 1 |
| FR-BB53 | Per-employee record API | FR-BB311 | 1 |
| FR-BB54 | Export API | FR-BB52, FR-BB53 | 2 |
| FR-BB55 | Audit log viewer | FR-BB19 | 2 |
| FR-BB56 | Frontend: admin dashboard | FR-BB51 | 2 |
| FR-BB57 | Frontend: per-exam analytics | FR-BB52 | 2 |
| FR-BB58 | Frontend: employee record | FR-BB53 | 2 |

## Phase 6 — Polish & Hardening

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB61 | Email notifications | FR-BB33 | 1 |
| FR-BB62 | Full i18n coverage | All phases | 1 |
| FR-BB63 | Accessibility | All frontend | 2 |
| FR-BB64 | Security hardening | All phases | 1 |
| FR-BB65 | Performance | All phases | 2 |
| FR-BB66 | Observability (structured logging, health) | All phases | 1 |

## Phase 7 — AI Layer

| FR | Title | Depends On | Priority |
|----|-------|-----------|---------|
| FR-BB71 | Question generation assist | FR-BB23 | 1 |
| FR-BB72 | Adaptive difficulty | FR-BB311 | 2 |
| FR-BB73 | Short-text auto-grading | FR-BB311 | 1 |
| FR-BB74 | Performance insight summaries | FR-BB52 | 2 |
| FR-BB75 | Loyalty profile narrative | FR-BB311 | 2 |
