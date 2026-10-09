# Code Review - ISS-103 (CA certificates in scratch api image)

Verdict: APPROVE (PASS)

Scope: backend/Dockerfile, docs/issue-reports/ISS-103-ca-certs-INNER-REPORT.md

Findings
- Critical/High: none. No secrets, no app code touched.
- Correct: scratch final stage now has /etc/ssl/certs/ca-certificates.crt, a default path Go's crypto/x509 searches on Linux; no SSL_CERT_FILE needed.
- Good: `test -s` in builder fails the build if the bundle is missing/empty.
- Low (optional): `apk add` placed before `go mod download` is fine for layer caching (rarely changes). Could pin nothing further; no action needed.
- Low (optional): Dockerfile comment is accurate; the report documents other Dockerfiles as out of scope (reasonable).
- Note: runtime TLS handshake not tested (acknowledged in report); the file-presence check in the built image is adequate evidence. Follow-up redeploy/UAT is correctly listed.
- Tests: none applicable (Dockerfile-only change).
