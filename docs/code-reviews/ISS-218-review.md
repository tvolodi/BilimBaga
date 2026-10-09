# Code Review: ISS-218 (GitHub #218, FR-BB117 D-2) department scoping for every role except super_admin

Reviewer: Code Reviewer subagent. Scope: `git diff --cached` (14 files). `go test -count=1 -p 1 ./internal/deptscope/...` passes.

## Verdict: PASS (0 Critical, 0 High)

## Consumer audit (grep of `deptscope` under backend/internal)
- reports/repository.go (16 call sites) and reports/service.go `authorizeUser`: all go through `FromContext(...).Arg()/Restricted`, so they now cover custom roles, examiner, empty role. `authorizeUser` still allows self (`sc.UserID == userID`), then subtree check, else ErrNotFound (no existence leak). OK.
- ai/repository.go insight data: `Predicate("user_id","$2")` with `Arg()`; ai/service.go `GetInsights` bypasses read AND write of the exam-wide shared cache whenever `Restricted`. OK. Loyalty narrative does not use deptscope: any non-super_admin goes through `IsEmployeeInAdminDepartment` (same department, NULL department never matches), so custom/examiner/empty role cannot read org-wide. OK.
- sessions/repository.go `ListGradingQueue`: only place using `GradingPredicate`. router.go: `gradingScope` is applied to exactly two routes (`GET /admin/grading/{sessionId}`, `POST /admin/grading/{sessionId}/answers/{questionId}`); every other session/user/report/certificate route keeps `sessionScope`/plain subtree. OK.
- Audit log: no deptscope usage, router unchanged for `/audit*`. Unchanged. OK.
- exams/users packages use their own role logic, untouched (see Low-2).

## Security properties checked
- Fail-closed: `FromContext` is a deny-list of one role. Empty role / no principal => Restricted; no department => all-zero sentinel (empty set); nil store => 500 in middleware (unchanged path); `OwnerArg()` is NULL unless Restricted and ExamOwnerID != "" (examiner with empty user id gets no carve-out).
- Role-name spoofing: `roles.name` is `UNIQUE`, so a custom role cannot be named `super_admin` or `examiner`. Role comes from the signed JWT.
- Carve-out cannot widen: `ExamOwnerID` is read only by `OwnerArg()`, which is consumed only by `GradingPredicate` (queue + `GradingSessionInScope`). `Predicate`, `UserInScope`, `SessionUserInScope`, reports, analytics, AI and user records ignore it. The carve-out is bound to `e.created_by` of the session's own exam via `JOIN exams e ON e.id = es.exam_id`, so it cannot reach other exams' sessions. Custom grading roles get none.
- Out-of-scope grading id returns the handler's own 404 (existence indistinguishable), same as before.
- SQL: all values bound as parameters; the only concatenated fragments are constant identifiers. Numbering is consistent: count query `$1..$3` filters, `$4` scope, `$5` owner; rows query `$1..$3`, `$4/$5` limit/offset, `$6` scope, `$7` owner; store query `$1` id, `$2` scope, `$3` owner. `$N::uuid` casts on each use, so type inference is fine. NULL semantics: `exams.created_by` is `NOT NULL`; with a NULL owner, `created_by = NULL` is NULL, and `NULL OR <subtree predicate>` yields the predicate's value (true/false), or NULL in the CASE which falls to `ELSE 0`/WHERE false. Correct, no widening and no accidental exclusion.

## Findings
- [Medium] sessions/scope_test.go, deptscope/store tests: SQL behaviour of `GradingPredicate`/`GradingSessionInScope` (NULL OR semantics, recursive CTE, join) is asserted only via fake DB (query-text and arg checks). No live-DB test verifies that an examiner sees own-exam out-of-department sessions and not other examiners' exams. Add an integration test when a test DB is available (issue report already notes live DB not run).
- [Low-1] ai/service.go `GetInsights`: every non-super_admin caller now skips the shared insight cache, so each request is a paid Anthropic call. Correct for privacy; consider a per-scope cache key or rely on the existing rate limit. Not blocking.
- [Low-2] exams/service.go:535 treats `examiner` as org-wide for exam listing, independent of deptscope; consistent with the carve-out only for exam metadata (no per-employee data), but should be confirmed against FR-BB117 D-2 in a follow-up.
- [Low-3] Pre-existing: the department claim comes from the JWT, so a moved user keeps the old scope until token refresh. Out of scope.
- [Low-4] Open decision recorded in the issue report: custom grading roles get no carve-out; acceptable and fail-closed.

## AC coverage (from issue report / D-2)
- Only super_admin unrestricted: covered (scope.go, deptscope tests).
- Custom role with reports:read or grading:* sees only its subtree: covered (reports, sessions, ai tests).
- Empty role / no principal / no department fail closed: covered.
- Examiner grading carve-out limited to queue, detail, grade: covered (deptscope middleware + sessions tests).
- Audit log unchanged: covered.

Summary: the deny-list change is correct and complete across all consumers, the examiner carve-out is narrow and bound to the session's own exam, SQL numbering and NULL handling are correct; only a live-DB test gap (Medium) remains.

## Cycle 2 - scope-keyed AI insights cache
Reviewer: separate Code Reviewer subagent (run ISS-218-cycle2). Verdict: **PASS** (0 Critical, 0 High).
Focus: cross-scope leakage, key collision, rate limit, TTL.
- Leakage: none. super_admin (`all`) uses only the ai_insight_cache DB row; every restricted caller uses only the in-process cache keyed by `{examID, scopeKey}`; no-department = `none` (data query also empty); scope-lookup failure skips cache read and write (fails safe); refresh writes only the scoped key.
- Collisions: none (struct key, 64-hex digest vs `all`/`none`, sorted+deduped UUID ids).
- Rate limit: unchanged (`GetInsights` never had one; only `GenerateQuestions` does). Usage logged per paid call only.
- TTL/concurrency: mutex-guarded, copies on read/write, `>=` TTL, bounded eviction.
- ExamOwnerID cannot alter insight data (ai uses only `Arg()`/`Predicate`).
Findings (all accepted, none High):
- [Medium] No singleflight: concurrent cold misses for one key each pay; bounded cost, no leak.
- [Medium] Scope id-set and insight data read in separate queries; a department move in between could store newer data under an older key for up to 24 h. Staleness only, not cross-scope.
- [Low] `SubtreeSQL` uses UNION while `Predicate` uses UNION ALL (differs only on cyclic parent_id).
- [Low] Expired entries swept only when full (max 512). [Low] Per-process cache (documented).
