# FR-BB65 — Performance

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB65 |
| Phase | 6 — Polish & Hardening |
| Priority | 2 |
| Status | Validated |
| Depends On | FR-BB12, FR-BB36, FR-BB37, FR-BB13, FR-BB43, FR-BB62 |

## Description
Establishes performance benchmarks and implements the optimizations required to meet them. PostgreSQL is given targeted indexes on all high-traffic query patterns. The Go API caches tenant configuration and logos in memory. The React frontend uses lazy-loaded route chunks and tuned React Query stale times to minimize redundant fetches. A k6 load test script serves as the acceptance artifact for the p95 latency target.

**Revision 2 (2026-10-09)**: most items are shipped; remaining scope is measured evidence (EXPLAIN, k6, Lighthouse) plus small tooling/Nginx deltas - see Delta Status.

## Delta Status (revised 2026-10-09)
The implementation is largely shipped. This revision turns the requirement into a **delta + evidence plan**: shipped ACs are cited with file evidence; remaining work is (1) measured evidence (EXPLAIN, k6, Lighthouse) and (2) small tooling/config deltas that the evidence is likely to demand.

| AC | State | Evidence / gap |
|----|-------|----------------|
| AC-1 | Code met; evidence missing | `backend/migrations/024_performance_indexes.up.sql` (+ `.down.sql`) creates all required indexes (`session_answers(session_id)`, `audit_log(created_at DESC, actor_id)`, `exam_sessions(user_id, exam_id, status)`, `question_translations(question_id, locale)`, `exam_assignments(exam_id)`, `certificates(session_id)`, `certificates(verification_code)`). No committed `EXPLAIN ANALYZE` output. |
| AC-2 | Script exists; never run | `docs/test-reports/k6-load-test.js` hits only `GET /api/v1/exams`; the AC says "all endpoints". No summary committed. Per-IP rate limiters (`backend/internal/ratelimit/middleware.go`) would return 429 for 100 VUs from one host unless `DISABLE_RATE_LIMIT=true` on the test stack. |
| AC-3 | Not measured | No Lighthouse tooling or report in repo. `deploy/nginx.conf` and `deploy/nginx/*` contain no gzip/brotli and no long-cache headers for hashed assets, which will likely cost Lighthouse points. `EmployeePortal` and `ExamTakingPage` are eagerly imported in `frontend/src/App.tsx`. |
| AC-4 | Met | `backend/internal/tenant/service.go` keeps an in-memory cache loaded once; `InvalidateAndRefresh` runs on update; tests in `backend/internal/tenant/cache_test.go` (incl. concurrent readers/writer). The cache lives in `Service` rather than the sketched `TenantCache` type; behavior is equivalent. |
| AC-5 | Met | `Service.GetLogoData` reads from the same cache; `Handler.GetLogo` (`backend/internal/tenant/handler.go`) sets `Cache-Control: public, max-age=3600`. Returns 204 (not 404) when no logo is set - accepted deviation. |
| AC-6 | Met | `frontend/src/api/useTenantConfig.ts` (5 min), `categories.ts` (300_000), `exams.ts` (5 min list, 30_000 session state), `grading.ts` (0). |
| AC-7 | Met | ~20 admin pages use `lazy()` in `frontend/src/App.tsx`. `AdminLayout` is eager (small shell, accepted). Evidence upgrade via D3 bundle check. |
| AC-8 | Met | `backend/internal/config/config.go` (`DB_MAX_OPEN_CONNS` 25, idle 10, lifetime 5m), wired in `backend/cmd/api/main.go`, documented in `backend/.env.example`. |

## Acceptance Criteria
- [x] AC-1: A migration adds indexes for all six specified query patterns (`session_answers`, `audit_log`, `exam_sessions`, `question_translations`, `exam_assignments`, `certificates`). **Shipped as migration 024.** Remaining evidence: `EXPLAIN (ANALYZE, BUFFERS)` output for one representative query per index, run on a seeded DB (>= 10k `session_answers`, >= 5k `audit_log` rows), is committed at `docs/test-reports/explain-analyze-FR-BB65.md`; each plan shows an Index or Bitmap Index Scan on the named index (if the planner picks a Seq Scan on a tiny table, the report re-runs with `SET enable_seqscan=off` and annotates it).
- [ ] AC-2: With 100 concurrent virtual users for 2 minutes against the production-like stack (`DISABLE_RATE_LIMIT=true` on the API for the run only), p95 `http_req_duration` is < 200 ms **per tagged endpoint group** for `GET /api/v1/exams`, `GET /api/v1/users/me`, `GET /api/v1/tenant/config`, `GET /api/v1/categories`, `GET /api/v1/portal/exams` (employee token), and `GET /api/v1/admin/dashboard` (admin token), and `http_req_failed` < 1%. The extended k6 script and the k6 summary (`--summary-export` JSON plus console text) are committed under `docs/test-reports/`.
- [ ] AC-3: Lighthouse (pinned v11) Performance score is >= 90 on the employee portal page (`/portal`) and the exam-taking page (an in-progress session), measured against the production build served by Nginx via Docker Compose (not the Vite dev server), median of 3 runs on the mobile preset (desktop preset also recorded). JSON and HTML reports are committed under `docs/test-reports/lighthouse/`. If < 90, the failing audits are recorded and D2 is applied before re-measuring.
- [x] AC-4: Tenant configuration is cached in Go process memory after first load; subsequent requests do not hit the database; the cache is refreshed atomically when `PUT /api/v1/admin/tenant/config` succeeds. (Shipped.)
- [x] AC-5: The tenant logo is served from the same in-memory cache with `Cache-Control: public, max-age=3600`, refreshed on the same update path. (Shipped.)
- [x] AC-6: React Query stale times: 5 min for tenant config, categories and exam list; 30 s for session state; 0 for the grading queue. (Shipped.)
- [x] AC-7: Admin-only routes are lazy-loaded (`React.lazy` + dynamic `import()`). Evidence upgrade: D3 bundle check shows admin pages in separate chunks and absent from the portal entry chunk.
- [x] AC-8: DB pool `MaxOpenConns=25`, `MaxIdleConns=10`, `ConnMaxLifetime=5m` from env with those defaults. (Shipped.)

## Delta Work Items
- **D1 (dev)**: Extend `docs/test-reports/k6-load-test.js` to the AC-2 endpoint groups with tags and per-group `p(95)<200` thresholds; add `setup()` that logs in via `POST /api/v1/auth/login` (env `K6_BASE_URL` default `http://localhost:8080`, `K6_ADMIN_EMAIL/PASS`, `K6_EMPLOYEE_EMAIL/PASS`); keep `TEST_TOKEN` as a fallback for `/exams`.
- **D2 (dev, only if Lighthouse < 90)**: in `deploy/nginx.conf` and `deploy/nginx/*.conf` add `gzip on` with text/JS/CSS/JSON/SVG types, `Cache-Control: public, max-age=31536000, immutable` for `/assets/*`, and `no-cache` for `index.html`. Lazy-load `ExamTakingPage`/`EmployeePortal` only if bundle analysis shows > 170 kB gzipped initial JS. No behavior changes otherwise.
- **D3 (dev)**: add `scripts/perf/lighthouse.sh` (uses `npx lighthouse@11`, headless Chrome, `--output=json,html`, writes to `docs/test-reports/lighthouse/`) and `scripts/perf/bundle-check.sh` (inspects `frontend/dist/assets`). Project-local only; no global installs.
- **D4 (UAT)**: executes the Evidence Plan below.

## Evidence Plan (owner: UAT, who owns the live stack)
Prerequisites: stack up via `make dev` or `docker compose up -d --build` (Nginx on `${HOST_HTTP_PORT:-80}`, API on 8080), migrations applied (`make migrate`), seed from `frontend/e2e/seed-test-env.ts` plus a perf seed (SQL embedded in the EXPLAIN report), admin `admin@bilimbaga.local`. k6 runs through `docker run --rm -i --network host grafana/k6` (no global install). `DISABLE_RATE_LIMIT=true` is set on the API only for the k6 run and restored afterward. Since PR #197 (#176 I-11) it disables all three limiters, including the per-session answer-save limiter (60 req/min), so a k6 answer-save group is no longer throttled; without it that group would hit 429. Never set it on bilimbaga-test or any shared instance; verify the variable is unset after the run (default is rate limiting ON).

| # | Evidence | Command | Report path |
|---|----------|---------|-------------|
| E1 | EXPLAIN ANALYZE (AC-1) | `docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB` with the seven queries | `docs/test-reports/explain-analyze-FR-BB65.md` |
| E2 | k6 (AC-2) | `docker run --rm -i --network host -e K6_BASE_URL=http://localhost:8080 -v "$PWD/docs/test-reports:/out" grafana/k6 run --summary-export=/out/k6-summary-<date>.json /out/k6-load-test.js` (console output teed to `.txt`) | `docs/test-reports/k6-summary-<date>.{json,txt}` |
| E3 | Lighthouse (AC-3) | `bash scripts/perf/lighthouse.sh <url>` for `/portal` and the exam-taking URL, 3 runs each, mobile and desktop | `docs/test-reports/lighthouse/*.{json,html}` plus `docs/test-reports/lighthouse-FR-BB65-<date>.md` (median table) |
| E4 | Bundle split (AC-7) | `cd frontend && npm run build && bash ../scripts/perf/bundle-check.sh` | appended to the Lighthouse summary |

### Pass thresholds
- E1: each of the seven queries shows its named index in the plan.
- E2: k6 exit code 0 - per-group p95 < 200 ms, failed-request rate < 1%, checks >= 99%.
- E3: median Performance >= 90 on both pages (mobile is gating).
- E4: no admin page module in the portal entry chunk; at least 10 separate lazy chunks.
- A miss files a dev issue naming the failing metric; UAT re-runs after the fix (cap 3 iterations, then escalate).

## Technical Specification

### Configuration / Infrastructure

**New environment variables** (add to `.env.example`):
```
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=5m
```

**Database connection pool setup** (`internal/database/db.go`):
```go
db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
```

**New migration** — `migrations/NNN_performance_indexes.sql`:
```sql
-- session_answers: fetch answers by session
CREATE INDEX IF NOT EXISTS idx_session_answers_session_id
  ON session_answers (session_id);

-- audit_log: time-range + actor queries
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at_actor
  ON audit_log (created_at DESC, actor_id);

-- exam_sessions: per-user + per-exam queries
CREATE INDEX IF NOT EXISTS idx_exam_sessions_user_exam_status
  ON exam_sessions (user_id, exam_id, status);

-- question_translations: locale lookups
CREATE INDEX IF NOT EXISTS idx_question_translations_question_locale
  ON question_translations (question_id, locale);

-- exam_assignments: assignment queries
CREATE INDEX IF NOT EXISTS idx_exam_assignments_exam_id
  ON exam_assignments (exam_id);

-- certificates: session lookup + verification
CREATE INDEX IF NOT EXISTS idx_certificates_session_id
  ON certificates (session_id);
CREATE INDEX IF NOT EXISTS idx_certificates_verification_code
  ON certificates (verification_code);
```

### Implementation Details

**In-memory tenant config cache** (`internal/tenant/cache.go`):
```go
type TenantCache struct {
    mu     sync.RWMutex
    config *TenantConfig
    logo   []byte
}

func (c *TenantCache) Get() *TenantConfig {
    c.mu.RLock(); defer c.mu.RUnlock()
    return c.config
}

func (c *TenantCache) Set(cfg *TenantConfig, logo []byte) {
    c.mu.Lock(); defer c.mu.Unlock()
    c.config = cfg
    c.logo = logo
}

func (c *TenantCache) Invalidate() { c.Set(nil, nil) }
```

- `TenantCache` is initialized as a singleton in `main.go` and injected into the tenant handler and any middleware that reads tenant config.
- On first request when cache is nil: loads from DB, stores in cache, returns value.
- `PUT /api/v1/admin/tenant/config` handler calls `cache.Invalidate()` after successful DB write.

**Logo HTTP handler**:
```go
func (h *TenantHandler) GetLogo(w http.ResponseWriter, r *http.Request) {
    logo := h.cache.GetLogo()
    if logo == nil {
        http.NotFound(w, r); return
    }
    w.Header().Set("Content-Type", "image/png") // or detect from bytes
    w.Header().Set("Cache-Control", "public, max-age=3600")
    w.Write(logo)
}
```

### Frontend Components

**React Query stale times** (`src/api/queryClient.ts`):
```ts
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 0 }, // default: always fresh
  },
});

// Per-query overrides in hooks:
// Tenant config
useQuery({ queryKey: ['tenantConfig'], queryFn: fetchTenantConfig, staleTime: 5 * 60 * 1000 });
// Categories
useQuery({ queryKey: ['categories'], queryFn: fetchCategories, staleTime: 5 * 60 * 1000 });
// Exam list
useQuery({ queryKey: ['exams'], queryFn: fetchExams, staleTime: 5 * 60 * 1000 });
// Session state
useQuery({ queryKey: ['session', id], queryFn: fetchSession, staleTime: 30 * 1000 });
// Grading queue — staleTime: 0 (default)
useQuery({ queryKey: ['gradingQueue'], queryFn: fetchGradingQueue });
```

**Route-level code splitting** (`src/router.tsx`):
```tsx
import { lazy, Suspense } from 'react';
import { createBrowserRouter } from 'react-router-dom';

const AdminShell = lazy(() => import('./pages/admin/AdminShell'));
const QuestionBank = lazy(() => import('./pages/admin/QuestionBank'));
const ExamConfig   = lazy(() => import('./pages/admin/ExamConfig'));
// ... all admin pages lazy

const EmployeePortal = lazy(() => import('./pages/portal/EmployeePortal'));
const ExamTaking     = lazy(() => import('./pages/portal/ExamTaking'));

export const router = createBrowserRouter([
  { path: '/admin/*', element: <Suspense fallback={<PageSpinner />}><AdminShell /></Suspense> },
  { path: '/portal/*', element: <Suspense fallback={<PageSpinner />}><EmployeePortal /></Suspense> },
]);
```

**Vite build configuration** (`vite.config.ts`):
```ts
build: {
  rollupOptions: {
    output: {
      manualChunks: {
        vendor: ['react', 'react-dom', 'react-router-dom'],
        query:  ['@tanstack/react-query'],
        ui:     ['@radix-ui/react-dialog', '@radix-ui/react-select'], // shadcn peers
      },
    },
  },
},
```

**k6 load test** — `docs/test-reports/k6-load-test.js`:
```js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 100,
  duration: '2m',
  thresholds: { 'http_req_duration': ['p(95)<200'] },
};

export default function () {
  const res = http.get('http://localhost:8080/api/v1/exams', {
    headers: { Authorization: `Bearer ${__ENV.TEST_TOKEN}` },
  });
  check(res, { 'status 200': r => r.status === 200 });
  sleep(0.5);
}
```

## Notes
- Lighthouse must be run against the Nginx-served production build, not `npm run dev`, because dev mode lacks minification and uses unoptimized source maps.
- The in-memory tenant config cache is process-local; in a future multi-replica deployment, a Redis cache with pub/sub invalidation would be required.
- `EXPLAIN ANALYZE` results for the six indexed queries should be captured and committed alongside the migration as evidence of AC-1 compliance.
- Connection pool sizing (`MaxOpenConns=25`) is calibrated for a single-instance PostgreSQL server; revisit when scaling horizontally.
