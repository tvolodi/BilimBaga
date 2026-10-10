# Requirements Index

> All FR-BBxxx requirement documents. Look here first to find any requirement by number.

## Format

`FR-BB{phase}{section}` — e.g. `FR-BB14` = Phase 1, Section 1.4 — Authentication

## Index

| ID | Slug | Title | Status |
|----|------|-------|--------|
| FR-BB11 | Project-scaffold | Project Scaffold | Implemented |
| FR-BB12 | Database-bootstrap | Database Bootstrap | Implemented |
| FR-BB13 | Tenant-configuration | Tenant Configuration | Implemented |
| FR-BB14 | Authentication | Authentication | Implemented |
| FR-BB15 | JWT-middleware | JWT Middleware | Implemented |
| FR-BB16 | RBAC | Role-Based Access Control | Implemented |
| FR-BB17 | Department-management | Department Management | Implemented |
| FR-BB18 | User-management-API | User Management API | Implemented |
| FR-BB19 | Audit-log | Audit Log | Implemented |
| FR-BB110 | Frontend-auth-screens | Frontend: Auth Screens | Implemented |
| FR-BB111 | Frontend-admin-shell | Frontend: Admin Shell | Implemented |
| FR-BB112 | Frontend-branding-settings | Frontend: Branding Settings | Implemented |
| FR-BB113 | Frontend-department-management | Frontend: Department Management | Implemented |
| FR-BB114 | Frontend-audit-log-viewer | Frontend: Audit Log Viewer (Phase 1) | Implemented |
| FR-BB115 | Account-recovery | Account Recovery: Forgot Password and Lockout Unlock | Implemented |
| FR-BB116 | User-profile-and-preferred-locale | User Profile Page and Persisted Preferred Locale | Implemented |
| FR-BB117 | Role-management | Role Management: custom roles and permission matrix (issue #135) | Implemented (backend migration 034, PR #185; frontend PR #206; static conformance done, open gaps G1/G2 in conformance/FR-BB117-PR204-PR211-PR205-conformance-20261009.md; pending live UAT) |
| FR-BB21 | Categories-and-tags | Categories and Tags | Implemented |
| FR-BB22 | Question-model | Question Model | Implemented |
| FR-BB23 | Question-CRUD-API | Question CRUD API | Implemented |
| FR-BB24 | Translation-API | Translation API | Implemented |
| FR-BB25 | Bulk-import-export | Bulk Import / Export | Implemented |
| FR-BB26 | Frontend-question-editor | Frontend: Question Editor | Implemented |
| FR-BB27 | Frontend-question-bank-list | Frontend: Question Bank List | Implemented |
| FR-BB28 | Frontend-categories-management | Frontend: Categories Management | Implemented |
| FR-BB29 | Frontend-tags-management | Frontend: Tags Management | Implemented |
| FR-BB31 | Exam-configuration-model | Exam Configuration Model | Implemented |
| FR-BB32 | Exam-configuration-API | Exam Configuration API | Implemented |
| FR-BB33 | Exam-assignment | Exam Assignment | Implemented |
| FR-BB34 | Employee-exam-portal-API | Employee Exam Portal API | Implemented |
| FR-BB35 | Session-creation | Session Creation | Implemented |
| FR-BB36 | Session-tables | Session Tables | Implemented |
| FR-BB37 | Answer-saving | Answer Saving | Implemented |
| FR-BB38 | Tab-switch-events | Tab Switch Events | Implemented |
| FR-BB39 | Session-submission | Session Submission | Implemented |
| FR-BB310 | Auto-submit-background-job | Auto-Submit Background Job | Implemented |
| FR-BB311 | Grading-engine | Grading Engine | Implemented |
| FR-BB312 | Frontend-exam-configuration-UI | Frontend: Exam Configuration UI | Implemented |
| FR-BB313 | Frontend-employee-portal | Frontend: Employee Portal | Implemented |
| FR-BB314 | Frontend-exam-taking-screen | Frontend: Exam Taking Screen | Implemented |
| FR-BB41 | Result-retrieval-API | Result Retrieval API | Implemented |
| FR-BB42 | Manual-grading-queue | Manual Grading Queue | Implemented |
| FR-BB43 | Certificate-generation | Certificate Generation | Implemented |
| FR-BB44 | PDF-generation | PDF Generation | Implemented |
| FR-BB45 | Frontend-result-screen | Frontend: Result Screen | Implemented |
| FR-BB46 | Frontend-employee-history | Frontend: Employee History | Implemented |
| FR-BB47 | Frontend-manual-grading-UI | Frontend: Manual Grading UI | Implemented |
| FR-BB48 | Public-certificate-verification-page | Public Certificate Verification Page | Implemented |
| FR-BB51 | Dashboard-metrics-API | Dashboard Metrics API | Partial (only AC-9 perf budget unverified) |
| FR-BB52 | Per-exam-analytics-API | Per-Exam Analytics API | Implemented |
| FR-BB53 | Per-employee-record-API | Per-Employee Record API | Implemented |
| FR-BB54 | Export-API | Export API | Implemented |
| FR-BB55 | Audit-log-viewer | Audit Log Viewer | Implemented |
| FR-BB56 | Frontend-admin-dashboard | Frontend: Admin Dashboard | Implemented |
| FR-BB57 | Frontend-per-exam-analytics | Frontend: Per-Exam Analytics | Implemented |
| FR-BB58 | Frontend-employee-record | Frontend: Employee Record | Implemented |
| FR-BB61 | Email-notifications | Email Notifications | Implemented |
| FR-BB62 | Full-i18n-coverage | Full i18n Coverage | Implemented |
| FR-BB63 | Accessibility | Accessibility | Implemented |
| FR-BB64 | Security-hardening | Security Hardening | Implemented |
| FR-BB65 | Performance | Performance | Validated |
| FR-BB66 | Observability | Observability | Implemented |
| FR-BB67 | e2e-coverage-employee-portal-exam-taking-grading | E2E Coverage: Employee Portal, Exam Taking, Result & Grading Flows | Implemented |
| FR-BB71 | Question-generation-assist | Question Generation Assist | Implemented |
| FR-BB72 | Adaptive-difficulty | Adaptive Difficulty | Implemented |
| FR-BB73 | Short-text-auto-grading | Short-Text Auto-Grading | Implemented |
| FR-BB74 | Performance-insight-summaries | Performance Insight Summaries | Implemented |
| FR-BB75 | Loyalty-profile-narrative | Loyalty Profile Narrative | Implemented |
| FR-BB315 | exam-step4-eligible-question-counts | Exam Step 4: Eligible Question Counts per Rule | Implemented |
| FR-BB316 | FR-BB316-portal-locale-switcher | Portal Locale Switcher | Implemented |
| FR-BB317 | department-treeview | Department Treeview Selector | Implemented (Users List filter shipped, #40) |
| FR-BB318 | exam-unpublish | Unpublish Exam | Implemented |
| FR-BB59 | Reports-hub-page | Frontend: Reports Hub Page | Implemented |
| FR-BB319 | Anti-cheat-event-hygiene | Anti-Cheat Event Hygiene: Single Report per Incident, Auto-Submit Only on Tab Switch, Fullscreen Exit | Validated |
| FR-BB320 | Design-system-conformance | Design System Conformance and Enforcement | Validated |
| FR-BB510 | Overdue-reminder-action | Overdue Employee Reminder: Real "Send Reminder" Action | Validated |

## Decisions

Decision records (not FR-BB requirements).

| ID | File | Title | Status |
|----|------|-------|--------|
| DEC-001 | DEC-001.Environments-production-class-demo-and-qa.md | Environments: Production-Class Demo and Separate QA | Accepted |

## Reference / Conventions

Cross-cutting reference documents (not FR-BB requirements).

| ID | File | Title | Status |
|----|------|-------|--------|
| REF-API | api-conventions.md | API Conventions (as implemented): envelope, error codes, UUID rule, auth, pagination, uploads, audit | Accepted |

## Status Values

| Status | Meaning |
|--------|---------|
| `draft` | Written, not yet validated |
| `validated` | Passed Requirement Validation |
| `in-progress` | Currently being implemented |
| `implemented` | Code complete, tests passing, committed |
| `partial` | Mostly shipped but one or more ACs unmet; see `docs/handoffs/ba-drift-check/report.md` |
| `archived` | Superseded or cancelled |

## Backlog / Implementation Order

See `requirements-backlog.md` for the recommended implementation order with phase dependencies.
