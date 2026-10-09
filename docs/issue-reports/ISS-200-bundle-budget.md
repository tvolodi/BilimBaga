# ISS-200 initial JS bundle over budget

Run: swarm-200. Issue #200 (FR-BB65 D2: initial JS <= 170 kB gzip).

Root cause: App.tsx eagerly imported employee/auth pages (ExamTaking, ResultPage, EmployeePortal, MyResults, ExamResultRedirect, ChangePassword, ForgotPassword, ResetPassword), inflating the entry chunk.

Fix: those pages are now React.lazy (LoginPage stays eager); Suspense (existing FullPageSpinner, i18n-free) wraps the public auth routes and AuthedRoutes. CI: tests.yml frontend job now runs `npm run build` + `scripts/perf/bundle-check.sh` (offline, no external tools).

Sizes (initial JS gzip, bundle-check.sh): before 174.2 kB (entry 139.7 + preloads), after 168.0 kB (entry 132.0). Budget 170 kB. CSS 9.0 kB; 62 lazy chunks.

Tests: tsc clean, lint clean, check:i18n OK, vitest 652/652 pass. Margin is only 2 kB; future growth needs further splitting (locale JSON ~35 kB gz is the next candidate).
