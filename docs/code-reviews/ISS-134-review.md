# Code Review: ISS-134 (question bank language filter + hide superseded versions)

Result: PASS

## Findings
- [Medium] backend/internal/questions/repository.go ListFiltered — SQL is not covered by a real-Postgres test (acknowledged in the issue report as needs-live-db). Verified by reading instead: `$9::bool` is the 9th element of `args` (after searchParam), count and list use identical predicates, and LIMIT $10 OFFSET $11 match `append(args, PerPage, offset)`. `questions.parent_id` exists (migration 009, indexed by idx_questions_parent_id), so the NOT EXISTS subquery is efficient. Suggestion: add an integration test when a DB test harness is available.
- [Low] backend/internal/questions/handler_test.go — diff includes gofmt-only realignment of struct fields (noise, harmless).
- [Low] QuestionBankPage.tsx — the "Missing translation" select reuses the aria-label/placeholder `localeMissing` (label exists in all three locales, line 175); fine. The new `locale` select reuses the existing `questionBank.filter.locale` key.
- [Low] Semantics: a parent is hidden whenever any child row exists, regardless of child status (archived/deleted-draft). Matches the FR-BB23 AC-4 versioning model; noted only.

## Checks
- SQL parameter numbering: $1-$8 unchanged, `$9::bool` new, LIMIT $10 OFFSET $11 correct; param is bound (no concatenation). OK
- Arg order: IncludeSuperseded appended last before pagination args. OK
- Count/list consistency: identical new predicate in both queries. OK
- Other callers: QuestionFilter constructed only in handler.go; ListFiltered is called only via service.go -> repository. The exams package ListFiltered is unrelated. No other call sites need updating. OK
- Handler: `include_versions == "true"` default false (hide old versions); thin handler, no business logic. OK
- Frontend: API client via existing builder (apiFetch path unchanged); `locale` and `include_versions` params added; UI state in URL search params; `hasActiveFilters` updated. OK
- i18n: `questionBank.filter.showOldVersions` added to en, ru, kk. No hardcoded strings (KK/RU/EN option codes match existing pattern). OK
- Tests: handler tests for param parsing and default; frontend tests for locale, locale_missing separation, include_versions toggle. Adequate.

## AC Coverage
- Language filter returns questions that have the translation: covered
- Old versions hidden by default, optional toggle: covered
- Missing-translation filter (FR-BB27 AC-2) preserved: covered

Summary: Correct and consistent change with no Critical or High findings; the SQL itself is unverified against a live DB (Medium).
