# Code Review: ISS-055 (GitHub #66) - export buttons routed through downloadFile

Result: PASS

## Scope
frontend/src/api/audit.ts, api/employees.ts, api/download.test.ts, components/analytics/ExportCSVButton.tsx (+ test),
pages/admin/AuditLogPage.tsx (+ test), pages/admin/EmployeeRecordPage.tsx.

## Findings
- [Medium] frontend/src/components/analytics/ExportCSVButton.tsx:~30 - `<Button>` inside the new wrapper `<div>` is not re-indented (formatting only).
- [Medium] frontend/src/pages/admin/EmployeeRecordPage.tsx:~185 - new error alert has no component test (AuditLogPage and ExportCSVButton do). exportEmployeeRecord endpoint/Bearer is covered in download.test.ts.
- [Medium] Behaviour change: Content-Disposition filename is no longer used; fallback names always apply (documented in the issue report, consistent with helper).
- [Low] The alert markup is duplicated in 3 places; a small shared component would help later.

No Critical or High findings:
- No raw fetch left in the three export paths; all go through downloadFile (Bearer, one 401 refresh+retry, URL revoked).
- Callers have catch + inline role="alert" using t(downloadErrorKey(err)); keys download.failed / download.session_expired exist in en.json (and kk/ru per issue report, check:i18n reported green).
- No `any`, no console.log, no dead imports; loading state reset in finally; error cleared on retry.
- No new API module, so the E2E Authorization-header rule is not triggered; unit tests assert the Bearer header.

## AC Coverage (from issue report)
- Exports use downloadFile with 401 refresh: covered
- Inline translated error display at all 3 call sites: covered (EmployeeRecordPage untested, see Medium)
- Tests updated/added: covered

Summary: Clean migration to the shared download helper with proper error display; only Medium/Low polish items.
