# Code Review: ISS-095 / GitHub #95 nginx client_max_body_size

Result: PASS

## Scope
- deploy/nginx.conf (container; validated with `nginx -t` in docker)
- deploy/nginx/bilimbaga.conf, deploy/nginx/bilimbaga-test.conf (host vhosts; reviewed only, need ssl certs)

## Findings
- [Low] Directive placement is correct in all three files: server-level, so it applies to every location including `/api/`. Value 11m is above the app cap (`MaxCSVBytes = 10 << 20`, `ParseMultipartForm(10 << 20)`) with multipart headroom.
- [Low] Container nginx runs behind the host vhost (3101 -> container :80), so the effective limit is the minimum of the two layers. Both are now 11m, so no layer rejects at 1 MB. Consistent.
- [Low] Behaviour at 10-11 MB: nginx passes the request and the app's own validation responds, which is the intent in the comment.
- [Low, informational] `ParseMultipartForm(10<<20)` is only the in-memory threshold, not a body cap, so the nginx 11m limit is now the effective hard body cap. Acceptable; no change needed.

## Missed configs check
- deploy/Dockerfile copies deploy/nginx.conf (covered).
- docker-compose.yml mounts deploy/nginx.conf (covered).
- deploy/docker-compose.prod.yml and docker-compose.test.yml use the image built from the Dockerfile (covered).
- No nginx config in frontend/. `git ls-files` shows only the three conf files. None missed.

## Critical/High
None.
