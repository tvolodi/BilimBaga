# BA Drift Check (branch swarm/34-gap-drift)

Sampled 24 README rows marked Implemented against code: FR-BB43, 44, 45, 46, 47, 51, 52, 53, 55/114, 58, 59, 73, 74, 315, 316, 317, 318, 310, 38, 66, 62, 110, 111 (spot checks of routes, components, key ACs; test presence noted). Rows verified OK: 43, 44, 47 (minor: unauthorised role is redirected, not shown a 403 message), 52, 53, 73, 74, 310, 315, 316, 318, 38, 66, 62, 111.

Downgraded to `Partial` in README:

## Root cause shared by FR-BB45, 46, 58, 59
Backend `auth.Authenticate` accepts only `Authorization: Bearer`. Several download helpers use raw `fetch` with `credentials: 'include'` and no Bearer header (or an `<a href>` navigation), so the request is 401 and the download silently fails (also violates the "no raw fetch / apiFetch only" convention).
- `frontend/src/components/results/ResultActions.tsx` (`fetch` portal certificate, no auth, `if (!res.ok) return`)
- `frontend/src/components/results/ResultsTable.tsx` (`<a href=/api/v1/portal/sessions/:id/certificate>`)
- `frontend/src/api/employees.ts` `downloadAdminCertificate`
- `frontend/src/api/reports.ts` `downloadDashboardPdf`, `downloadExamCsv`

## Per row
| Row | Unmet ACs |
|-----|-----------|
| FR-BB45 | AC-6 (certificate download from result screen fails with 401; no error shown) |
| FR-BB46 | AC-8 (certificate link navigates to an unauthenticated URL, download fails); AC-3 not verifiable end to end |
| FR-BB58 | AC-4 (admin certificate download sends no Bearer token; error code handling unreachable in practice) |
| FR-BB59 | AC-3 and AC-7 (PDF/CSV export helpers send no Bearer token); AC-2 (non-allowed roles are redirected to /admin or /portal, not /login; `department_admin` holds `reports:read` but is excluded from the route guard) |
| FR-BB51 | AC-2 (`completion_rate_by_exam` inner-joins assignments, so an active exam with no assignments is omitted instead of listed with zero counts) |
| FR-BB114 | AC-1 (route guard is `super_admin, hr_admin, examiner`; spec and RBAC seed (`audit:read`) are super_admin only, so examiners reach a page whose API returns 403; also `hr_admin` role is not seeded) |
| FR-BB317 | Users List department filter (`frontend/src/pages/admin/users/UsersListPage.tsx`) still uses a flat `<select>` and ignores the tree; Create/Edit drawers and Step 3 use `DepartmentTreeSelect` |

## Test-coverage observations (not status-changing)
No test files for: grading components/pages (FR-BB47), result components (FR-BB45), employee record components (FR-BB58), ExamWizard Step4 eligible counts (FR-BB315). Project policy requires tests per implementation.

## Dev issues to file (one line each)
1. fix(frontend): route all authenticated file downloads (portal/admin certificate, dashboard PDF, exam CSV) through a shared Bearer-aware `downloadFile` helper; surface errors via toast; add tests (FR-BB45/46/58/59).
2. fix(reports): `GetCompletionRateByExam` must list every active exam, using LEFT JOIN so unassigned exams return zeros; add test (FR-BB51 AC-2).
3. fix(frontend): align `/admin/audit` guard to `super_admin` only (FR-BB114 AC-1) and align `/admin/reports` guard/redirect with FR-BB59 AC-2 (decide on `department_admin`).
4. feat(frontend): use `DepartmentTreeSelect` in the Users List department filter (FR-BB317).
5. test(frontend): add tests for grading UI, result components, employee record, Step4 eligible counts.
6. feat: implement FR-BB116 (profile page, `PATCH /users/me`, persisted preferred_locale).
