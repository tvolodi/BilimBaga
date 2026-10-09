# Code Review: ISS-054 AuditLogTable React key warning (GitHub #58)

Result: PASS

Scope: uncommitted changes: `frontend/src/components/audit/AuditLogTable.tsx`, new `AuditLogTable.test.tsx`, `docs/issue-reports/ISS-054-auditlogtable-react-key-warning.md`, README row.

## Findings
- [Critical] none
- [High] none
- [Medium] none
- [Low] Test asserts `errorSpy` not called at all; this could become flaky if an unrelated console.error appears (e.g. future i18n/router warnings). Acceptable now, the specific "unique key" filter assertion already isolates the target.
- [Low] The empty-state test only checks the "—" placeholder, which is pre-existing hard-coded text (not introduced by this change).

## Checks
- Fix is correct: key moved to the outermost element of the map callback (`<Fragment key={entry.id}>`); redundant inner keys removed. Behaviour and markup unchanged. `Fragment` imported from react.
- No other bare `<>` inside a `.map` found in the same pattern (other `<>` hits in CategoryTreeNode/DepartmentTreeNode are conditional renders, not list items).
- Regression test renders 4 entries, expands 2 (main + meta rows), asserts no "unique key" console.error; uses the real i18n and MemoryRouter. Reasonable and targeted.
- No new user-visible strings, no API/RBAC/backend impact.
- Issue report follows the ISS template; README row present and consistent. Heavy suites not rerun per instructions (report claims 336 passing, tsc and eslint clean).

Summary: Minimal, correct fix with a meaningful regression test; no required changes.
