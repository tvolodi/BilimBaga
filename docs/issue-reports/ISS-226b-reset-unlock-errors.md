# ISS-226b: swallowed errors in admin users list

Reopened by UAT: `handleResetPassword` in `frontend/src/pages/admin/users/UsersListPage.tsx` had an empty catch, so a 403 FORBIDDEN (rank rule, PR #222) showed nothing.

## Handler audit

| Handler | File | Before | After |
|---|---|---|---|
| handleResetPassword | admin/users/UsersListPage.tsx | empty catch, silent | role="alert" notice: users.messages.forbidden on FORBIDDEN, else users.messages.error_generic |
| handleUnlock | admin/users/UsersListPage.tsx | any error -> unlock_error (403 not distinguished) | FORBIDDEN -> users.messages.forbidden, else unlock_error |
| deactivate | pages/users/DeactivateConfirmDialog.tsx | surfaces userErrorKey/generic | unchanged |
| create / edit drawers | pages/users/UserCreateDrawer, UserEditDrawer (admin versions re-export) | surface error | unchanged |
| import (preview/commit) | pages/users/ImportModal.tsx (admin re-exports) | surface error | unchanged |
| reset password | pages/users/UsersListPage.tsx | surfaces resetError | unchanged |
| delete / bulk / role change | no such handlers exist in these pages (role change is within the edit drawer) | n/a | n/a |

No other empty `catch {}` / `.catch(() => {})` in frontend/src/pages/users or admin/users. No new i18n keys needed.

## Tests
`npx vitest run --maxWorkers=1 src/pages/admin/users`: 11 passed (new: unlock 403, reset 403/500/success). `npm run check:i18n` passed. Full vitest and tsc not run (host memory alert).
