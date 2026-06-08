# Issue Reports Index

| ID | Title | Severity | Layer | Module | Status | Resolved |
|----|-------|----------|-------|--------|--------|---------|
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
