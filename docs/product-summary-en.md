# BilimBaga — Corporate Exam & Assessment Platform

**BilimBaga** is an enterprise-grade exam and knowledge assessment platform built for organizations that need structured, measurable employee testing — across departments, in multiple languages, with full accountability.

---

## What it does

### Employee exam experience
Employees access a personal portal where they see assigned exams, take them in a clean, distraction-aware interface, and receive immediate results. The platform supports five question types — single choice, multiple choice, true/false, Likert scale, and short text — with a built-in timer, question flagging, and anti-tab-switch detection. After submission, employees see their score, pass/fail status, and can download a certificate if they passed.

### Exam authoring
Admins and examiners build exams through a guided 4-step wizard: configure basic settings (time limit, passing score, shuffle, anti-cheat, certificate issuance) → define question rules (manual selection or random draw from categories/tags) → assign to departments or individual users → review and publish.

### Question bank
A centralized, versioned question bank supports full multilingual content (per-locale question text and answer options). Questions can be created manually, bulk-imported from CSV, or generated using AI with a configurable prompt (topic, difficulty, count). The bank includes category hierarchies, tags, difficulty levels, and a full edit-and-approval workflow (draft → review → active → archived).

### Analytics and reporting
Admins see a live dashboard with KPIs (headcount, active exams, average scores, completion rates, overdue employees). Individual exam analytics include score distribution, pass rate, and per-question difficulty breakdowns. AI-generated narrative insights are available for values and loyalty exam tracks. Employee records show full assessment history and track progress.

### Grading
Short-text answers are routed to a manual grading queue. Examiners score each answer (0–100) with written feedback, and the final session score is recalculated automatically once all answers are graded.

### User and department management
Hierarchical department structure with role-based access control across five roles: **Super Admin**, **Department Admin**, **HR Admin**, **Examiner**, and **Employee**. Admins can create users individually or bulk-import via CSV. Accounts include temporary-password flows, forced password change on first login, and deactivation without data loss.

### Compliance and audit
An immutable append-only audit log records every significant action — logins, user changes, exam publications, grade submissions — with full CSV export for compliance review.

### White-label branding
Each deployment is fully branded: custom app name, logo, primary and accent colors, and supported languages are all configurable via an admin settings page with a live preview.

---

## Key capabilities at a glance

| Capability | Detail |
|---|---|
| Question types | Single choice, multiple choice, true/false, Likert, short text |
| Languages | Fully multilingual — any number of locales per question |
| AI features | AI question generation, exam analytics insights, values narrative |
| Access control | 5 roles, department-scoped permissions |
| Anti-cheat | Tab-switch detection, configurable per exam |
| Certificates | Auto-issued on passing; QR verification |
| Bulk operations | CSV import for users and questions |
| Audit trail | Immutable log, filterable, CSV export |
| Branding | Full white-label per deployment |

---

*Built on Go + React + PostgreSQL. Deployable via Docker Compose. Designed for single-tenant enterprise deployment with multi-tenant architecture ready.*
