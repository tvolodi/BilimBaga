# Code Review: ISS-241 (run swarm-241)

Requirement: FR-BB58 AC-4

Result: PASS

Findings:
- [Medium] frontend/src/api/download.ts:25 - DOWNLOAD_ERROR_KEYS is a plain object, so a server code such as "constructor" or "toString" would resolve to an inherited function (truthy) and return a non-string key. Not realistic for server codes, but a null-prototype object, Map or Object.hasOwn check would harden it.
- [Low] The AC text says "toast", the implementation uses the existing role=alert inline message in SessionHistoryTable. Pre-existing pattern, accepted by the tests.
- [Low] downloadErrorKey is shared by 8 other call sites, which now also get the specific messages for these two codes. Harmless, and the codes only occur on certificate endpoints.

Checks: no secrets or raw fetch added. Server error codes propagate through downloadError (download.ts:68-70). en/ru/kk have the same two keys. Tests cover the mapping, the unknown-code fallback and both per-code UI messages. Docs (AC-4 ticked, READMEs, ISS-241 report) are consistent.

AC Coverage:
- AC-4: covered (code-specific i18n message for SESSION_NOT_PASSED and EXAM_NOT_CERTIFIABLE)

Summary: Zero Critical and zero High findings; one Medium hardening suggestion.
