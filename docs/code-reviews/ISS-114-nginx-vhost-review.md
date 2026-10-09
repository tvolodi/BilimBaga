# Code Review: ISS-114 nginx vhost (bilimbaga-qa)

Result: PASS

Files reviewed: deploy/nginx/bilimbaga-qa.conf, docs/issue-reports/ISS-114-nginx-vhost-followup.md
Compared against: deploy/nginx/bilimbaga-test.conf, deploy/nginx/bilimbaga.conf

## Findings
- [Critical] none
- [High] none
- [Medium] none
- [Low] deploy/nginx/bilimbaga-qa.conf:6 - comment references deploy/docker-compose.qa.yml, which does not exist on this branch (it comes with PR #117, still open). Port 3114 is therefore unverifiable here. Confirm the compose publishes 127.0.0.1:3114:80 when #117 merges.
- [Low] deploy/nginx/bilimbaga-qa.conf - the file is ASCII with LF endings. The existing vhosts are UTF-8 with CRLF. This is harmless for nginx and arguably better. A raw `diff` against the test vhost therefore shows every line as changed. A comparison that ignores line endings shows only the intended differences.
- [Low] The cert paths are the same /etc/ssl/cloudflare/ai-dala.{pem,key} as the other vhosts. The conf flags this as an assumption, and so does the doc. Infra must confirm the paths on the host before reload. This is correctly documented.

## Requirement check
- server_name bilimbaga-qa.ai-dala.com, listen 443 ssl: OK.
- proxy_pass http://127.0.0.1:3114: OK. It does not collide with 3101 (prod) or 3111 (test).
- client_max_body_size 11m: OK. It matches the other vhosts, including the FR-BB64 / #95 comment.
- Proxy settings: OK. All six directives are identical to the test and prod vhosts (HTTP/1.1, Host, X-Real-IP, X-Forwarded-For, X-Forwarded-Proto).
- No edits to existing files: OK. Both staged paths are additions only (62 insertions, 0 deletions).
- No dependence on bilimbaga-test: OK. It is mentioned only in comments. The server block is self-contained.
- No secrets in source: OK. Only cert file paths appear.
- The doc matches the conf. Its claims about the port, headers, cert assumption, validation and install steps are consistent with the file.
- nginx -t was reported as passed by the spawner (nginx:alpine, dummy cert). Not re-run here.

## Notes (not defects)
- There is no WebSocket upgrade, no proxy timeouts and no HTTP listener. The existing vhosts have the same limits, so parity is correct.
- The doc names the DNS record and the Cloudflare edge redirect as prerequisites, which is correct.

Summary: The vhost is a faithful, self-contained clone of the test vhost with only the hostname and port changed. There are no Critical or High findings.
