# Code Review: ISS-131 (e2e un-fixme / no silent skips)

Reviewer: Code Reviewer subagent. Run ID: ISS-131. Nothing was fixed or executed (no e2e, docker or playwright run).

Result: PASS

## Findings

- [Medium] frontend/e2e/certificates.spec.ts:23 - target guard: `const API = process.env.E2E_API_URL || ...` is read raw and not passed through `requireTarget`, and the new test uses it for the public verify `fetch`. It is a GET only, and the same run's global-setup/seed (`BASE` via `requireTarget`) throws first on the frozen host, so there is no real exposure. The guard is not weakened, but this adds a second unguarded URL read. The same pattern already exists in account-recovery.spec.ts, downloads-bearer.spec.ts, exam-lifecycle.spec.ts and loyalty-narrative.spec.ts. -> Export `BASE` from seed.ts (or call `requireTarget` here) instead of duplicating the env read.
- [Medium] frontend/e2e/fixtures/seed.ts (`startEmployeeSession`) - handles only `SESSION_ALREADY_OPEN`. That matches the backend (sessions/handler.go:56) and the throw-on-failure behaviour is right, but the older `startSession` in the same file also accepts `sessionAlreadyOpen`. -> Cosmetic only; no change needed.
- [Medium] frontend/e2e/certificates.spec.ts and tab-switch.spec.ts cleanup - `deleteTestExam` and `deleteTestQuestion` do not check the HTTP status, so `.catch(() => undefined)` only covers network errors. An exam with sessions or a certificate will likely stay behind, since it cannot be deleted. Titles are unique per run (`Date.now()`), so there are no collisions, only residue. -> Acceptable and documented in the code comments. Optionally log non-2xx.
- [Medium] frontend/e2e/password-reset.spec.ts - overlap with account-recovery.spec.ts: the forgot-form neutral confirmation is covered live there (via Mailhog) and by stub here. The overlap is limited to the confirmation text. The new file's value is the Authorization-header and request-body contract, the login-link entry point and the invalid/missing-token states. The happy-path reset was dropped, so duplication is acceptable.
- [Low] frontend/e2e/certificates.spec.ts - the new test depends on global-setup (`getSeedData` and `employee.json`), so it is meaningful only in the live config. certificates.spec.ts is in the `chromium-live-admin` project of playwright.live.config.ts, so it runs there. In the default `playwright.config.ts` it would fail loudly (no `.auth` files), not skip. That is consistent with the ISS-131 goal.

## Verification against real code

- Password-reset selectors and copy (`src/pages/auth/ForgotPasswordPage.tsx`, `ResetPasswordPage.tsx`, `LoginForm.tsx:105`, `locales/en.json auth.recovery.*`):
  - Link "Forgot password?" matches `/forgot password\?/i`.
  - Email label "Email" has a single match, and the button "Send reset link" matches.
  - Confirmation is `role="status"` with "If an account exists ...".
  - "New password" and "Confirm new password" labels match. The show-password buttons are labelled "Show password", so `/^new password/i` and `/confirm new password/i` are not ambiguous.
  - Button "Reset password" matches `/^reset password$/i`.
  - The invalid-token state renders `role="alert"` ("This reset link is invalid or has expired.") plus a link "Request a new reset link" to `/forgot-password`. A missing token takes the same branch immediately, as the new test asserts.
  - `src/api/recovery.ts` uses plain fetch with no Authorization header, so `authorization` is undefined as the test asserts.
  - `i18n.ts` reads `i18n-lang` at load, so the `addInitScript` pin to `en` is correct. Contexts are closed in `finally`, and there are no fixed sleeps.
- Certificate download flow:
  - `ResultActions.tsx` shows the `result.download_certificate` button only when passed and `certificateEnabled`. ru is "Скачать сертификат" and en is "Download Certificate", so the regex matches.
  - `downloadFile` hits `/api/v1/portal/sessions/:id/certificate`.
  - The backend sets `Content-Disposition: attachment; filename="certificate-<code>.pdf"` and `Content-Type: application/pdf` (certificates/handler.go:112), so the regex extraction is correct.
  - The verify endpoint returns `{data:{valid, exam_title,...}}`, and the service validates the code as a UUID.
  - `waitForResponse` is armed together with the click in `Promise.all`, so there is no race.
- Seed API contracts:
  - `max_attempts` only needs to be > 0 (500 is fine). `on_tab_switch` accepts log, warn or submit. `certificate_enabled` and `passing_score_pct` are passed through.
  - The order add rule, publish, assign is valid, since `CreateAssignment` accepts active or draft exams.
  - Session start returns `session_id` and `questions[].options[].{id,text}` (sessions/model.go).
  - The answer PUT body (`selected_option_ids`, `text_answer`, `time_spent_seconds`) matches `SaveAnswerInput`.
  - `createTestQuestion` makes "Option A" the correct option, and a single-choice question is auto-graded, so score 100% passes the 50% threshold.
  - `startEmployeeSession` resumes an open session via `open_session_id`. The tab-switch tests never submit a real session (the auto-submit result is stubbed), so resumption works and attempts (500) are not exhausted.
- No `test.skip` or `test.fixme` and no `waitForTimeout` remain in the three specs. Failures to open a session throw.
- The frozen host `bilimbaga-test.ai-dala.com` is not referenced anywhere in the changed files. seed.ts still resolves `BASE` and `APP_URL` through `requireTarget`, so the target-guard is unchanged.
- Employee token cache (10 min vs 15 min JWT TTL) avoids auth rate limits and is sound.

## AC Coverage (GitHub #131)

- AC-1 password-reset.spec.ts un-fixmed, selectors confirmed against shipped components and locales: covered
- AC-2 tab-switch uses a dedicated exam per run and fails instead of skipping: covered
- AC-3 certificate PDF test via a seeded certificate-enabled exam, replacing the fixme: covered
- AC-4 not executed against a live stack: the author states NOT EXECUTED, and UAT must run the specs.

Summary: No Critical or High findings. Selectors, labels and API contracts match the real components and backend, there are no sleeps or silent skips, and the target-guard is intact. Only Medium and Low hygiene notes remain.
