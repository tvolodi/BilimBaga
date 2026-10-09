# ISS-114 Follow-up: nginx vhost for the QA instance

Refs #114. PR #117 (compose, redeploy script, env example) was still open when this was written, so this
follow-up is a standalone file; fold it into `ISS-114-qa-instance-compose.md` if desired.

## Change
- Added `deploy/nginx/bilimbaga-qa.conf`: host nginx vhost for `bilimbaga-qa.ai-dala.com`, HTTPS only
  (HTTP->HTTPS redirect is done at the Cloudflare edge, as for the test vhost).
- Proxies to `127.0.0.1:3114` (the port published by `deploy/docker-compose.qa.yml`).
- `client_max_body_size 11m` (FR-BB64 AC-3 / #95), same proxy headers as the existing vhosts
  (Host, X-Real-IP, X-Forwarded-For, X-Forwarded-Proto, HTTP/1.1). The existing vhosts set no extra
  security headers at nginx level, so none were added.
- No existing file was edited.

## Certificate paths (assumption)
`/etc/ssl/cloudflare/ai-dala.pem` and `.key` are a shared `*.ai-dala.com` Cloudflare origin wildcard (not
host-specific), so the QA host is covered by the same files. The vhost does not depend on the test vhost
file itself. If infra issues a QA-specific cert, only these two lines change. Infra must confirm the paths
on the host.

## Validation
- `nginx -t` with the conf mounted in the locally present `nginx:alpine` image, using a throwaway
  self-signed cert generated inside the container: syntax ok, test successful.
- No image pulled, no stack started, nothing deployed.

## Install (for infra, not run here)
Copy to `/etc/nginx/sites-available/`, symlink into `sites-enabled/`, `nginx -t`, reload. Cloudflare DNS
record for `bilimbaga-qa.ai-dala.com` is also required.
