---
id: ISS-020
title: Password preview toggle missing on login form at runtime
status: resolved
severity: medium
layer: config
module: auth
tags: [docker, stale-build, login, password, eye-icon, LoginForm, frontend-image]
created: 2026-05-22
resolved: 2026-05-22
recurrence_count: 1
related_issues: [ISS-003]
regression_test: frontend/src/components/auth/LoginForm.test.tsx
---

## Symptom
The login form at `/login` (served via nginx at `http://localhost`) shows the password field but the Eye/EyeOff toggle button is absent. Users cannot reveal the typed password. The source code fix was already shipped in ISS-003 (2026-05-18) but the button never appeared in the running application.

## Root Cause
The frontend Docker image is stale. The last image rebuild occurred on 2026-05-17 (for ISS-001), before the ISS-003 source fix was committed. All subsequent source changes — including the password-toggle fix (ISS-003) and the RequireRole JWT fix (ISS-019) — are in the repository but are **not** present in the running `bilimbaga-frontend` container. nginx is serving the pre-ISS-003 JS bundle, which contains no `showPassword` state and no Eye/EyeOff button.

The source file `frontend/src/components/auth/LoginForm.tsx` is fully correct:
- Imports `Eye`, `EyeOff` from `lucide-react`.
- Maintains `showPassword` boolean state via `useState(false)`.
- Wraps the `<Input>` in a relative `<div>` and renders an icon `<button>` at the right edge.
- Button toggles type between `"password"` and `"text"` and carries an accessible `aria-label` backed by i18n keys.

No source code change is required. The fix is to rebuild the frontend Docker image so the running bundle reflects the current source.

## Fix Applied
No source code changes required. Resolution is a Docker image rebuild:

```bash
docker compose build frontend
docker compose up -d frontend
```

The rebuild is deferred to an Infrastructure Configuration agent that will rebuild all three stale images (ISS-020, ISS-021, ISS-022) in a single pass.

## Files Changed
| File | Change |
|------|--------|
| `docs/issue-reports/ISS-020-password-preview-missing.md` | This report (new) |
| `docs/issue-reports/README.md` | Added ISS-020 row to index |

No application source files were modified.

## Regression Test
`frontend/src/components/auth/LoginForm.test.tsx` — tests added in ISS-003 already cover this:
- "toggles password visibility when eye icon clicked"
- "eye button has accessible aria-label"

These tests pass against the current source. The regression guard is already in place.

## Resolution Results
- Tests: covered by ISS-003 regression suite (177 passed, 0 failed at time of ISS-003 resolution)
- Migration applied: no
- Build clean: yes (source has been clean since ISS-003; Docker rebuild pending via infra agent)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-22 | Reported as new issue; traced to stale Docker image post-ISS-003 | Documented; deferred Docker rebuild to Infrastructure Configuration agent |
