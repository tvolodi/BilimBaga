# ISS-071 vitest upgrade (2.x -> 5.0.3)

- vitest ^2.0.3 -> ^5.0.3, @vitest/coverage-v8 ^2.1.9 -> ^5.0.3, @types/node -> ^22 (required peer of vitest 5; vite stays 6.4.4, jsdom unchanged, no @vitest/ui).
- npm audit: before 7 vulnerabilities (3 moderate, 1 high, 3 critical: esbuild, vite, vite-node, tinypool, @vitest/mocker); after 0.
- Config fixes: setup.ts imports `@testing-library/jest-dom/vitest` (matcher typings); tsconfig `types` adds `node`.
- Verification: vitest x2 92 files / 653 tests pass; coverage run passes; tsc, lint, build clean; bundle-check PASS (initial JS 168.4 kB / 170 kB).
- CI: vitest 5 needs Node >=22.12; security.yml bumped from Node 20 to 22 (tests.yml already 22). Full `npm audit` made blocking in security.yml since it is clean.
