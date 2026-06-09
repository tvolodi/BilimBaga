---
run_id: analytics-and-reporting-20260609-iter3
scenario_path: docs/uat-scenarios/analytics-and-reporting-20260609.md
executed: 2026-06-10T00:00:00Z
executor: UAT Runner
result: TARGETED VERIFICATION - Scenario 2 Steps 1-5 - ALL PASS
---

# UAT Report - Analytics and Reporting (Iteration 3 - Targeted Verification)

## Verification Target

ISS-049 fix verification: avg_score and median_score display after Docker image rebuild.

Context: In iteration 2, Step 4 of Scenario 2 failed because the frontend Docker image had not been rebuilt after the ISS-049 fix was applied to StatsSummaryRow.tsx. The image has now been rebuilt and this iteration verifies that the fix is live.

---

## Scenario 2 Results

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Navigate to /admin/exams, find UAT Security Assessment | Exam visible in list | UAT Security Assessment visible in exams list | Playwright | PASS | e2e-uat-results-temp/iter3-s2/s2-step1-exam-found.png |
| 2 | Super Admin | Navigate to analytics page | Per-exam analytics page visible | Navigated to /admin/exams/19655eb7-47a9-43ae-a1a3-8ce77b051f50/analytics; page loads with Exam Analytics title | Playwright | PASS | e2e-uat-results-temp/iter3-s2/s2-step2-analytics-page.png |
| 3 | Super Admin | Assert score distribution chart visible | Chart rendered | Score Distribution histogram visible with bars at 0-10, 30-40, 90-100 buckets | Playwright | PASS | e2e-uat-results-temp/iter3-s2/s2-step3-chart.png |
| 4 | Super Admin | Assert summary statistics - avg and median score show reasonable values | Avg Score approx 33.3%, not 3330% | Average Score: 33.3%; Median Score: 16.7%; Total Attempts: 4; Unique Participants: 1; Pass Rate: 25.0% | Playwright | PASS - ISS-049 RESOLVED | e2e-uat-results-temp/iter3-s2/s2-step4-stats-full-page.png |
| 5 | Super Admin | Assert per-question statistics table visible | Table with question data | Question Analysis table visible below charts | Playwright | PASS | e2e-uat-results-temp/iter3-s2/s2-step5-question-table.png |

---

## ISS-049 Verification Detail

Screenshot: e2e-uat-results-temp/iter3-s2/s2-step4-stats-full-page.png

| Statistic | Iteration 2 Value (FAIL) | Iteration 3 Value (PASS) | Expected |
|-----------|--------------------------|--------------------------|----------|
| Average Score | 3330.0% | 33.3% | approx 33.3% |
| Median Score | 1666.5% | 16.7% | approx 16.7% |
| Total Attempts | 4 | 4 | 1 or more |
| Unique Participants | 1 | 1 | 1 or more |
| Pass Rate | 25.0% | 25.0% | Any value 0 to 100% |

All percentage values observed on page: 33.3%, 16.7%, 25.0%, 0.0%, 0.0%, 25.0%, 50.0%, 66.7%

No value exceeds 100%, confirming the x100 multiplication bug is no longer present.

---

## Summary

- Total steps verified: 5
- Passed: 5
- Failed: 0
- Blocked: 0
- Screenshots taken: 8

ISS-049: RESOLVED

The Docker image rebuild successfully deployed the StatsSummaryRow.tsx fix. The analytics page now correctly displays avg_score and median_score as percentage values without the erroneous x100 multiplication.

---

## Combined UAT Result: Iteration 2 + Iteration 3

| Scenario | Iteration 2 Status | Iteration 3 Status | Final |
|---------|-------------------|--------------------|-------|
| S1: Admin Dashboard KPIs (5 steps) | ALL PASS | Not re-run | PASS |
| S2: Per-Exam Analytics Steps 1-5 | Steps 1,3,5 PASS; Step 4 FAIL (ISS-049) | ALL PASS | PASS |
| S2: Steps 6-7 Export CSV | PASS | Not re-run | PASS |
| S3: Employee Record (4 steps) | ALL PASS | Not re-run | PASS |
| S4: Audit Log (7 steps) | ALL PASS | Not re-run | PASS |
| S5: Dept Admin Isolation (4 steps) | ALL PASS | Not re-run | PASS |

Combined result: ALL PASS (27 scenario steps total across iterations 2 and 3)

---

## Precondition Setup

| Precondition | Status | Notes |
|-------------|--------|-------|
| Backend health HTTP 200 | PASS | http://localhost:8080/api/v1/health 200 |
| Frontend health HTTP 200 | PASS | http://localhost 200 |
| Docker image rebuilt | PASS | Frontend serves updated bundle with ISS-049 fix |
| Admin credentials | PASS | admin@test.com / Admin1234! login confirmed |
| UAT Security Assessment exam exists | PASS | ID: 19655eb7-47a9-43ae-a1a3-8ce77b051f50, status: active, 4 sessions |

---

## Environment

- Frontend: http://localhost (port 80, Nginx)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright default)
- Stack status: already running (Docker Compose)
- Playwright test: frontend/e2e/uat-temp/analytics-s2-verify-iter3.spec.ts (deleted after run)
- Screenshots: e2e-uat-results-temp/iter3-s2/
