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

## Acceptance Criteria
- [ ] AC-1: A new migration adds indexes for all six specified query patterns (`session_answers`, `audit_log`, `exam_sessions`, `question_translations`, `exam_assignments`, `certificates`); `EXPLAIN ANALYZE` on representative queries confirms index usage.
- [ ] AC-2: API p95 response time is below 200ms for all endpoints under a k6 load test simulating 100 concurrent virtual users; the k6 script and its summary report are committed as acceptance artifacts in `docs/test-reports/`.
- [ ] AC-3: Lighthouse Performance score is ≥ 90 on both the employee portal page and the exam-taking page when measured against a production build served via Nginx (not the Vite dev server).
- [ ] AC-4: Tenant configuration (app_name, colors, locales, etc.) is cached in Go process memory after the first load; subsequent API requests do not hit the database for tenant config; the cache is invalidated atomically when `PUT /api/v1/admin/tenant/config` succeeds.
- [ ] AC-5: The tenant logo bytes are cached in Go process memory alongside tenant config; invalidated on the same config update path; served from cache with an appropriate `Cache-Control: public, max-age=3600` header.
- [ ] AC-6: React Query stale times are configured as specified: 5 minutes for tenant config, categories, and exam list; 30 seconds for session state; 0 (always fresh) for the manual grading queue.
- [ ] AC-7: Admin-only routes are lazy-loaded via React Router's `lazy()` / `React.lazy()` + dynamic `import()` so that employees downloading the employee portal bundle do not receive admin UI code.
- [ ] AC-8: The database connection pool is configured with `MaxOpenConns=25`, `MaxIdleConns=10`, and `ConnMaxLifetime=5m`; these values are read from environment variables with the stated values as defaults.

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
