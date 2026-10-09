# Issue Reports Index

| ID | Title | Severity | Layer | Module | Status | Resolved |
|----|-------|----------|-------|--------|--------|---------|
| [ISS-171](ISS-171-self-change-revokes-tokens.md) | Self change-password does not revoke older access tokens | medium | backend | auth | resolved | 2026-10-09 |
| [ISS-131](ISS-131-e2e-fixme-and-silent-skips.md) | e2e specs still test.fixme / silently skip (password-reset, tab-switch, certificates) | low | frontend | e2e | resolved | 2026-10-09 |
| [ISS-141](ISS-141-uuid-params-422.md) | Malformed UUID id params (query and path) reach Postgres and return 500 | medium | backend | users | resolved | 2026-10-09 |
| [ISS-102](ISS-102-e2e-specs-uncovered-flows.md) | Add e2e specs for uncovered flows (issue #16) | low | frontend | e2e | resolved | 2026-10-09 |
| [ISS-001](ISS-001-stale-dist-usematches-crash.md) | Stale dist `useMatches` crash on page load | high | frontend | auth | resolved | 2026-05-16 |
| [ISS-002](ISS-002-e2e-walkthrough-false-positive-passes.md) | E2E walkthrough false-positive passes | high | frontend | auth | resolved | 2026-05-16 |
| [ISS-003](ISS-003-login-no-show-password-toggle.md) | Login form has no show/hide password toggle | medium | frontend | auth | resolved | 2026-05-18 |
| [ISS-004](ISS-004-autojob-audit-exams-no-tenant.md) | Auto-submit audit INSERT references non-existent exams.tenant_id column | high | backend | sessions | resolved | 2026-05-18 |
| [ISS-005](ISS-005-question-create-422-payload-mismatch.md) | POST /api/v1/questions returns 422 — flat stem/body instead of nested translations | high | frontend | questions | resolved | 2026-05-18 |
| [ISS-006](ISS-006-question-save-422.md) | New question save returns 422 when form filled in non-English locale | high | frontend | questions | resolved | 2026-05-18 |
| [ISS-007](ISS-007-e2e-locale-selector-mismatch.md) | E2E locale selector mismatch | medium | frontend | auth | resolved | 2026-05-18 |
| [ISS-008](ISS-008-e2e-seeddata-pagination-editor-navigation.md) | getSeedData pagination miss + question-editor unreliable navigation | high | frontend | questions | resolved | 2026-05-19 |
| [ISS-009](ISS-009-e2e-token-injection-checkbox-toggle.md) | E2E token injection + checkbox toggle | medium | frontend | auth | resolved | 2026-05-19 |
| [ISS-010](ISS-010-exam-wizard-step3-save-errors.md) | Exam wizard step 3 — sections URL empty ID, assignment 422, i18n type label | high | frontend, backend | exams | resolved | 2026-05-19 |
| [ISS-011](ISS-011-exam-publish-422.md) | Publish exam returns 422 — "0 available" because questions are in draft status | medium | frontend | exams | resolved | 2026-05-19 |
| [ISS-012](ISS-012-question-status-400-wrong-json-key.md) | Question status transition returns 400 — frontend sends wrong JSON key | high | frontend | questions | resolved | 2026-05-19 |
| [ISS-013](ISS-013-question-edit-option-text-blank.md) | Question edit page shows blank answer option text (пусто) for all locales | high | frontend | questions | resolved | 2026-05-19 |
| [ISS-014](ISS-014-axe-accessibility-violations-admin.md) | Axe violations — missing main landmark, h1, region, skip-link on admin/auth pages | medium | frontend | auth | resolved | 2026-05-19 |
| [ISS-015](ISS-015-auth-token-expiry-publish-button-ux.md) | Access token expiry causes cascading 401s; publish button not disabled after validation failure | high | frontend | auth, exams | resolved | 2026-05-19 |
| [ISS-016](ISS-016-save-button-archives-active-question.md) | Save on active question archives it — frontend stays on stale URL, subsequent saves fail | high | frontend, backend | questions | resolved | 2026-05-19 |
| [ISS-017](ISS-017-question-tags-not-saved.md) | Question tag field cleared after save — tags not persisted | high | backend | questions | resolved | 2026-05-19 |
| [ISS-018](ISS-018-create-user-no-department-role-options.md) | Create/Edit User drawers show no Department or Role options | high | frontend | users | resolved | 2026-05-19 |
| [ISS-019](ISS-019-requirerole-currentuser-gc-redirect.md) | RequireRole redirects to /login after ~5 min — currentUser GC'd from React Query cache | high | frontend | auth | resolved | 2026-05-21 |
| [ISS-020](ISS-020-password-preview-missing.md) | Password preview toggle missing on login form — stale frontend Docker image | medium | config | auth | resolved | 2026-05-22 |
| [ISS-021](ISS-021-usematches-data-router-error.md) | useMatches data-router error at runtime — stale frontend Docker image (never rebuilt) | high | config | auth | open | null |
| [ISS-022](ISS-022-reports-page-empty.md) | Reports page at /admin/reports appears empty — stale frontend Docker image | medium | config | reports | resolved | 2026-05-22 |
| [ISS-023](ISS-023-docker-db-port-mismatch.md) | make dev fails when DB_PORT is set to host-mapped 5442 | high | config | auth | resolved | 2026-05-25 |
| [ISS-024](ISS-024-audit-log-loading-stuck.md) | Audit log page remains stuck in loading state due to unstable default from filter | high | frontend | audit | resolved | 2026-05-25 |
| [ISS-025](ISS-025-change-password-no-show-hide-toggle.md) | ChangePasswordForm — all three password fields have no show/hide toggle | medium | frontend | auth | resolved | 2026-06-08 |
| [ISS-026](ISS-026-change-password-autofill-stale-credential.md) | ChangePasswordPage — stale browser autofill in current-password causes INVALID_CREDENTIALS | high | frontend | auth | resolved | 2026-06-08 |
| [ISS-027](ISS-027-portal-blank-page-employee.md) | Employee portal blank page — TDZ crash in useCountdown for expired exams + no error boundary | high | frontend | portal | resolved | 2026-06-08 |
| [ISS-036](ISS-036-uat-defect-exam-taking-no-saved-indicator.md) | UAT Defect: Employee Exam Taking — Auto-save 'Saved ✓' indicator not displayed after answer selection | medium | frontend | exam-taking | resolved | 2026-06-09 |
| [ISS-038](ISS-038-uat-defect-portal-card-wrong-status-open-second-attempt.md) | UAT Defect: Portal card shows 'Passed'/'View result' when in-progress second attempt exists | high | frontend | employee-portal | resolved | 2026-06-09 |
| [ISS-039](ISS-039-uat-defect-retake-exam-button-broken-route.md) | UAT Defect: 'Retake Exam' button navigates to non-existent route | high | frontend | exam-taking | resolved | 2026-06-09 |
| [ISS-040](ISS-040-uat-defect-manual-grading-api-500.md) | UAT Defect: Manual Grading — POST grading answer endpoint returns HTTP 500 | high | backend | grading | resolved | 2026-06-09 |
| [ISS-041](ISS-041-uat-defect-manual-grading-question-sort-order.md) | UAT Defect: Manual Grading — Questions displayed in wrong order on grading detail page | low | backend | grading | resolved | 2026-06-09 |
| [ISS-042](ISS-042-uat-defect-manual-grading-employee-403-redirect.md) | UAT Defect: Manual Grading — Employee redirected to blank /admin page instead of 403 message | low | frontend | grading | resolved | 2026-06-09 |
| [ISS-043](ISS-043-uat-defect-manual-grading-pending-result-page.md) | UAT Defect: Manual Grading — Result page shows 0%/Failed for grading_pending session | high | frontend | grading | resolved | 2026-06-09 |
| [ISS-044](ISS-044-uat-defect-manual-grading-retake-button-visible-pending.md) | UAT Defect: Manual Grading — Retake Exam button visible while session is grading_pending | medium | frontend | grading | resolved | 2026-06-09 |
| [ISS-045](ISS-045-uat-defect-manual-grading-feedback-not-shown-employee.md) | UAT Defect: Manual Grading — Examiner manual_feedback not displayed on employee result screen | medium | frontend+backend | grading | resolved | 2026-06-09 |
| [ISS-046](ISS-046-uat-defect-per-section-scores-non-empty-flat-exam.md) | UAT Defect: Result & Certification — per_section_scores non-empty for flat exams | medium | both | results | resolved | 2026-06-09 |
| [ISS-047](ISS-047-uat-defect-error-states-not-rendered-results-pages.md) | UAT Defect: Result & Certification — error states not rendered on My Results and Result Detail pages | low | frontend | results | resolved | 2026-06-09 |
| [ISS-048](ISS-048-uat-defect-employee-record-missing-export-button.md) | UAT Defect: Analytics & Reporting — Employee Record page missing CSV Export button | medium | frontend | analytics / employee record | resolved | 2026-06-09 |
| [ISS-049](ISS-049-uat-defect-exam-analytics-score-display-inflated.md) | UAT Defect: Analytics & Reporting — Exam Analytics average/median score inflated (×100 double multiply) | high | frontend | analytics / per-exam | resolved | 2026-06-09 |
| [ISS-050](ISS-050-seed-script-blocked-live-test-env.md) | seed-test-env.ts blocked by rate limiting, wrong question type/field names, invalid likert_polarity | medium | config | questions | resolved | 2026-09-01 |
| [ISS-051](ISS-051-frontend-lint-config-coverage.md) | Frontend lint cannot parse TypeScript; low coverage; unlocalized strings and a11y warnings | medium | frontend | tenant | resolved | 2026-10-09 |
| [ISS-052](ISS-052-admin-audit-reports-route-guards.md) | /admin/audit and /admin/reports route guards disagree with RBAC | medium | frontend | audit / reports | resolved | 2026-10-09 |
| [ISS-053](ISS-053-users-list-department-tree-filter.md) | Users List department filter is a flat select that ignores the department tree | low | frontend | users | resolved | 2026-10-09 |
| [ISS-054](ISS-054-auditlogtable-react-key-warning.md) | AuditLogTable logs React "unique key" warning (bare fragment in map) | low | frontend | audit | resolved | 2026-10-09 |
| [ISS-055](ISS-055-export-buttons-bypass-download-helper.md) | CSV export buttons bypass shared download helper (no 401 refresh / error display) | medium | frontend | reports | resolved | 2026-10-09 |
| [ISS-060](ISS-060-frontend-tests-grading-results-employee-step4.md) | No frontend tests for grading UI, result components, employee record, Step 4 eligible counts | low | frontend | exams | resolved | 2026-10-09 |
| [ISS-059](ISS-059-e2e-portal-redirect-wrong-assertion.md) | e2e full-walkthrough test 21 asserts wrong redirect for admin on /portal | low | frontend | e2e | resolved | 2026-10-09 |
| [ISS-82](ISS-82-tenant-id-audit.md) | AI insights/loyalty SQL references nonexistent exams.tenant_id, session_questions.order_num, exams.category_id (schema audit) | high | backend | ai | resolved | 2026-10-09 |
| [ISS-191](ISS-191-csv-formula-injection.md) | CSV formula injection guard on all CSV exports; results CSV auto_submitted + unknown exam 404 (#178) | high | backend | reports | resolved | 2026-10-09 |
| [ISS-165](ISS-165-dept-admin-report-scoping.md) | department_admin not scoped to its department on reports/analytics/dashboard endpoints | high | backend | reports | resolved | 2026-10-09 |
| [ISS-160](ISS-160-backend-enforce-password-change.md) | Backend does not enforce force_password_change (SPA-only) | high | backend | auth | resolved | 2026-10-09 |
| [ISS-176](ISS-176-api-inconsistencies.md) | API inconsistencies: VALIDATION_ERROR 422, import body cap, DISABLE_RATE_LIMIT answer-save | medium | backend | auth | resolved | 2026-10-09 |
| [ISS-181](ISS-181-unique-email-index-guarded.md) | UNIQUE INDEX on lower(email) that can never break startup (migration 035) + duplicate-twin admin report (#181) | medium | database | users | resolved | 2026-10-09 |
| [ISS-240](ISS-240-users-authz-d4.md) | Users authz hardening D-4: strict subset, no self role/deactivate, stale JWT claims, 404 parity, malformed ids, ambiguous import dept | high | backend | users | resolved | 2026-10-09 |
| [ISS-218](ISS-218-deptscope-all-roles.md) | Department scoping must apply to every role except super_admin (custom roles/examiner saw org-wide data) | high | backend | reports | resolved | 2026-10-09 |
