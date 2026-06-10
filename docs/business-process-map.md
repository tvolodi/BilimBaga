# BilimBaga — Business Process Map

## Dependency Chain

```
┌─────────────────────────────────┐
│      Tenant Configuration       │  ← foundation; no dependencies
│  FR-BB13, FR-BB112, FR-BB62    │
│  Actor: Super Admin             │
│  Output: branding, languages    │
└────────────────┬────────────────┘
                 │
┌────────────────▼────────────────┐
│        User Onboarding          │
│  FR-BB14–19, FR-BB110–111      │
│  Actors: Super Admin, Dept Admin│
│  Output: active user accounts   │
└──────────┬─────────────┬────────┘
           │             │
           │    ┌────────▼────────────────────┐
           │    │     Question Authoring       │
           │    │  FR-BB21–29, FR-BB71        │
           │    │  Actors: Examiner, Admin    │
           │    │  Output: active questions   │
           │    └────────┬────────────────────┘
           │             │
           │    ┌────────▼────────────────────┐
           │    │  Exam Configuration &        │
           │    │      Publishing              │
           │    │  FR-BB31–32, FR-BB312       │
           │    │  Actors: Examiner, Admin    │
           │    │  Output: published exams    │
           │    └────────┬────────────────────┘
           │             │
┌──────────▼─────────────▼────────┐
│         Exam Assignment          │
│  FR-BB33, FR-BB34, FR-BB61     │
│  Actors: Admin, Dept Admin      │
│  Output: assignment records;    │
│          emails to employees    │
└──────────────┬──────────────────┘
               │
┌──────────────▼──────────────────┐
│      Employee Exam Taking        │
│  FR-BB34–39, FR-BB310–314      │
│  FR-BB72                        │
│  Actor: Employee, System        │
│  Output: session record         │
└────────┬─────────────┬──────────┘
         │             │
┌────────▼────────┐  ┌─▼──────────────────────┐
│  Manual Grading │  │  Result Review &        │
│  FR-BB42,47,73  │  │    Certification        │
│  Actor: Examiner│  │  FR-BB41–46            │
│  Output: scored ├──►  Actors: Employee,Admin │
│  short-text     │  │  Output: result screen, │
└─────────────────┘  │  PDF certificate        │
                     └─────────┬───────────────┘
                               │
┌──────────────────────────────▼──┐
│     Analytics & Reporting        │
│  FR-BB51–59, FR-BB19, 74, 75   │
│  Actors: Admin, Dept Admin      │
│  Output: dashboards, exports,   │
│  audit logs, AI summaries       │
└──────────────────────────────┬──┘
                               │
┌──────────────────────────────▼──┐
│   AI-Assisted Content Authoring  │
│  FR-BB71–75                     │
│  Actors: Examiner, AI Service   │
│  Output: draft question          │
│  suggestions, adaptive scoring,  │
│  loyalty narratives              │
└──────────────────────────────────┘
```

---

## Process Dependency Table

| Process | Depends On | FR-BBs | Key Actors |
|---------|-----------|--------|------------|
| **Tenant Configuration** | — | BB13, BB112, BB62 | Super Admin |
| **User Onboarding** | Tenant Config | BB14–19, BB110–111 | Super Admin, Dept Admin |
| **Question Authoring** | User Onboarding | BB21–29, BB71 | Examiner, Admin |
| **Exam Configuration** | Question Authoring | BB31–32, BB312 | Examiner, Admin |
| **Exam Assignment** | Exam Config + User Onboarding | BB33–34, BB61 | Admin, Dept Admin |
| **Employee Exam Taking** | Exam Assignment | BB34–39, BB310–314, BB72 | Employee, System |
| **Manual Grading** | Exam Taking (short-text sessions) | BB42, BB47, BB73 | Examiner |
| **Result & Certification** | Exam Taking + Manual Grading | BB41–46 | Employee, Admin |
| **Analytics & Reporting** | All of the above | BB51–59, BB19, BB74–75 | Admin, Dept Admin |
| **AI Content Authoring** | Question Authoring + Analytics | BB71–75 | Examiner, AI Service |

---

## Cross-Cutting Concerns

These are not standalone processes — they apply across all processes:

| Concern | FR-BBs | Affects |
|---------|--------|---------|
| Email notifications | BB61 | Assignment, grading complete, deadline reminders |
| i18n / localization | BB62 | All UI processes (KK / RU / EN) |
| Accessibility | BB63 | All frontend processes |
| Security hardening | BB64 | Auth, session, API layer |
| Performance | BB65 | Exam taking, analytics queries |
| Observability | BB66 | All backend services |
| E2E test coverage | BB67 | Exam taking, grading, portal |
