# FR-BB54 — Export API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB54 |
| Phase | 5 — Analytics & Reporting |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB52, FR-BB53 |

## Description
Provides three data export endpoints for HR admins and examiners. Two endpoints stream CSV downloads (per-exam results and per-employee session history). A third generates a PDF summary report for a specified date range, including completion rate tables, pass rate tables, and the top/bottom-performing questions. All exports are scoped to the caller's tenant and stream directly from the database without storing temporary files.

## Acceptance Criteria
- [ ] AC-1: All three endpoints require `role IN (examiner, hr_admin, super_admin)`; employees receive HTTP 403.
- [ ] AC-2: CSV responses set `Content-Type: text/csv; charset=utf-8` and `Content-Disposition: attachment; filename="..."`.
- [ ] AC-3: The exam results CSV includes a header row followed by one data row per session; columns: `employee_name`, `department`, `started_at`, `submitted_at`, `score_pct`, `passed`, `time_taken_seconds`, plus `question_{N}_score` for every question in the exam (N = 1-indexed question position).
- [ ] AC-4: If a session has no score recorded for a particular question (e.g. still grading_pending), the corresponding CSV cell is empty, not zero.
- [ ] AC-5: The user record CSV includes columns: `exam_title`, `started_at`, `submitted_at`, `score_pct`, `passed`, `time_taken_seconds`, `status`.
- [ ] AC-6: The dashboard PDF export accepts `from` and `to` query params (ISO 8601 dates); if omitted, defaults to the last 30 days.
- [ ] AC-7: The dashboard PDF includes: company logo header, date range label, a completion rates table, a pass rates table, and two question lists — top 5 by `correct_rate` and bottom 5 by `correct_rate` (across all exams in the range).
- [ ] AC-8: CSV is produced with Go's `encoding/csv`. Amended 2026-10-09 (BA, #163): a handler may build the body in memory before writing (so a query/encoding error can still return a proper error status and never an empty 200), provided the exam results export is bounded (page/row cap documented in the handler) and a failure after headers is sent is logged. Rules for every CSV export: finished sessions with statuses `submitted`, `auto_submitted` and `graded` are included; an unknown exam id returns 404 `NOT_FOUND`; cells beginning with `=`, `+`, `-` or `@` are prefixed with `'` (formula-injection guard); no BOM is required.
  - Status 2026-10-09 (PR #194, ISS-191, merged at 6f3f417): **partially shipped**. Shipped: formula-injection guard `api.CSVSafe` (`backend/internal/api/csv.go:26`) on user-controlled text cells of all four CSV exports (results `reports/service.go:434-435`, user record `reports/service.go:511,517`, audit `audit/handler.go:119-125`, questions `questions/import_export_handler.go:466`); the guard covers a leading `=`, `+`, `-`, `@`, TAB and CR (wider than the list above) and also escapes a cell that already starts with `'` plus one of those characters so the questions importer can invert it (`api.CSVUnsafe`, `import_export_handler.go:263`); server-formatted numeric cells (score_pct, time_taken_seconds, question scores) are deliberately NOT guarded so `-1` stays numeric; status set `submitted`, `auto_submitted`, `grading_pending` (`reports/repository.go:783,813`); 404 `NOT_FOUND` for an unknown exam id (`reports/handler.go:183-186`, `reports/service.go:342`); no BOM. Deviations: (a) `graded` is not a `session_status` value, so it is not in the filter; `grading_pending` rows are included instead (CSV rows equal the analytics attempt count); (b) the results export is buffered in memory (`writeBufferedCSV`, `reports/handler.go:155`) with **no row or size cap**, so the "bounded" condition of this AC is **not met** (gap G1 in `conformance/PR194-PR198-conformance-20261009.md`); the service comment still says "without buffering"; (c) a malformed (non-UUID) exam id is not mapped to 404 in the handler (see gap G2). Not guarded by design: leading-whitespace values (spreadsheets do not evaluate them), the static header rows, timestamps and booleans formatted by the server.
- [ ] AC-9: The dashboard PDF is generated using `GeneratePDF`-compatible infrastructure from FR-BB44 (same PDF library, same tenant config injection); PDF bytes are buffered and streamed.
- [ ] AC-10: Filenames in `Content-Disposition` are deterministic: `results-{exam_id}-{YYYYMMDD}.csv`, `record-{user_id}-{YYYYMMDD}.csv`, `dashboard-report-{from}-{to}.pdf`.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/exams/:id/results/export` | examiner+ | CSV of all session results for one exam |
| GET | `/api/v1/admin/users/:id/record/export` | examiner+ | CSV of all sessions for one user |
| GET | `/api/v1/admin/dashboard/export` | examiner+ | PDF summary report |

#### Query Parameters

`GET /admin/dashboard/export`
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `from` | ISO date | NOW() - 30 days | Report start date (inclusive) |
| `to` | ISO date | NOW() | Report end date (inclusive) |

#### CSV Column Spec — Exam Results Export

```
employee_name, department, started_at, submitted_at, score_pct, passed, time_taken_seconds,
question_1_score, question_2_score, ... question_N_score
```

Example rows:
```csv
employee_name,department,started_at,submitted_at,score_pct,passed,time_taken_seconds,question_1_score,question_2_score
Aibek Seitkali,Operations,2026-05-14T10:00:00Z,2026-05-14T10:30:00Z,84.50,true,1800,1.0,0.5
Zhansaya Nurova,HR,2026-05-13T09:00:00Z,2026-05-13T09:22:00Z,60.00,false,1320,,0.0
```

#### CSV Column Spec — User Record Export

```
exam_title, started_at, submitted_at, score_pct, passed, time_taken_seconds, status
```

### Go Implementation Notes

```go
// CSV streaming pattern — no in-memory buffering of all rows
func (h *ExportHandler) ExamResultsCSV(w http.ResponseWriter, r *http.Request) {
    examID := chi.URLParam(r, "id")
    // ... auth and tenant checks ...

    filename := fmt.Sprintf("results-%s-%s.csv", examID, time.Now().Format("20060102"))
    w.Header().Set("Content-Type", "text/csv; charset=utf-8")
    w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

    csvWriter := csv.NewWriter(w)
    defer csvWriter.Flush()

    // Write header
    questions, _ := h.repo.GetExamQuestions(r.Context(), examID)
    header := buildCSVHeader(questions)
    csvWriter.Write(header)

    // Stream rows from cursor
    rows, _ := h.repo.StreamExamResults(r.Context(), examID, tenantID)
    defer rows.Close()
    for rows.Next() {
        row := scanResultRow(rows, len(questions))
        csvWriter.Write(row)
    }
}
```

### Dashboard PDF Content Structure

```
┌──────────────────────────────────────────────────────────────┐
│  [Logo]  BilimBaga — Analytics Report                        │
│  Date Range: 01 Apr 2026 – 30 Apr 2026                       │
├──────────────────────────────────────────────────────────────┤
│  Completion Rates by Exam                                     │
│  ┌─────────────────┬──────────┬──────────┬──────────┐        │
│  │ Exam            │ Assigned │ Completed│ Pass Rate│        │
│  ├─────────────────┼──────────┼──────────┼──────────┤        │
│  │ Fire Safety     │ 150      │ 112      │ 87.5%    │        │
│  └─────────────────┴──────────┴──────────┴──────────┘        │
├──────────────────────────────────────────────────────────────┤
│  Top 5 Questions (Highest Correct Rate)                       │
│  1. "What is the first step when..." — 97%                    │
│  ...                                                          │
│  Bottom 5 Questions (Lowest Correct Rate)                     │
│  1. "Describe the MSDS classification..." — 23%              │
│  ...                                                          │
└──────────────────────────────────────────────────────────────┘
```

### Repository Queries

```sql
-- Exam results with per-question scores (streaming cursor)
SELECT
  u.full_name AS employee_name,
  d.name AS department,
  es.started_at,
  es.submitted_at,
  es.score_pct,
  es.passed,
  EXTRACT(EPOCH FROM (es.submitted_at - es.started_at))::INT AS time_taken_seconds,
  es.id AS session_id
FROM exam_sessions es
JOIN users u ON u.id = es.user_id
LEFT JOIN departments d ON d.id = u.department_id
WHERE es.exam_id = $1
  AND es.tenant_id = $2
  AND es.status IN ('submitted','grading_pending')
ORDER BY es.started_at;

-- Per-question scores for sessions (bulk fetch, joined in Go)
SELECT session_id, question_id, score
FROM session_question_scores
WHERE session_id = ANY($1::UUID[]);
```

## Notes
- The per-question columns in the exam CSV are dynamic (vary per exam); the header is built by querying the exam's question list before streaming rows.
- For large exports (>10,000 rows), the CSV streaming pattern prevents OOM; Go's `net/http` response writer supports chunked transfer encoding by default.
- Dashboard PDF generation reuses the `gofpdf` instance from FR-BB44; a separate `GenerateDashboardPDF(data *DashboardReportData, tenantCfg *TenantConfig) ([]byte, error)` function is implemented in `internal/reports/pdf.go`.
