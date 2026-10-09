---
id: ISS-82
title: ai GetExamInsightData selects nonexistent exams.tenant_id (and other SQL/schema drift)
status: resolved
severity: high
layer: backend
module: ai
tags: [tenant_id, GetExamInsightData, GetSessionCategoryTrack, order_num, category_id, schema-guard]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-75, ISS-38]
regression_test: backend/internal/schemaguard/schema_guard_test.go, backend/internal/ai/repository_test.go
---

## Symptom
`GET /api/v1/exams/{id}/insights` (AI insights) fails with 500: `pq: column "tenant_id" does not exist`
from `backend/internal/ai/repository.go:178` (`SELECT title, passing_score_pct, tenant_id FROM exams`).
Same class as ISS-75: the repository SQL is only exercised against fake drivers, so references to
columns that were never migrated are invisible until runtime.

## Root Cause
The schema is single-tenant: only `audit_log` carries `tenant_id` (migration 007). The AI repository
was written against a hypothetical multi-tenant `exams` table. A full audit (below) found two more
columns of the same kind in the same package, all in `internal/ai/repository.go`:

1. `exams.tenant_id` (select + comparison) - `GetExamInsightData`.
2. `session_questions.order_num` - the column is `sort_order` (migration 014; 015 only dropped `id`).
   Would have failed the per-question stats query even after (1) was fixed.
3. `exams.category_id` - exams have no category; categories hang off `exam_question_rules.category_id`
   (migration 012). Broke `GetSessionCategoryTrack` (loyalty narrative, FR-BB75) with a 500.

## Fix Applied
- `GetExamInsightData`: header query is now `SELECT title, passing_score_pct FROM exams WHERE id = $1`;
  the tenant comparison and `examInsightRow.TenantID` were removed. The `tenantID` parameter is kept
  (interface stable, service/handler/mocks unchanged) but is unused (`_`) and documented as such.
  Unknown exam still returns `ErrExamNotFound`.
- Per-question stats: `MIN(sq.sort_order) AS order_num` (alias kept, so the scan struct is unchanged).
- `GetSessionCategoryTrack`: track is derived from the exam's first (by `sort_order`) question rule that
  has a category with a non-null track, via `LEFT JOIN LATERAL` on `exam_question_rules` + `categories`.
  Limitation: manual-mode rules have no category, and only the rule's own category `track` is
  considered (not its ancestors); both yield an empty track -> `ErrNotLoyaltySession`, as before
  for an exam without a category. Behaviour is unverified against live data (no DB in this run).
- New test-only package `internal/schemaguard` replays `migrations/*.up.sql` (CREATE/DROP TABLE,
  ADD/DROP/RENAME COLUMN, in order) and statically checks every SQL string literal under
  `backend/internal` (concatenations folded): `alias.col` references, INSERT column lists, UPDATE SET
  columns, and, for single-table statements, WHERE columns and plain SELECT-list columns; plus
  unknown tables. It also asserts only `audit_log` has `tenant_id`.

## Audit result
Method: schema replayed through migration 30; 245 SQL statements in 20 files statically checked
(counts below) plus manual review of dynamic SQL in `audit/repository.go`.
Not checked statically: unqualified columns inside multi-table queries and in non-trivial
expressions (SELECT lists with functions, ORDER BY / GROUP BY). Those were reviewed only through the
alias-qualified references, which is the dominant style in this code base.

### Offenders (all fixed)
| file:line (before fix) | table | column | exists? |
|---|---|---|---|
| ai/repository.go:178 (+ comparison ~188, struct field ~155) | exams | tenant_id | NO |
| ai/repository.go:213 (and `ORDER BY order_num` alias) | session_questions | order_num | NO (sort_order) |
| ai/repository.go:295 | exams | category_id | NO (exam_question_rules.category_id) |

### Explicitly verified, valid
| file:line | table | columns | exists? |
|---|---|---|---|
| sessions/repository.go:872 | audit_log | tenant_id, actor_id, action, entity_type, entity_id, ip, metadata | yes |
| sessions/repository.go:1444 | audit_log | same | yes |
| sessions/autojob.go:192 | audit_log | same | yes |
| audit/writer.go:70 | audit_log | same | yes |
| audit/repository.go:22, 65 (dynamic) | audit_log, users | tenant_id, actor_id, action, entity_type, entity_id, ip, metadata, created_at; users.full_name, users.id | yes |
| reports/repository.go (all) | exams, exam_sessions, ... | no tenant_id (fixed in #77) | yes |

### Statements checked per file (all columns/tables resolved after the fix)
ai/repository.go 11, audit/writer.go 1, auth/repository.go 10, categories/repository.go 7,
certificates/repository.go 4, departments/repository.go 7, email/repository.go 7,
exams/repository.go 29, portal/repository.go 4, questions/import_export_repository.go 14,
questions/repository.go 34, questions/translation_repository.go 10, rbac/cache.go 1,
reports/repository.go 20, sessions/autojob.go 4, sessions/grading.go 11, sessions/repository.go 53,
tags/repository.go 7, tenant/repository.go 2, users/repository.go 12.
Every `tenant_id` mention outside `audit_log` queries: none remain (`grep tenant_id` in non-test Go:
only audit writer/repository/types, sessions audit inserts, and the auth tenant middleware comment).

## Files Changed
| File | Change |
|------|--------|
| backend/internal/ai/repository.go | remove tenant_id select/compare; order_num -> sort_order; category track via exam_question_rules; document unused tenantID |
| backend/internal/ai/repository_test.go | new: not-found, tenant ignored, sort_order, track query tests |
| backend/internal/ai/fakedb_test.go | new: fake database/sql driver (copy of reports' helper) |
| backend/internal/schemaguard/schema_guard_test.go | new: repo-wide schema guard |
| docs/issue-reports/ISS-82-tenant-id-audit.md | this report |

## Regression Test
- `internal/schemaguard`: `TestRepositorySQL_MatchesMigratedSchema` (fails on the pre-fix ai code with
  the three offenders above), `TestCheckQuery_FlagsKnownBadQuery`, `TestOnlyAuditLogHasTenantID`.
- `internal/ai`: `TestGetExamInsightData_NotFound` (ErrExamNotFound, no tenant_id in query),
  `TestGetExamInsightData_IgnoresTenantAndUsesSortOrder`, `TestGetSessionCategoryTrack_*`.

## Resolution Results
- Tests: `go vet ./...` clean; `go test -p 2 ./...` all packages ok
- Migration applied: no (none needed)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
