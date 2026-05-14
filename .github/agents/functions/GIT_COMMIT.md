# Function: GIT_COMMIT

> Auto-commit changes after successful tests.

## Metadata

| Property | Value |
|----------|-------|
| **Category** | Workflow Control |
| **Used By** | Release Finalizer |
| **Depends On** | All tests must pass (Pipeline A and B) |

---

## Purpose

Commit changes to git after a successful pipeline run. This eliminates manual commit steps.

**Prerequisite**: Tests MUST have passed for Pipeline A and B. Never commit when tests are failing.

---

## Steps

```bash
# 1. Check what will be committed
git status

# 2. Stage specific files — NEVER use git add -A blindly
git add backend/internal/ backend/cmd/ backend/migrations/
git add frontend/src/
git add docs/requirements/ docs/architecture-guide.md
# Always exclude: .env, node_modules/, dist/, vendor/, *.log, docs/handoffs/

# 3. Verify staged files look correct
git diff --cached --stat

# 4. Commit
git commit -m "{type}({scope}): {subject}"
```

---

## Commit Message Format

**Convention**: Conventional Commits

| Type | When |
|------|------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `chore` | Config, deps, Docker, migrations |
| `test` | Adding or fixing tests |

**Scopes**:

| Scope | Area |
|-------|------|
| `auth` | Authentication, JWT |
| `users` | User management, RBAC |
| `departments` | Department management |
| `questions` | Question bank |
| `exams` | Exam configuration |
| `sessions` | Exam session delivery |
| `grading` | Grading engine |
| `certs` | Certificate generation |
| `reports` | Analytics, reporting |
| `audit` | Audit log |
| `tenant` | Tenant branding/config |
| `frontend` | Frontend-only cross-domain changes |
| `infra` | Docker, Nginx, migrations, env |

**Examples**:

```bash
git commit -m "feat(auth): FR-BB14 login endpoint with JWT + refresh cookie"
git commit -m "fix(sessions): ISS-007 auto-submit fires for already-submitted sessions"
git commit -m "chore(infra): add SMTP env vars to .env.example"
git commit -m "docs: add FR-BB14 requirement doc"
```

**Rules**:
- Subject max 72 characters
- Do NOT use `git add -A`
- Never commit `.env`, `node_modules/`, `dist/`, `vendor/`
- Migration files go in `backend/migrations/` — stage them explicitly
