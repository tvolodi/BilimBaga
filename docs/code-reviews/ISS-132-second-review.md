# ISS-132 Second Review (PR #142, commit 686164b)

Scope: commit 2 ("disable Start outside window, refuse publishing empty exams"). Commit 1 was reviewed earlier.
Reviewed via `git show origin/swarm/132-exam-start:<path>`; nothing executed, no branch switched.

## Verdict: APPROVE (with 1 should-fix and 4 non-blocking notes)

No Critical/High defects. The should-fix is a stale/vacuous E2E test that now contradicts the new behaviour.

## Findings

### Verified OK
- **SQL columns**: `exams.available_from` / `available_until` are `TIMESTAMPTZ` (migration 012_exams.up.sql:21-22, CHECK from < until). Portal queries select `e.available_from, e.available_until` from `exams e`; no tenant_id referenced. Scan struct uses `*time.Time` with matching `db` tags.
- **Window semantics agree** with sessions/service.go:117-123 (`current.Before(from)` / `current.After(until)`, both strict, nil = open): frontend `now < from` -> notOpen, `now > until` -> closed. Boundary equality is open on both ends in both layers. UTC: Go serialises RFC3339 with offset, `new Date()` parses to an instant, so comparison is timezone-independent.
- **Status semantics**: `computeStatus` is NOT changed in this diff (the brief mentioned it; it is untouched), so 'not_started' / 'expired' / 'failed' semantics and existing portal tests are unaffected. Window gating is purely a frontend overlay on `not_started`. 'expired' (deadline) already hides the CTA; 'failed' means attempts exhausted so no retry path is wrongly left enabled; 'passed'/'in_progress' remain usable (resume after window close is not blocked by the card).
- **Contract**: `available_from`/`available_until` are additive nullable fields on list item and detail; TS types are optional/nullable. No removed or renamed fields.
- **Publish guard** `!e.Adaptive && len(rules)==0`: manual mode cannot bypass it, because manual question lists hang off a rule (`exam_question_rules.mode='manual'` + per-rule question table, 012_exams.up.sql:51-75); an exam with manual questions always has a rule. A manual rule with zero selected questions is already caught by the existing `ErrRulesUnsatisfied` path. Adaptive exams are exempt (consistent with the start-time guard `len(resolvedIDs)==0 && !cfg.Adaptive`). Guard runs before adaptive/rule-satisfaction checks and after the draft check; correctly wrapped with `%w`; handler maps to 422 `INSUFFICIENT_QUESTIONS` via `errors.Is`. Unit + handler tests added.
- **Error-code consistency**: `INSUFFICIENT_QUESTIONS` is shared between publish (admin) and start (portal); all 7 start codes + `generic` exist in en/ru/kk `portal.startError`; new `windowNotOpen`/`windowClosed` present in all three locales; placeholders `{{date}}` preserved.

### Should fix (not blocking merge)
1. `frontend/e2e/exam-wizard.spec.ts:122-148` "publish without rules succeeds and shows success feedback" now describes the opposite of the behaviour (publish returns 422). It still passes only because it asserts `not.toContainText(/unexpected error|failed to publish/i)`, which the new message ("The exam has no questions...") does not match, and the confirm step is conditional. Rename/rewrite to assert the refusal (alert text visible, no success banner) so the E2E suite does not encode the bug.

### Non-blocking notes
2. **Admin UX/i18n**: Step4Review.tsx `catch` shows `apiErr.message` (English backend text) for a 422 without `unsatisfiedRules`; ru/kk admins see English. Map `INSUFFICIENT_QUESTIONS` to an `exam.*` i18n key.
3. **Accessibility**: disabled button is not programmatically tied to the reason (no `aria-describedby` / id on the `role="status"` paragraph); disabled buttons are unfocusable so keyboard/SR users may miss the reason. Also `text-amber-600` + `text-xs` on white is ~3.2:1 (< AA 4.5:1). `role="status"` per card is fine (test uses single `getByRole('status')`; would break if two closed exams render, but it is a test-only concern).
4. **Date display**: `toLocaleString()` uses the browser locale, not the app's i18n language (kk/ru UI with en browser shows mixed formats). Prefer `toLocaleString(i18n.language)`.
5. **Clock skew**: the card decides with the client clock while the server decides on start; mismatch (client early/late) is covered by the `EXAM_OUTSIDE_WINDOW` modal message from commit 1, so acceptable. Exams already active with zero rules are only caught at start time (guard in sessions service), not retroactively; acceptable.

## Checklist summary
Backend layering, error wrapping, response envelope, UUID/UTC: pass. i18n parity: pass. Tests added for service, handler, portal service, card (closed/not-open/open) and per-code modal: pass. Tests not executed in this read-only review.
