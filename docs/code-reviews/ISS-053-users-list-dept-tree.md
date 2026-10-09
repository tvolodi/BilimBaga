# Code Review: ISS-053 Users List department tree filter

Result: PASS

Scope: frontend/src/pages/admin/users/UsersListPage.tsx, frontend/src/pages/admin/users/UsersListPage.test.tsx, docs/issue-reports/ISS-053-users-list-department-tree-filter.md, docs/issue-reports/README.md (index row).

## Findings
- [Medium] DepartmentTreeSelect.tsx (pre-existing, not in diff) — the clear control is an `<X>` icon with an onClick nested inside the trigger `<Button>`. It is not keyboard-reachable on its own. Out of scope for this change; consider a follow-up.
- [Low] UsersListPage.test.tsx — `vitest --maxWorkers=2` fails in this repo ("minThreads and maxThreads must not conflict", config issue). Running the single file without the flag works. Not caused by this change.

No Critical or High findings.

## Checklist
- No direct fetch; departments come via the `useDepartments` React Query hook inside the shared component. OK.
- Removed the unused `useDepartments` import and call. `Select` is still used by the other filters. OK.
- No new strings. Placeholder and aria-label reuse the existing key `users.filters.department`. OK.
- Loading, error and empty states are handled by DepartmentTreeSelect. OK.
- Filter wiring: `value={filters.department_id ?? null}`, `onChange` maps null to '' and so clears the param, and page resets to 1 via `setFilter`. OK.
- Tests: 3 tests cover the tree combobox, nested child selection sending `department_id`, and clear. Run on that single file: 3/3 pass. `tsc --noEmit` and eslint on the changed files are clean.

## AC Coverage
- FR-BB317.4 (Users List uses DepartmentTreeSelect): covered
- Child departments selectable (the bug itself): covered (test selects nested "Engineering")
- Clearing the filter: covered

Summary: Minimal, correct swap of the flat select for DepartmentTreeSelect, with regression tests that pass and no Critical or High findings.
