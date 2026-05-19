# UI Workflows Inventory

Generated: 2026-05-18

## Summary

| Metric | Value |
|--------|-------|
| Total distinct workflows | 34 |
| Functional areas | 10 |
| Roles | 5 |

---

## Workflows by Functional Area

### 1. Authentication (3 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 1 | User Login | All | Enter email/password → JWT issued → redirect to role home or force-change-password |
| 2 | Change Password | All | Enter current + new password → submit → redirect to home |
| 3 | View Profile (Me) | All | Auto-fetched on login; displayed in header/sidebar |

**Key files:** `frontend/src/pages/auth/`, `backend/internal/auth/`

---

### 2. Employee Exam Taking (5 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 4 | Browse Available Exams | employee | Navigate /portal → view assigned exam cards with status |
| 5 | Start an Exam | employee | Click Start → confirm modal → session created → redirect to exam |
| 6 | Take an Exam | employee | Answer questions (single/multiple/true-false/Likert/short-text) → navigate questions → flag for review → timer → submit with confirm modal |
| 7 | Handle Tab Switch Warning | employee | Tab-switch detected → modal shown → continue or exit exam |
| 8 | View Exam Results | employee | After submission → result screen (score, pass/fail, answers) → optional certificate download → all-results history at /portal/results |

**Key files:** `frontend/src/pages/EmployeePortal/`, `frontend/src/pages/ExamTaking/`, `backend/internal/portal/`, `backend/internal/sessions/`

---

### 3. User Management (6 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 9 | List & Filter Users | All admin | Navigate /admin/users → filter by dept/role/status → sort/search |
| 10 | Create a User | super_admin, dept_admin, hr_admin | Create drawer → fill email/name/dept/role → submit → temporary password shown |
| 11 | Edit a User | super_admin, dept_admin, hr_admin | Edit drawer → update name/dept/role → submit |
| 12 | Reset User Password | super_admin, dept_admin, hr_admin | Reset action → new temp password displayed → user forced to change on next login |
| 13 | Deactivate / Reactivate User | super_admin, dept_admin, hr_admin | Deactivate action → confirmation dialog → status toggled |
| 14 | Bulk Import Users | super_admin, dept_admin, hr_admin | Upload CSV → dry-run preview → confirm → users created with temp passwords |

**Key files:** `frontend/src/pages/admin/users/`, `backend/internal/users/`

---

### 4. Question Management (6 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 15 | Browse Question Bank | super_admin, dept_admin, examiner | Filter by category/tags/difficulty/status → search → paginate |
| 16 | Create / Edit a Question | super_admin, dept_admin, examiner | Select type & difficulty → set category & tags → add multilingual stem/explanation → add answer options with per-locale text → mark correct answers → set Likert weights/polarity or short-text model answer → save/submit/approve |
| 17 | Delete / Archive a Question | super_admin, dept_admin, examiner | Delete (if unused) or archive via action menu → confirm |
| 18 | Generate Questions with AI | super_admin, dept_admin, examiner | AI dialog → set category/count/difficulty/context → generate → review drafts → select and import |
| 19 | Import Questions from CSV | super_admin, dept_admin, examiner | Upload CSV → dry-run validation → confirm → created as drafts |
| 20 | Export Questions to CSV | super_admin, dept_admin, examiner | Filter bank → click Export → CSV downloaded |

**Key files:** `frontend/src/pages/admin/questions/`, `frontend/src/components/questions/AIGenerateDialog.tsx`, `backend/internal/questions/`

---

### 5. Content Organization (2 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 21 | Manage Categories | super_admin, dept_admin, examiner | View hierarchy → create/edit/delete category → drag-drop reorder |
| 22 | Manage Tags | super_admin, dept_admin, examiner | Search/filter tags → create/rename/delete → sort by name/usage/date → paginate |

**Key files:** `frontend/src/pages/admin/categories/`, `frontend/src/pages/admin/tags/`, `backend/internal/categories/`, `backend/internal/tags/`

---

### 6. Exam Creation & Lifecycle (5 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 23 | Create an Exam (4-step wizard) | super_admin, dept_admin, examiner | Step 1: basic settings (title, time limit, passing score, shuffle, anti-cheat, certificate) → Step 2: question rules (manual pick or random from category/tags) → Step 3: assign to departments/users → Step 4: review & publish |
| 24 | Edit an Exam | super_admin, dept_admin, examiner | Same 4-step wizard pre-populated; only editable in draft status |
| 25 | Publish an Exam | super_admin, dept_admin, examiner | Wizard Step 4 → Publish → backend validates rules → exam goes active |
| 26 | Archive an Exam | super_admin, dept_admin, examiner | Archive action → confirm → exam archived; employees can no longer access |
| 27 | List Exams | All admin | Navigate /admin/exams → filter by status → search → paginate |

**Key files:** `frontend/src/pages/ExamWizard/`, `frontend/src/pages/admin/ExamsListPage.tsx`, `backend/internal/exams/`

---

### 7. Grading (2 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 28 | View Grading Queue | super_admin, dept_admin, examiner | Navigate /admin/grading → view sessions pending manual grade → sort/paginate |
| 29 | Grade a Session | super_admin, dept_admin, examiner | Open session → per-question: view student answer → enter score (0–100) → add feedback → navigate questions → submit all grades → final score calculated |

**Key files:** `frontend/src/pages/admin/GradingQueuePage.tsx`, `frontend/src/pages/admin/GradingDetailPage.tsx`, `frontend/src/components/grading/`, `backend/internal/sessions/grading.go`

---

### 8. Analytics & Reporting (3 workflows)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 30 | View Admin Dashboard | super_admin, examiner, hr_admin | KPIs (employee count, active exams, avg score) → completion rate chart → overdue employees table → recent activity feed |
| 31 | View Exam Analytics | super_admin, dept_admin, examiner | Navigate to exam analytics → score distribution → pass rate → question difficulty stats → AI insights → export CSV |
| 32 | View Employee Record | super_admin, dept_admin, examiner | Navigate to employee record → info header → track progress → session history → AI values narrative (loyalty exams) |

**Key files:** `frontend/src/pages/admin/AdminDashboardPage.tsx`, `frontend/src/pages/admin/ExamAnalyticsPage.tsx`, `frontend/src/pages/admin/EmployeeRecordPage.tsx`, `backend/internal/analytics/`, `backend/internal/ai/`

---

### 9. Compliance & Audit (1 workflow)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 33 | View Audit Log | super_admin, hr_admin, examiner | Navigate /admin/audit → filter by date range/actor/action/entity → paginate → export to CSV |

**Key files:** `frontend/src/pages/admin/AuditLogPage.tsx`, `frontend/src/components/audit/`, `backend/internal/audit/`

---

### 10. System Configuration (1 workflow)

| # | Workflow | Roles | Key Steps |
|---|----------|-------|-----------|
| 34 | Configure Branding | super_admin only | Navigate /admin/settings/branding → set app name, logo, primary/accent colors, available locales, default locale → live preview → save |

**Key files:** `frontend/src/pages/admin/settings/BrandingSettingsPage.tsx`, `frontend/src/components/settings/`, `backend/internal/tenant/`

---

## Workflows by Role

| Role | Workflow Count | Workflows (#) |
|------|----------------|---------------|
| employee | 5 | 4–8 |
| examiner | 20 | 1–3, 9, 15–22, 23–32, 33 |
| department_admin | 22 | all examiner + 10–14 (own dept only) |
| hr_admin | 11 | 1–3, 9–14, 30, 33 |
| super_admin | 34 | all workflows |

---

## Backend Operations Reference

| Domain | Handler Methods |
|--------|----------------|
| auth | Login, ChangePassword, RefreshToken |
| users | ListUsers, CreateUser, UpdateUser, DeleteUser, ResetPassword, GetMe, ImportUsers |
| exams | Create, Update, Delete, Publish, Archive, AddSection, AddRule, ListExams, SetManualQuestions |
| questions | Create, Update, Delete, List, UpdateStatus, ImportQuestions, ExportQuestions, AddTag |
| categories | Create, Update, Delete, List, ListTree |
| tags | Create, Update, Delete, List |
| sessions | CreateSession, SaveAnswer, SubmitSession, ReportEvent, GetSession |
| portal | ListMyExams, GetMyExam |
| grading | GetGradingQueue, GetGradingSession, SubmitGrade |
| analytics | GetDashboard, GetExamAnalytics |
| audit | ListAuditLog, ExportAuditLog |
| tenant | UpdateConfig, GetConfig |
| ai | GenerateQuestions, GetExamInsights, GenerateValuesNarrative |
