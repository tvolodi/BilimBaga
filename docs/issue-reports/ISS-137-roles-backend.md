# ISS-137 - FR-BB117 role management backend

Scope: migration 034, `internal/roles` package (CRUD, permission catalogue), `GET /users/me` permissions, default-deny scoping audit.

## Scoping audit (role-name switches in backend/internal)
| Location | Before | After |
|---|---|---|
| users.ListUsers | `department_admin` only scoped; any other role (custom) org-wide, and department_admin with empty dept org-wide | only `super_admin` org-wide; others scoped to own dept; no dept => empty result |
| users.GetUser | `department_admin` same-dept; default self only | self, or same-dept if role is department_admin or holds `users:read` (via permission checker); default-deny |
| users.Create/Update/Deactivate/ResetPassword/Unlock | scoping only for `department_admin` | scoped for every non-super_admin role; Update also forbids moving a user out of the caller's dept |
| users.ImportUsers | dept scope only for `department_admin`; no super_admin-assignment guard | scoped for all non-super_admin; non-super callers cannot import `super_admin` rows (pre-existing escalation hole) |
| exams.CreateAssignment/DeleteAssignment | dept limit only for `department_admin` | org-wide only for `super_admin` and `examiner`; everyone else dept-limited |
| ai.GetLoyaltyNarrative | `!= super_admin` dept check | unchanged (already default-deny) |
| reports/audit/questions/sessions | no role-name branching (permission-gated only) | unchanged |

Known residual: exam assignments of type `user` are not department-checked for any role (pre-existing).

## Verification
`go build ./...`, `go vet ./...`, `go test -p 1 ./...` green (no live Postgres; needs-live-db).
