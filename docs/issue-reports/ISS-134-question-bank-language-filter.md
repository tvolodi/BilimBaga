---
id: ISS-134
title: Question bank language filter has no effect; old versions listed as separate rows
status: resolved
severity: high
layer: frontend
module: questions
tags: [locale, locale_missing, include_versions, ListFiltered, QuestionBankPage, parent_id]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/admin/questions/QuestionBankPage.test.tsx
---

## Symptom
GitHub #134 / #7 items 2.1, 2.2: language dropdown in the question bank does not filter (it shows questions WITHOUT the language); after editing an active question, v1 (archived) and v2 (draft) both appear as separate rows ("pile"). Translations already group on one row (locale_coverage icons).

## Root Cause
1. QuestionBankPage's only language dropdown was wired to `locale_missing` (NOT EXISTS translation) and the API client had no `locale` param, although the backend `locale=` filter (EXISTS translation) was correct.
2. `ListFiltered` returned every `questions` row; editing an active question inserts a new row (parent_id -> old) and archives the old one (FR-BB23 AC-4), so both versions were listed.

## Fix Applied
- Backend: `QuestionFilter.IncludeSuperseded`; `?include_versions=true` handler param; list+count SQL add `AND ($9::bool OR NOT EXISTS (SELECT 1 FROM questions child WHERE child.parent_id = q.id))` so superseded versions are hidden by default (count is over the same predicate, so pagination stays correct). Version history slide-over unchanged.
- Frontend: language dropdown now sends `locale` ("All locales" = no param); the missing-translation filter (FR-BB27 AC-2) kept as its own dropdown; "Show previous versions" checkbox sends `include_versions=true`. i18n key `questionBank.filter.showOldVersions` in en/ru/kk.
- No translation-grouping rewrite: translations already live on one question row.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/questions/{model,handler,repository}.go | include_versions filter |
| backend/internal/questions/handler_test.go | param parsing tests |
| frontend/src/api/questions.ts, pages/admin/questions/QuestionBankPage.tsx | locale + versions wiring |
| frontend/src/locales/{en,ru,kk}.json | new key |

## Regression Test
QuestionBankPage.test.tsx (locale param, locale_missing separate, include_versions toggle); handler_test.go (TestQHandlerList_ParsesLocaleAndVersionParams, TestQHandlerList_DefaultHidesSupersededVersions). SQL not covered by real-Postgres test (needs-live-db).

## Resolution Results
- Tests: backend go test ./... pass; frontend questions tests 16 pass; check:i18n green
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
