# ISS-103 - api image has no CA certificates (Refs #103)

Root cause: backend/Dockerfile final stage is `FROM scratch`; no /etc/ssl/certs/ca-certificates.crt, so Go cannot verify any TLS server certificate (Anthropic API, TLS SMTP).

Fix: builder stage runs `apk add --no-cache ca-certificates && test -s /etc/ssl/certs/ca-certificates.crt` (build-time check fails the build if the bundle is missing/empty); final stage copies the bundle to /etc/ssl/certs/ca-certificates.crt (a path Go searches by default; no SSL_CERT_FILE needed). Cost ~180 KB.

Other Dockerfiles: deploy/Dockerfile (nginx:alpine, only proxies to api over plain HTTP) and frontend/Dockerfile (alpine static server) make no outbound HTTPS; unchanged.

Verification: `docker build` of backend image succeeded; `docker create` + export shows etc/ssl/certs/ca-certificates.crt (179359 bytes in tar); `docker cp` yields 181248 bytes. Runtime TLS handshake not tested (scratch image has no shell; a Go network test would be CI-flaky). Temp container/image removed.

Follow-up (Infra): rebuild and redeploy image to bilimbaga-test; UAT re-verify loyalty narrative TLS path.
