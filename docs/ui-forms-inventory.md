# UI Inventory

Generated: 2026-05-18

## Summary

| Metric | Value |
|--------|-------|
| Total screen forms | 34 |
| Form categories | 12 |
| Total workflows | 34 |
| Workflow functional areas | 10 |
| Roles | 5 |

---

## Forms by Category

### Authentication (2)

| # | Form | File | Elements |
|---|------|------|----------|
| 1 | Login | `frontend/src/pages/auth/LoginPage.tsx` | Email input, Password input, Submit button, Language selector |
| 2 | Change Password | `frontend/src/pages/auth/ChangePasswordPage.tsx` | Current password, New password, Confirm password, Submit button, Error display |

---

### User Management (7)

| # | Form | File | Elements |
|---|------|------|----------|
| 3 | Users List | `frontend/src/pages/users/UsersListPage.tsx` | Status filter select, Import button, Create user button, Pagination buttons |
| 4 | User Create Drawer | `frontend/src/pages/users/UserCreateDrawer.tsx` | Email input, Full name input, Department ID input, Role ID input, Submit button, Cancel button, Error display |
| 5 | User Edit Drawer | `frontend/src/pages/users/UserEditDrawer.tsx` | Full name input, Department ID input, Role ID input, Submit button, Cancel button, Error display |
| 6 | Import Modal | `frontend/src/pages/users/ImportModal.tsx` | CSV file input, Preview button, Commit button, Close button, Validation display |
| 7 | Password Reset Modal | `frontend/src/pages/users/PasswordResetModal.tsx` | Temporary password display (read-only), Copy button, Close button |
| 8 | Deactivate Confirm Dialog | `frontend/src/pages/users/DeactivateConfirmDialog.tsx` | Confirm button, Cancel button, Error display |
| 9 | Admin Users List | `frontend/src/pages/admin/users/UsersListPage.tsx` | Department filter, Role filter, Status filter, Sortable column headers, Import button, Create user button, Pagination buttons |

---

### Settings (1)

| # | Form | File | Elements |
|---|------|------|----------|
| 10 | Branding Settings | `frontend/src/pages/admin/settings/BrandingSettingsPage.tsx` | App name input, Logo uploader (drag & drop), Primary color picker, Accent color picker, Available locales checkboxes (kk/ru/en), Default locale select, Save button, Success/error feedback |

---

### Categories (2)

| # | Form | File | Elements |
|---|------|------|----------|
| 11 | Categories Page | `frontend/src/pages/admin/categories/CategoriesPage.tsx` | Create category button, Delete confirm dialog, Edit modal trigger, Drag & drop reorder |
| 12 | Category Edit Modal | `frontend/src/pages/admin/categories/CategoryEditModal.tsx` | Name input (max 100), Parent category select, Track input (max 50), Sort order number input, Save button, Cancel button, Inline error messages |

---

### Tags (3)

| # | Form | File | Elements |
|---|------|------|----------|
| 13 | Tags Page | `frontend/src/pages/admin/tags/TagsPage.tsx` | Search input (debounced), Create tag button, Sort controls, Page size select (25/50/100), Pagination buttons, Actions menu per tag |
| 14 | Tag Create Modal | `frontend/src/pages/admin/tags/TagCreateModal.tsx` | Tag name input (max 64), Save button, Cancel button, Error display |
| 15 | Tag Rename Modal | `frontend/src/pages/admin/tags/TagRenameModal.tsx` | Tag name input (max 64), Save button, Cancel button, Error display |

---

### Questions (1)

| # | Form | File | Elements |
|---|------|------|----------|
| 16 | Question Editor | `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` | Question type select, Difficulty select, Category picker, Tags combobox (multi-select), Locale tabs (EN/KK/RU), Question stem textarea, Explanation textarea, Answer options (draggable, per-locale: body input, radio/checkbox, Likert weight & polarity), Model answer textarea (shorttext), Auto-grade toggle, Status transition buttons, Save button, Auto-save indicator, Back button |

---

### Exam Wizard (4)

| # | Form | File | Elements |
|---|------|------|----------|
| 17 | Step 1 — Basic Settings | `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` | Title input, Description textarea, Time limit (1–300 min), Passing score (0–100%), Max attempts (1–10), Available from/until datetime pickers, Show answers select, On tab switch select, Shuffle questions toggle, Shuffle options toggle, Certificate enabled toggle, Next button |
| 18 | Step 2 — Question Rules | `frontend/src/pages/ExamWizard/Step2QuestionRules.tsx` | Per rule: mode toggle (manual/random), category select, difficulty select, tags select with pills, count input, edit questions button, delete rule button; Add rule button, Back/Next buttons |
| 19 | Step 2 — Question Picker Modal | `frontend/src/pages/ExamWizard/Step2QuestionPickerModal.tsx` | Search/filter inputs, Question selection checkboxes, Confirm button, Cancel button |
| 20 | Step 3 — Assignments | `frontend/src/pages/ExamWizard/Step3Assignments.tsx` | Assignee type select (user/department/all), User select, Department select, Deadline datetime picker, Add assignment button, Save/Cancel/Delete buttons, Back/Next buttons |

---

### Exam Taking (7)

| # | Form | File | Elements |
|---|------|------|----------|
| 21 | Exam Taking — Main Container | `frontend/src/pages/ExamTaking/index.tsx` | ExamLayout wrapper, question input components |
| 22 | Single Choice Input | `frontend/src/pages/ExamTaking/SingleChoiceInput.tsx` | Radio button group, Option labels |
| 23 | Multiple Choice Input | `frontend/src/pages/ExamTaking/MultipleChoiceInput.tsx` | Checkbox group, Option labels |
| 24 | Short Text Input | `frontend/src/pages/ExamTaking/ShortTextInput.tsx` | Textarea (4 rows) |
| 25 | Likert Input | `frontend/src/pages/ExamTaking/LikertInput.tsx` | Radio button group (styled as button row) |
| 26 | Submit Confirm Modal | `frontend/src/pages/ExamTaking/SubmitConfirmModal.tsx` | Confirm submit button, Cancel button, Warning text |
| 27 | Tab Switch Warning Modal | `frontend/src/pages/ExamTaking/TabSwitchWarningModal.tsx` | Continue button, Exit exam button (destructive) |

---

### Grading (3)

| # | Form | File | Elements |
|---|------|------|----------|
| 28 | Grading Queue Page | `frontend/src/pages/admin/GradingQueuePage.tsx` | Pagination buttons |
| 29 | Question Grader | `frontend/src/components/grading/QuestionGrader.tsx` | Question stem display (read-only), Grading status badge, AI reasoning collapsible, Answer textarea (read-only), Score slider (0–100), Score number input (0–100), Feedback textarea, Error display |
| 30 | Grading Detail Page | `frontend/src/pages/admin/GradingDetailPage.tsx` | Multiple QuestionGrader instances, Question navigator, Submit all grades button, Previous/Next question buttons |

---

### Audit Logs (2)

| # | Form | File | Elements |
|---|------|------|----------|
| 31 | Audit Log Page | `frontend/src/pages/admin/AuditLogPage.tsx` | AuditFilterBar, Export CSV button, Pagination buttons |
| 32 | Audit Filter Bar | `frontend/src/components/audit/AuditFilterBar.tsx` | From datetime input, To datetime input, Actor name input (debounced), Entity type select, Action toggle chips (multi-select), Clear filters button |

---

### Employee Portal (1)

| # | Form | File | Elements |
|---|------|------|----------|
| 33 | Start Exam Modal | `frontend/src/pages/EmployeePortal/StartExamModal.tsx` | Confirm button, Cancel button, Warning text display |

---

### AI Generation (1)

| # | Form | File | Elements |
|---|------|------|----------|
| 34 | AI Generate Dialog | `frontend/src/components/questions/AIGenerateDialog.tsx` | Category select, Difficulty select, Count input (1–10), Context textarea (max 2000), Generate button; Draft preview: question checkboxes, Select all button, Reset button, Expandable draft cards, Confirm selected button, Cancel button |

---

## Element Type Summary

| Element Type | Approx. Count |
|---|---|
| Text inputs | 25+ |
| Textareas | 12+ |
| Select / Dropdown | 20+ |
| Number inputs | 15+ |
| Buttons (action / submit) | 50+ |
| Checkboxes | 8+ |
| Radio buttons | 6+ |
| Datetime pickers | 4 |
| Toggle / Switch | 3 |
| File inputs | 2 |
| Color pickers | 2 |
| Slider | 1 |

---

## Workflows Inventory

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
| 6 | Take an Exam | employee | Answer questions (single/multiple/true-false/Likert/short-text) → navigate → flag for review → timer → submit with confirm modal |
| 7 | Handle Tab Switch Warning | employee | Tab-switch detected → modal shown → continue or exit exam |
| 8 | View Exam Results | employee | After submission → result screen (score, pass/fail, answers) → optional certificate download → results history at /portal/results |

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
| 16 | Create / Edit a Question | super_admin, dept_admin, examiner | Select type & difficulty → set category & tags → add multilingual stem/explanation → add answer options → mark correct answers → set Likert weights or short-text model answer → save/submit/approve |
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

### Workflows by Role

| Role | Workflow Count | Workflows (#) |
|------|----------------|---------------|
| employee | 5 | 4–8 |
| examiner | 20 | 1–3, 9, 15–22, 23–32, 33 |
| department_admin | 22 | all examiner + 10–14 (own dept only) |
| hr_admin | 11 | 1–3, 9–14, 30, 33 |
| super_admin | 34 | all workflows |

---

### Backend Operations Reference

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

---

## E2E Test Coverage

Generated: 2026-05-18  
Framework: Playwright Test v1.60.0 · Chromium · Base URL: `http://localhost:4173` (mock) / `http://localhost:5173` (live)

### Test Files

| File | What It Covers |
|------|---------------|
| `e2e/auth.spec.ts` | Login form, invalid credentials, successful login redirect, admin access guard |
| `e2e/full-walkthrough.spec.ts` | 22-test live-backend walkthrough covering all major admin screens |
| `e2e/employee-portal.spec.ts` | Employee portal exam cards, start exam modal |
| `e2e/exam-taking.spec.ts` | All 5 question types, navigation, flag, tab-switch warning, submit |
| `e2e/exam-result.spec.ts` | Result screen states (passed/failed/pending), score display, navigation |
| `e2e/exam-wizard.spec.ts` | 4-step wizard (create/edit/publish), validation, role-based access |
| `e2e/question-bank.spec.ts` | Question list, difficulty badge, search, empty state, new-question nav |
| `e2e/question-editor.spec.ts` | New question form, type selector, edit pre-population, answer options |
| `e2e/categories.spec.ts` | Category list, create modal |
| `e2e/tags.spec.ts` | Tag list, usage counts, search, create button |
| `e2e/branding.spec.ts` | Branding form fields, save button |
| `e2e/my-results.spec.ts` | Employee results history page |
| `e2e/admin-grading.spec.ts` | Grading queue, detail page, question navigation, score validation, feedback, submit |
| `e2e/grading/ai-grading.spec.ts` | AI-graded badge, reasoning collapsible |
| `e2e/accessibility.spec.ts` | WCAG 2.1 AA: skip link, keyboard navigation |
| `e2e/loyalty-narrative.spec.ts` | Values Profile AI narrative generation |

### Coverage by Workflow

| # | Workflow | Status | Notes |
|---|----------|--------|-------|
| 1 | User Login | COVERED | `auth.spec.ts`, `full-walkthrough.spec.ts` (01–02) |
| 2 | Change Password | COVERED | `full-walkthrough.spec.ts` (19) |
| 3 | View Profile | PARTIAL | User data mocked via fixture; no explicit profile UI test |
| 4 | Browse Available Exams | COVERED | `employee-portal.spec.ts` (01–02) |
| 5 | Start an Exam | COVERED | `employee-portal.spec.ts` (05) |
| 6 | Take an Exam | COVERED | `exam-taking.spec.ts` (11 tests, all question types) |
| 7 | Handle Tab Switch Warning | COVERED | `exam-taking.spec.ts` (11) |
| 8 | View Exam Results | COVERED | `exam-result.spec.ts` (5 tests), `my-results.spec.ts` |
| 9 | List & Filter Users | COVERED | `full-walkthrough.spec.ts` (04) |
| 10 | Create a User | COVERED | `full-walkthrough.spec.ts` (04, 17) |
| 11 | Edit a User | **NOT COVERED** | ✅ Added in `user-management.spec.ts` |
| 12 | Reset User Password | **NOT COVERED** | ✅ Added in `user-management.spec.ts` |
| 13 | Deactivate / Reactivate User | **NOT COVERED** | ✅ Added in `user-management.spec.ts` |
| 14 | Bulk Import Users | **NOT COVERED** | ✅ Added in `user-management.spec.ts` |
| 15 | Browse Question Bank | COVERED | `question-bank.spec.ts`, `full-walkthrough.spec.ts` (08) |
| 16 | Create / Edit Question | COVERED | `question-editor.spec.ts`, `full-walkthrough.spec.ts` (09) |
| 17 | Delete / Archive Question | **NOT COVERED** | ✅ Added in `question-management.spec.ts` |
| 18 | Generate Questions with AI | **NOT COVERED** | ✅ Added in `question-management.spec.ts` |
| 19 | Import Questions | **NOT COVERED** | ✅ Added in `question-management.spec.ts` |
| 20 | Export Questions | **NOT COVERED** | ✅ Added in `question-management.spec.ts` |
| 21 | Manage Categories | COVERED | `categories.spec.ts`, `full-walkthrough.spec.ts` (06) |
| 22 | Manage Tags | COVERED | `tags.spec.ts`, `full-walkthrough.spec.ts` (07) |
| 23 | Create an Exam (wizard) | COVERED | `exam-wizard.spec.ts` (14 tests) |
| 24 | Edit an Exam | COVERED | `exam-wizard.spec.ts` (edit suite) |
| 25 | Publish an Exam | COVERED | `exam-wizard.spec.ts` (publish suite) |
| 26 | Archive an Exam | **NOT COVERED** | ✅ Added in `exam-lifecycle.spec.ts` |
| 27 | List Exams | COVERED | `full-walkthrough.spec.ts` (10) |
| 28 | View Grading Queue | COVERED | `admin-grading.spec.ts` (01) |
| 29 | Grade a Session | COVERED | `admin-grading.spec.ts` (02–06) |
| 30 | View Admin Dashboard | COVERED | `full-walkthrough.spec.ts` (03, 15) |
| 31 | View Exam Analytics | COVERED | `full-walkthrough.spec.ts` (20) |
| 32 | View Employee Record | COVERED | `full-walkthrough.spec.ts` (18), `loyalty-narrative.spec.ts` |
| 33 | View Audit Log | COVERED | `full-walkthrough.spec.ts` (13) |
| 34 | Configure Branding | COVERED | `branding.spec.ts`, `full-walkthrough.spec.ts` (14) |

### Coverage Summary

| Status | Count | % |
|--------|-------|---|
| COVERED | 25 | 73.5% |
| PARTIAL | 1 | 2.9% |
| NOT COVERED → now added | 8 | 23.5% |
| **After new tests** | **34** | **~100%** |

### New Test Files Added

| File | Workflows Covered |
|------|------------------|
| `e2e/user-management.spec.ts` | Edit User (11), Reset Password (12), Deactivate/Reactivate (13), Bulk Import (14) |
| `e2e/question-management.spec.ts` | Delete Question (17), Archive Question (17), AI Generate (18), Import Questions (19), Export Questions (20) |
| `e2e/exam-lifecycle.spec.ts` | Archive Exam (26) |
