# Roadmap gap audit, 2026-10-11

BA, from `origin/main` (c1d5e06 and later), `corporate_exam_platform_roadmap.md`, `docs/requirements/` and the code. Docs only; no host contacted. Scope: what changed since the last audit (FR-BB319): dark mode FR-BB321, design system FR-BB320, AI provider and background grading (#232, #528), security fixes.

## Result

No Validated requirement lacks code. The new gaps sit around AI grading, which changed from a synchronous call to a background job without its requirement following. Two requirement issues are filed (the maximum); the rest are listed for the Supervisor to route.

## Filed

| ID | Gap | Issue | Prio |
|----|-----|-------|------|
| G1 | No policy for how far an AI score may be trusted. Since PR 529 the AI score finalises the session (pass/fail, email, certificate) with no person involved; the answer text is part of the grader prompt and the architect named the injection risk. FR-BB73 still describes the old synchronous flow. BA recommendation: per-exam `ai_grading_mode` off / suggest / auto, `suggest` by default where a certificate is issued | #534 | p1 |
| G2 | Employee side of the wait is not specified or built: result page never refetches, copy says "manual grading" although AI grades in minutes, no portal status for grading pending, no "submitted" mail, certificate link in the result mail is an authenticated API path, no per-question pending marker | #535 | p2 |

## Not filed (route as needed)

| ID | Gap | Evidence | Suggested route |
|----|-----|----------|-----------------|
| G3 | AI provider configuration (anthropic, openai_compatible, x.ai, `AI_BASE_URL`, `AI_MODEL`, `AI_GRADING_DAILY_LIMIT`, `AI_INSIGHTS_DAILY_LIMIT`, key rules) has no requirement; the roadmap Phase 7 says Anthropic only | `backend/internal/ai/provider.go`, `backend/internal/config/ai.go`; no FR mentions them | BA, fold into the G1 amendment or a small FR-BB71 note |
| G4 | README says Implemented, the docs say Draft: FR-BB19 (Audit log) and FR-BB114 (Audit log viewer) | `docs/requirements/README.md` rows 21 and 26 against the Status field of each doc | BA, editorial PR |
| G5 | Other status mismatches: FR-BB316 `draft` in the doc; FR-BB110, FR-BB14, FR-BB16, FR-BB17 use "Ready", not in the README vocabulary; four `*-process.md` docs still `status: draft` | `docs/requirements/` | BA, same editorial PR as G4 |
| G6 | Certificates are issued lazily on first request and only when `passed = true`; nothing says what happens to an issued certificate when an examiner later changes a graded answer | `backend/internal/certificates/service.go` lines 58 to 65 | Part of G1 (examiner override after finalisation) |
| G7 | Anti-cheat events (`tab_switch`, `blur`, `fullscreen_exit`) are written and counted but no admin view of them was found; FR-BB319 covers capture and hygiene only | `backend/internal/sessions/repository.go` around line 705; nothing in `reports/` or `frontend/src/pages/admin` | BA candidate for the next audit; owner decides whether an examiner needs it |
| G8 | Phase 7 FRs FR-BB71 to FR-BB75 are marked Implemented with no UAT evidence found for the features beyond the insights and grading checks done on #232 and #528 | `docs/uat-reports/` | UAT sweep with a key on the local stack (owner's go for paid calls) |
| G9 | The demo cannot show any of the AI features until it has a key and a full redeploy; per the owner brief the key decision is the owner's | `docs/handoffs/demo-full-redeploy-20261011/owner-brief.md` | Owner |

## Checked and fine

- FR-BB320 and FR-BB321: code present (`frontend/src/components/ThemeProvider.tsx`, `ThemeToggle.tsx`, design gallery at `/__design`), accessibility and theme specs passed on main per #435.
- FR-BB319: capture and hygiene in `sessions/service.go` and `TabSwitchWarningModal.tsx`.
- FR-BB510: reminders in `email/service.go`, migration 036.
- Security follow-ups (#415, #439, #455, #457, #466, #475, #478, #481): closed with UAT evidence.

## Note on method

A first scan reported FR-BB28 and FR-BB29 as missing from the README; both rows exist and the claim was dropped. Claims G6 to G8 are from code reading and were not run.
