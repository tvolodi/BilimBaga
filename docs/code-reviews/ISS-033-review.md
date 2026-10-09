# ISS-033 Code Review (FR-BB115 Account Recovery)

Verdict: PASS (no Critical/High).

Medium: found vs unknown email timing difference (purge moved to found path; residual DB-write difference remains); throttle count-then-insert race (low impact); plain fetch in recovery.ts is intentional (public endpoints).
Low: no Referrer-Policy on reset page; access JWTs valid until expiry (AC-4 revokes refresh tokens only); email lookup case-sensitive like login; password_reset_failed audit per bad attempt (bounded by AuthLimiter); SQL only covered by fakes + schemaguard until live DB run.

Security verified: 32-byte crypto/rand token, SHA-256 hash stored, atomic single-use claim in one tx, weak password does not consume token, tokens never logged/audited, session revocation, unlock RBAC and 404.
