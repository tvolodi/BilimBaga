Apply the infrastructure or configuration change described in the argument. This covers env vars, Docker Compose, Nginx, migrations, CORS, and Go Config struct. Does NOT cover product features.

---

## Step 1 — Understand the Change

1. Read the affected config files (docker-compose.yml, nginx.conf, .env.example, Config struct).
2. If a migration is requested: list `backend/migrations/` to find the next sequential number.

## Step 2 — Apply Changes

### Environment Variables
1. Add the new variable to `.env.example` with a description comment.
2. Add the typed field to `backend/internal/config/config.go`.
3. Never generate real secrets — use placeholder values only.

### Docker Compose
1. Add/modify service definitions following existing patterns.
2. Ensure health checks are present for new services.
3. Verify port mappings do not conflict.

### Migrations
1. Write `backend/migrations/{NNN}_{slug}.up.sql` and `{NNN}_{slug}.down.sql`.
2. Never edit an existing migration file.
3. Apply immediately:
   - Try `make migrate` first.
   - If unavailable: `docker exec bilimbaga-db-1 psql -U bilimbaga -d bilimbaga < backend/migrations/{NNN}_{slug}.up.sql`
   - Record: `docker exec bilimbaga-db-1 psql -U bilimbaga -d bilimbaga -c "INSERT INTO schema_migrations (version, dirty) VALUES ({NNN}, false) ON CONFLICT DO NOTHING;"`
4. Verify: `docker exec bilimbaga-db-1 psql -U bilimbaga -d bilimbaga -c "\d {table_name}"`

### Nginx
1. Edit `deploy/nginx.conf`.
2. Validate: `docker exec bilimbaga-nginx nginx -t`
3. Reload: `docker exec bilimbaga-nginx nginx -s reload`

### CORS
1. Find the CORS middleware in the Go router.
2. Add/update allowed origins — never use `*` as an allowed origin.

## Step 3 — Verify

1. Confirm the API starts cleanly: `curl http://localhost:8080/api/v1/health`
2. If a migration was applied: verify the table/column exists.

## Step 4 — Security Rules

- Never commit `.env` with real values.
- Never expose database ports on the host network.
- Never disable TLS verification in production config.
- Never add `*` as an allowed CORS origin.

## Step 5 — Report

Summarize:
- Changes applied (type: env / docker / nginx / migration / cors / config, description)
- Migration files created and applied (or "none")
- Verification status
- Any env vars the user must supply a real value for
