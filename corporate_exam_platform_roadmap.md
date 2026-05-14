# Corporate Exam Platform — Development Roadmap

> **Stack:** Go 1.22 · React 18 + TypeScript · PostgreSQL 16 · Docker Compose  
> **Tenancy:** Single-tenant, schema-per-tenant ready  
> **Document purpose:** Agent-facing task breakdown. Each phase is a self-contained Claude Code session scope.

---

## Phase 1 — Foundation

*Goal: a running, deployable skeleton with auth, branding, and user management. No exam logic yet.*

### 1.1 Project scaffold
- `docker-compose.yml` with four services: `db`, `api`, `frontend`, `nginx`
- Go module init, Chi router, sqlx, golang-migrate wired up
- Vite + React 18 + TypeScript project init, Tailwind CSS, shadcn/ui baseline
- Nginx config: static file serving for frontend, `/api/*` proxy to Go service
- `.env.example` with all required variables documented
- `Makefile` targets: `dev`, `build`, `migrate`, `test`

### 1.2 Database bootstrap
- Migration runner executes on API container startup
- `migrations/` folder, numbered SQL files (`001_init.sql`, etc.)
- Initial tables: `schema_migrations` (managed by golang-migrate)
- PostgreSQL connection pool configured via env (max conns, idle timeout)

### 1.3 Tenant configuration
- Table: `tenant_config (key TEXT PK, value JSONB, updated_at)`
- Keys: `app_name`, `logo` (base64 blob), `primary_color`, `accent_color`, `default_locale`, `available_locales`
- Go service loads config at startup, caches in memory, invalidates on update
- `GET /api/v1/tenant/config` — returns public config (name, colors, locales); no auth required (used by frontend before login)
- `PUT /api/v1/tenant/config` — updates config; super admin only
- `GET /api/v1/tenant/logo` — serves logo bytes with correct Content-Type; no auth required

### 1.4 Authentication
- Table: `users (id UUID PK, email, password_hash, full_name, department_id, role_id, status, force_password_change, created_at, updated_at)`
- `POST /api/v1/auth/login` — email + password → JWT access token (15 min) + httpOnly refresh cookie (7 days)
- `POST /api/v1/auth/refresh` — validates refresh cookie → new access token + rotated refresh cookie
- `POST /api/v1/auth/logout` — invalidates refresh token, clears cookie
- `POST /api/v1/auth/change-password` — requires current password; clears `force_password_change` flag
- Passwords: bcrypt cost 12
- Account lockout: `failed_attempts` counter; lock after 5 failures; unlock via admin or email link
- All auth events written to `audit_log`

### 1.5 JWT middleware
- Middleware validates Bearer token on all protected routes
- Injects `user_id`, `role`, `department_id` into request context
- Returns 401 on missing/expired token, 403 on insufficient role
- Tenant context injected at same middleware level (phase 1: always `public` schema)

### 1.6 RBAC
- Table: `roles (id, name)` — seeded: `super_admin`, `department_admin`, `examiner`, `employee`
- Table: `permissions (id, resource, action)` — e.g. `questions:write`, `users:manage`
- Table: `role_permissions (role_id, permission_id)`
- Middleware helper `RequirePermission(resource, action)` used as route-level guard
- Seed data: full permission matrix for all four roles

### 1.7 Department management
- Table: `departments (id UUID PK, name, parent_id UUID nullable, created_at)`
- `GET /api/v1/departments` — tree structure; admin only
- `POST /api/v1/departments` — create; super admin only
- `PUT /api/v1/departments/:id` — rename; super admin only
- `DELETE /api/v1/departments/:id` — only if no users assigned; super admin only

### 1.8 User management API
- `GET /api/v1/users` — paginated list; filterable by department, role, status; admin only
- `POST /api/v1/users` — create user; sets `force_password_change = true`; admin only
- `GET /api/v1/users/:id` — get user detail; admin or self
- `PUT /api/v1/users/:id` — update name, department, role; admin only
- `POST /api/v1/users/:id/deactivate` — sets status inactive; preserves all records; admin only
- `POST /api/v1/users/:id/reset-password` — generates temp password, sets `force_password_change`; admin only
- `POST /api/v1/users/import` — CSV bulk import; returns preview with validation errors before commit
- `GET /api/v1/users/me` — current user's own profile

### 1.9 Audit log
- Table: `audit_log (id UUID PK, actor_id UUID, action TEXT, entity_type TEXT, entity_id UUID, ip TEXT, metadata JSONB, created_at)`
- Append-only; no UPDATE or DELETE ever issued against this table
- Go helper `audit.Write(ctx, action, entityType, entityID, metadata)` used in all handlers
- `GET /api/v1/audit` — paginated; filterable by date range, actor, action; super admin only
- `GET /api/v1/audit/export` — CSV download; super admin only

### 1.10 Frontend: auth screens
- Login page: company logo (from tenant config), email + password fields, language selector
- Forced password change screen (shown when `force_password_change = true`)
- Error states: invalid credentials, account locked, network error
- Tenant config loaded before render to apply brand colors and logo

### 1.11 Frontend: admin shell
- Sidebar navigation (collapsible): Dashboard, Users, Departments, Questions, Exams, Reports, Settings
- Top bar: current user name, role badge, logout
- Breadcrumb component
- Route guards: redirect to login if unauthenticated; redirect to employee portal if role = employee
- User list page with filters, status badges, bulk import button
- User create/edit drawer

### 1.12 Frontend: branding settings page
- Form: application name, logo upload (drag-and-drop + file picker), primary color picker, accent color picker
- Live preview panel showing logo and color applied to a sample header
- WCAG AA contrast warning if selected colors fail
- Available locales multi-select; default locale selector

---

## Phase 2 — Content Management

*Goal: full question bank with multilanguage support. No exam delivery yet.*

### 2.1 Categories and tags
- Table: `categories (id UUID PK, name, parent_id nullable, track TEXT, sort_order)`
- Table: `tags (id UUID PK, name UNIQUE)`
- Seeded categories for three built-in tracks: security, safety, loyalty/values
- `GET /api/v1/categories` — full tree; authenticated
- `POST /api/v1/categories`, `PUT`, `DELETE` — examiner and above
- `GET /api/v1/tags`, `POST /api/v1/tags` — examiner and above

### 2.2 Question model
- Table: `questions (id UUID PK, category_id, difficulty ENUM('easy','medium','hard'), type ENUM('single','multiple','truefalse','likert','shorttext'), default_locale, status ENUM('draft','review','active','archived'), created_by, version INT, parent_id UUID nullable, created_at, updated_at)`
- Table: `question_translations (question_id, locale, stem TEXT, explanation TEXT, updated_at)` — one row per locale
- Table: `answer_options (id UUID PK, question_id, sort_order INT, is_correct BOOL, likert_weight DECIMAL nullable, likert_polarity ENUM('positive','negative') nullable)`
- Table: `answer_translations (option_id, locale, text TEXT)`
- `parent_id` links a new version to its predecessor; only the latest version is `active`

### 2.3 Question CRUD API
- `GET /api/v1/questions` — paginated; filterable by category, tag, difficulty, type, status, locale coverage; examiner+
- `POST /api/v1/questions` — create in draft; requires default locale stem + at least 2 answer options for choice types
- `GET /api/v1/questions/:id` — full question with all translations and options
- `PUT /api/v1/questions/:id` — edit; creates new version if status was active; old version archived
- `POST /api/v1/questions/:id/status` — transition status (draft→review→active, active→archived)
- `DELETE /api/v1/questions/:id` — only allowed on draft; hard delete
- `GET /api/v1/questions/:id/versions` — version history list
- Tag association: `POST /api/v1/questions/:id/tags`, `DELETE /api/v1/questions/:id/tags/:tagId`

### 2.4 Translation API
- `GET /api/v1/questions/:id/translations` — all locale variants
- `PUT /api/v1/questions/:id/translations/:locale` — upsert stem, explanation, and all answer option texts for a locale
- `DELETE /api/v1/questions/:id/translations/:locale` — remove a non-default locale translation
- Locale coverage indicator: computed field on question list showing which locales have translations vs. missing

### 2.5 Bulk import / export
- `POST /api/v1/questions/import` — accepts CSV or JSON; dry-run mode returns validation errors; commit mode inserts
- CSV columns: `type, difficulty, category_path, default_locale, stem, explanation, option_1 … option_N, correct (comma-separated indices), tags`
- `GET /api/v1/questions/export` — CSV or JSON; applies same filters as list endpoint
- Duplicate detection: warn (not block) if trigram similarity > 0.8 against existing active questions

### 2.6 Frontend: question editor
- Two-panel layout: left = form, right = live preview rendering the question exactly as an employee would see it
- Language tabs across the top: default locale always first, other configured locales as tabs with ✓/✗ coverage indicator
- Question type selector: changes the answer section dynamically
- For choice types: drag-to-reorder answer options; mark correct toggle per option
- For Likert: weight field and polarity toggle per option
- Difficulty selector, category tree picker, tag input (autocomplete)
- Status badge with transition buttons (Submit for review / Approve / Archive)
- Explanation field (collapsible, per-locale)
- Save as draft auto-saves every 30s

### 2.7 Frontend: question bank list
- Filterable table: category tree filter, tag filter, difficulty chips, type filter, status filter, locale coverage filter
- Quick-edit inline for status transitions
- Bulk actions: export selected, archive selected
- Import button: opens CSV/JSON upload with preview table of parsed rows and validation errors

---

## Phase 3 — Exam Engine

*Goal: full exam lifecycle from configuration through session delivery and auto-grading of objective questions.*

### 3.1 Exam configuration model
- Table: `exams (id UUID PK, title, description, status ENUM('draft','active','archived'), time_limit_minutes INT, passing_score_pct DECIMAL, max_attempts INT, available_from TIMESTAMPTZ nullable, available_until TIMESTAMPTZ nullable, shuffle_questions BOOL, shuffle_options BOOL, show_answers ENUM('never','after_completion','after_all_attempts'), on_tab_switch ENUM('log','warn','submit'), certificate_enabled BOOL, created_by, created_at, updated_at)`
- Table: `exam_sections (id UUID PK, exam_id, title nullable, sort_order)` — optional grouping within an exam
- Table: `exam_question_rules (id UUID PK, exam_id, section_id nullable, mode ENUM('manual','random'), category_id nullable, tag_ids JSONB, difficulty TEXT nullable, count INT, sort_order)` — drives question selection
- Table: `exam_manual_questions (rule_id, question_id, sort_order)` — used when mode = manual

### 3.2 Exam configuration API
- `GET /api/v1/exams` — list; filterable by status; examiner+
- `POST /api/v1/exams` — create in draft
- `GET /api/v1/exams/:id` — full config including question rules
- `PUT /api/v1/exams/:id` — update config; only allowed in draft
- `POST /api/v1/exams/:id/publish` — transitions to active; validates that question rules can be satisfied (enough active questions exist)
- `POST /api/v1/exams/:id/archive` — transitions to archived; no new sessions can start
- `DELETE /api/v1/exams/:id` — draft only; hard delete

### 3.3 Exam assignment
- Table: `exam_assignments (id UUID PK, exam_id, assignee_type ENUM('user','department','all'), assignee_id UUID nullable, deadline TIMESTAMPTZ nullable, assigned_by, assigned_at)`
- `POST /api/v1/exams/:id/assign` — body: `{ assignee_type, assignee_id, deadline }`; super admin and department admin
- `DELETE /api/v1/exams/:id/assign/:assignmentId` — remove assignment
- `GET /api/v1/exams/:id/assignments` — list assignments with completion stats per assignee

### 3.4 Employee exam portal API
- `GET /api/v1/portal/exams` — exams assigned to current user; includes status (not_started, in_progress, passed, failed, expired), attempts used, deadline
- `GET /api/v1/portal/exams/:id` — exam detail: title, description, time limit, passing score, attempt info

### 3.5 Session creation
- `POST /api/v1/portal/exams/:id/sessions` — starts a new session
  - Checks: exam active, within availability window, attempts not exhausted, no currently open session for this exam
  - Resolves questions from rules: random selection runs at session creation time using a seeded PRNG (seed stored in session for reproducibility)
  - Applies shuffle if configured
  - Creates `exam_sessions` record with `started_at`, `expires_at` (= started_at + time_limit), status = `in_progress`
  - Creates `session_questions (session_id, question_id, question_version_id, sort_order)` — snapshot of which questions/versions were assigned
  - Returns session ID, question list (stems + options, no correct flags), remaining seconds

### 3.6 Session tables
- Table: `exam_sessions (id UUID PK, exam_id, user_id, status ENUM('in_progress','submitted','auto_submitted','grading_pending'), started_at, expires_at, submitted_at, score_pct DECIMAL nullable, passed BOOL nullable, seed BIGINT)`
- Table: `session_questions (session_id, question_id, question_version_id, sort_order)`
- Table: `session_answers (id UUID PK, session_id, question_id, selected_option_ids JSONB, text_answer TEXT nullable, saved_at, time_spent_seconds INT)`
- Table: `tab_switch_events (session_id, occurred_at, action_taken TEXT)`

### 3.7 Answer saving
- `PUT /api/v1/portal/sessions/:id/answers/:questionId` — upsert answer for one question
  - Validates: session in_progress, not expired, question belongs to session, option IDs belong to question
  - Server returns `{ remaining_seconds }` on every save so client clock stays server-authoritative
- `GET /api/v1/portal/sessions/:id` — resume: returns session state, all saved answers, remaining seconds; used on page reload

### 3.8 Tab-switch and focus events
- `POST /api/v1/portal/sessions/:id/events` — body: `{ type: 'tab_switch' | 'blur' | 'fullscreen_exit' }`
  - Writes to `tab_switch_events`
  - If exam configured `on_tab_switch = 'submit'`: triggers auto-submit flow
  - If `on_tab_switch = 'warn'`: returns `{ warn: true }` in response; client shows modal

### 3.9 Session submission
- `POST /api/v1/portal/sessions/:id/submit` — employee manually submits
  - Marks session `submitted`; triggers grading (see 3.10)
  - If any short-text questions present: status = `grading_pending` instead; triggers manual grading queue notification

### 3.10 Auto-submit background job
- Go `time.Ticker` goroutine runs every 60 seconds
- Finds all sessions where `status = 'in_progress'` AND `expires_at < NOW()`
- Marks each `auto_submitted` and triggers grading
- Writes audit log entry per auto-submitted session

### 3.11 Grading engine
- Single / true-false: 1 point for correct option, 0 otherwise
- Multiple choice: full points if selected options exactly match correct set; partial credit if enabled: `max(0, correct_selected - incorrect_selected) / total_correct`
- Likert: `sum(option.likert_weight * (polarity == positive ? value : 6 - value))` normalized to 0–100 across all Likert questions in the session
- Short text (phase 3): marks question as `pending_manual_grade`; contributes 0 to score until graded
- Total score: `sum(question_scores) / sum(max_possible_scores) * 100`
- Writes score to `exam_sessions.score_pct`; sets `passed` based on exam passing threshold

### 3.12 Frontend: exam configuration UI
- Multi-step form wizard: (1) basic settings, (2) question rules, (3) assignment, (4) review & publish
- Step 2: rule builder — add rule rows; each row: mode toggle (manual/random), category picker, difficulty filter, count, drag-to-reorder
- Manual mode row: opens question picker modal (filtered question bank, multi-select)
- Live validation: "not enough active questions match your rules" shown before publish
- Publish button with confirmation modal showing question count, time limit, passing score summary

### 3.13 Frontend: employee portal
- Exam card grid: status pill (not started / in progress / passed / failed), deadline countdown, time limit, attempt counter
- "Start exam" / "Continue" / "View result" CTA per card
- Confirmation modal before starting: time limit, rules, cannot-leave warning

### 3.14 Frontend: exam taking screen
- Full-focus layout: no sidebar, no navigation
- Top bar: exam title, timer (warning color at 20% remaining), progress `X / Y answered`
- Question navigator panel (collapsible on mobile): grid of question numbers, colored by answered/flagged/unanswered
- Question display: stem, options (radio / checkbox / Likert scale / text area)
- Flag for review toggle per question
- Auto-save indicator (saving… / saved / connection lost)
- "Finish exam" button → review screen listing unanswered and flagged questions → confirm submit

---

## Phase 4 — Results & Certificates

*Goal: complete result flow, certificate generation, and manual grading queue.*

### 4.1 Result retrieval API
- `GET /api/v1/portal/sessions/:id/result` — own result; available once session not in_progress
  - Returns: score_pct, passed, time_taken, per-question breakdown (if show_answers enabled), per-section scores
- `GET /api/v1/admin/sessions/:id/result` — any session result; examiner+
- `GET /api/v1/portal/exams/:id/history` — all own sessions for this exam; chronological

### 4.2 Manual grading queue
- `GET /api/v1/admin/grading` — sessions with status `grading_pending`; filterable by exam, date; examiner+
- `GET /api/v1/admin/grading/:sessionId` — session detail with all short-text answers
- `POST /api/v1/admin/grading/:sessionId/answers/:questionId` — submit grade (score 0–1) and optional feedback text
- Once all pending questions in a session are graded → triggers final score recalculation → session status = `submitted`
- Audit log entry per grade submitted

### 4.3 Certificate generation
- Table: `certificates (id UUID PK, session_id UNIQUE, verification_code UUID UNIQUE, issued_at, employee_name, exam_title, score_pct, template_snapshot JSONB)`
- Certificate created on first request (lazy generation), not pre-generated
- `GET /api/v1/portal/sessions/:id/certificate` — returns PDF; creates certificate record if not exists; own sessions only
- `GET /api/v1/admin/sessions/:id/certificate` — same for admins; any session
- `GET /verify/:code` — public endpoint; returns JSON `{ valid: true, employee_name, exam_title, issued_at }` or 404
- PDF template: company logo, app name, employee full name, exam title, score, date, signatory name + title, verification URL with QR code, certificate ID

### 4.4 PDF generation (Go)
- Use `unipdf` library (or `wkhtmltopdf` if preferred) for server-side PDF
- Template defined as Go struct with field substitution; no external template files required
- Logo fetched from tenant config cache
- QR code generated inline pointing to verification URL

### 4.5 Frontend: result screen
- Score dial / percentage display, pass/fail banner in brand colors
- Time taken, attempt number
- Per-section score breakdown (if exam has sections)
- Per-question review table (shown if exam allows): question stem, employee's answer, correct answer, points earned; explanation text shown per question if set
- Certificate download button (shown if passed and certificate enabled)
- "Retake exam" button (shown if attempts remaining)

### 4.6 Frontend: employee history
- Tab on employee portal: "My results"
- Table: exam name, date taken, score, pass/fail, certificate link
- Sortable by date and score

### 4.7 Frontend: manual grading UI
- Queue table: session ID, employee name, exam name, submission date, questions pending
- Grading detail page: one question at a time; employee's text answer; score slider 0–100 + feedback field; next/previous navigation; submit all grades button

---

## Phase 5 — Analytics & Reporting

*Goal: admin-facing analytics, export, and audit log viewer.*

### 5.1 Dashboard metrics API
- `GET /api/v1/admin/dashboard` — returns:
  - `completion_rate_by_exam[]`: exam_id, title, assigned_count, completed_count, passed_count
  - `overdue_employees[]`: user_id, name, exam_title, deadline (limit 20)
  - `recent_activity[]`: last 20 completed sessions with employee name, exam, score, passed, timestamp
  - `avg_score_by_track{}`: security, safety, loyalty averages across all sessions last 90 days

### 5.2 Per-exam analytics API
- `GET /api/v1/admin/exams/:id/analytics` — returns:
  - Score distribution: histogram buckets (0–10%, 10–20% … 90–100%) with counts
  - Pass rate, average score, median score, total attempts, unique participants
  - Per-question stats: `{ question_id, stem_preview, correct_rate, avg_time_seconds, answer_distribution[] }`
  - `answer_distribution`: for each option, count of employees who selected it (anonymized aggregate)

### 5.3 Per-employee record API
- `GET /api/v1/admin/users/:id/record` — all sessions with exam name, date, score, passed, certificate_id, time_taken; paginated
- `GET /api/v1/admin/users/:id/progress` — per track: questions answered, last activity, pass status of required exams

### 5.4 Export API
- `GET /api/v1/admin/exams/:id/results/export` — CSV of all sessions for this exam: employee name, department, date, score, passed, time taken, per-question scores (one column per question)
- `GET /api/v1/admin/users/:id/record/export` — CSV of all sessions for this user
- `GET /api/v1/admin/dashboard/export` — PDF summary report: company logo, date range, completion rates table, pass rates table, top/bottom questions

### 5.5 Audit log viewer
- `GET /api/v1/audit` — already built in phase 1; this phase adds frontend
- Filterable table: date range picker, actor name search, action type multi-select, entity type filter
- Row detail expandable: shows full metadata JSON
- Export to CSV button

### 5.6 Frontend: admin dashboard
- KPI cards: total employees, exams active, completion rate (last 30 days), pass rate (last 30 days)
- Completion rate bar chart by exam (recharts)
- Overdue employees table with quick-assign-reminder action
- Recent activity feed

### 5.7 Frontend: per-exam analytics page
- Score distribution histogram
- Pass rate gauge
- Per-question difficulty table: sortable by correct rate; highlights questions with correct rate < 40% (consider revising) and > 95% (consider removing)
- Export button

### 5.8 Frontend: employee record page
- Accessible from Users list → employee row → "View record"
- Session history table with certificate download links
- Track progress summary: security / safety / loyalty completion status

---

## Phase 6 — Polish & Hardening

*Goal: production-ready quality: full i18n, accessibility, security hardening, performance.*

### 6.1 Email notifications
- Go email service using `net/smtp`; SMTP config via env (host, port, user, password, TLS)
- Templates (plain text + HTML multipart): new exam assigned, exam completed (pass — with certificate link), exam completed (fail — with retake info), deadline reminder (48h), password reset link
- All user-facing strings in templates loaded from locale files
- Emails sent to recipient's locale
- Background goroutine for deadline reminders: scans assignments daily at 08:00 tenant timezone
- `POST /api/v1/admin/notifications/test` — sends test email to current admin; super admin only

### 6.2 Full i18n coverage
- All UI strings externalized to `src/locales/{locale}.json`; zero hardcoded user-visible strings in components
- RTL layout: CSS logical properties throughout; `dir="rtl"` on `<html>` when active locale is RTL; tested with Arabic
- Number and date formatting via `Intl` API respecting active locale
- Locale switcher available on login screen and in user profile settings
- Content language fallback: if question has no translation for user's locale, renders default locale with a "(original language)" badge

### 6.3 Accessibility
- All interactive elements keyboard-navigable; focus ring visible
- ARIA labels on icon-only buttons, status badges, progress indicators
- Exam timer announces remaining time via `aria-live` region at 5-minute intervals and at 1 minute
- Color contrast meets WCAG AA across all brand color combinations
- Screen reader tested: VoiceOver (macOS) and NVDA (Windows) for exam-taking flow

### 6.4 Security hardening
- Rate limiting: auth endpoints 10 req/min/IP; answer save 60 req/min/session (Chi middleware)
- CSP header configured in Nginx: `default-src 'self'`; no inline scripts
- `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`
- All file uploads re-validated server-side regardless of Content-Type header
- Exam answer validation: option IDs must belong to session's questions; unknown IDs rejected 400
- Dependency audit: `go mod verify`, `npm audit` both pass with no high/critical findings

### 6.5 Performance
- PostgreSQL: indexes on `session_answers(session_id)`, `audit_log(created_at, actor_id)`, `exam_sessions(user_id, exam_id)`, `question_translations(question_id, locale)`
- API response times: p95 < 200ms for all endpoints under 100 concurrent users (k6 load test)
- Frontend: Lighthouse score ≥ 90 performance on employee portal and exam-taking pages
- Tenant config and logo cached in Go process memory; invalidated via cache-bust on update
- React Query used for all API state; stale-while-revalidate for non-critical data

### 6.6 Observability
- Structured JSON logging (zerolog): all requests logged with method, path, status, latency, user_id
- Health endpoint: `GET /api/v1/health` returns `{ status, db_ok, version }`
- Docker Compose includes log volume mounts; logs rotated by Docker log driver
- Go panic recovery middleware: logs full stack trace, returns 500 without leaking internals

---

## Phase 7 — AI Layer

*Goal: AI-assisted content authoring and intelligent result insights. Requires phase 1–5 data to be meaningful.*

### 7.1 Question generation assist
- `POST /api/v1/admin/ai/generate-questions` — body: `{ category_id, difficulty, count, context_text }`
  - Calls Anthropic API with structured prompt; returns N draft questions in question import JSON format
  - Questions inserted as drafts, not active; examiner reviews before publishing
  - Usage logged to `ai_usage_log (id, user_id, feature, tokens_used, created_at)`

### 7.2 Adaptive difficulty (exam engine extension)
- Exam config gains optional `adaptive: true` flag
- Session question selection algorithm: after each answer, next question difficulty adjusts based on running correct rate (3PL IRT approximation)
- Requires minimum question bank size per difficulty level; validation at publish time

### 7.3 Short-text auto-grading
- When a short-text question has `auto_grade: true` and a model answer set, grading engine calls Anthropic API to score the employee's answer 0–100 against the model answer
- Result marked `ai_graded`; examiner can override
- Fallback to manual queue if API unavailable

### 7.4 Performance insight summaries
- `GET /api/v1/admin/ai/insights/:examId` — calls Anthropic API with anonymized aggregate stats; returns 3–5 bullet natural-language observations (e.g. "Question 7 has a 23% correct rate suggesting the stem may be ambiguous")
- Results cached for 24h; regeneration button for admins

### 7.5 Loyalty profile narrative
- For loyalty/values assessment exams: `GET /api/v1/admin/ai/loyalty-summary/:sessionId`
  - Generates a paragraph-length narrative of the employee's values profile based on Likert responses
  - Visible to department admin and above only; same visibility rules as raw loyalty scores
  - Clearly labeled "AI-generated summary" in UI

---

## Cross-cutting conventions (all phases)

- All API responses: `{ data: ..., error: null }` on success; `{ data: null, error: { code, message } }` on failure
- All list endpoints: `{ data: [], meta: { page, per_page, total } }`
- All timestamps: UTC ISO 8601 (`2026-05-14T10:30:00Z`)
- All IDs: UUID v4
- Go: one package per domain (`auth`, `users`, `questions`, `exams`, `sessions`, `certificates`, `reports`, `audit`); no circular imports
- Migrations: never edit existing migration files; always add new numbered files
- Secrets: never in source code or committed `.env` files; `.env.example` documents all required vars with descriptions
- Tests: each Go handler package has a `_test.go` file; minimum coverage for auth, grading engine, and session lifecycle
