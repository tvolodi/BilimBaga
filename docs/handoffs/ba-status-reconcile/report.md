# README Status Reconciliation

All rows below were set to `Implemented` in docs/requirements/README.md (case normalised).

Evidence summary (code + tests present in worktree):
- FR-BB11: Makefile, docker-compose.yml, deploy/nginx.conf, backend/cmd/api, frontend scaffold; doc status already `implemented`.
- FR-BB113: frontend/src/pages/admin/departments/* with 5 test files; route /admin/departments.
- FR-BB114, FR-BB55: pages/admin/AuditLogPage.tsx + test; components/audit/*; routes /admin/audit.
- FR-BB26: pages/admin/questions/QuestionEditorPage.tsx + test (doc uat-verified).
- FR-BB27: QuestionBankPage.tsx + test (doc uat-verified).
- FR-BB28/29: categories/ and tags/ pages with tests.
- FR-BB39, 310: sessions/handler.go (SubmitSession), sessions/autojob.go + autojob_test.go.
- FR-BB313: pages/EmployeePortal/* + test.
- FR-BB314: pages/ExamTaking/* + __tests__ (doc uat-verified).
- FR-BB42, 47: sessions grading handlers (/admin/grading*), pages/admin/Grading*Page.tsx, components/grading/*.
- FR-BB43, 44: backend/internal/certificates (service, handler, pdf.go + tests).
- FR-BB45: pages/ResultPage.tsx, components/results/*.
- FR-BB51, 52, 53: reports/handler.go routes /admin/dashboard, /admin/exams/{id}/analytics, /admin/users/{id}/record; reports tests.
- FR-BB56, 57, 58, 59: AdminDashboardPage, ExamAnalyticsPage, EmployeeRecordPage, ReportsPage (+tests).
- FR-BB62, 67, 71: doc statuses already implemented; locales kk/ru/en present; ai package + tests.
- FR-BB73: sessions/grading.go AI grading + grading_ai_test.go (doc Implemented).
- FR-BB74: ai handler /admin/ai/insights/{examId}, AIInsightsCard.test.tsx.
- FR-BB315: exams GetEligibleCounts endpoint + Step4Review useEligibleCounts.
- FR-BB316: LocaleSwitcher used in layouts/PortalLayout.tsx.
- FR-BB317: components/DepartmentTreeSelect.tsx used in Step3Assignments and user drawers.
- FR-BB318: exams Unpublish route.

Rows only case-normalised: FR-BB28, 29, 39, 310, 313, 42, 55, 56, 57, 62, 67, 71, 318 (`implemented` -> `Implemented`).
Rows Draft -> Implemented: FR-BB11, 113, 114, 26, 27, 314, 43, 44, 45, 47, 51, 52, 53, 58, 73, 74, 315, 316, 317, 59.

Left unchanged:
- FR-BB64: Validated (delta not built).
- FR-BB65: Draft; only partially built (indexes, tenant cache, pool, lazy routes, k6 script exist; no k6 summary report in docs/test-reports, no Lighthouse evidence).
