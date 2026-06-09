# FR-BB25 — Bulk Import / Export

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB25 |
| Phase | 2 — Content Management |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB23 |

## Description
Enables content managers to migrate large question banks into the platform via CSV or JSON upload and to export the current question bank for backup or cross-system transfer. Import supports a dry-run mode that validates all rows and reports errors and duplicate-similarity warnings without persisting anything, followed by a commit mode that inserts valid rows. Export applies the same filters as the question list endpoint and streams the result in the requested format.

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/questions/import?dry_run=true` parses the uploaded CSV or JSON, validates every row against the schema rules (required fields, valid enum values, valid category path, minimum option count per type), and returns a structured report with `valid_count`, `error_rows`, and `warning_rows` — no database writes occur.
- [ ] AC-2: `POST /api/v1/questions/import` (without `dry_run`) inserts all valid rows in a single transaction; if any row fails validation the entire batch is rolled back and a `422` response is returned identical in shape to the dry-run error report; successfully committed rows are returned as a list of created question IDs.
- [ ] AC-3: The maximum import batch size is 500 questions per request; batches exceeding this limit return `413` with error code `ERR_BATCH_TOO_LARGE` before any parsing begins.
- [ ] AC-4: Duplicate detection computes trigram similarity between each imported stem and all existing `active` question stems for the same `default_locale`; rows with similarity ≥ 0.8 are included in `warning_rows` with the matching question ID and similarity score, but are not blocked from import.
- [ ] AC-5: CSV column order is: `type, difficulty, category_path, default_locale, stem, explanation, option_1 … option_N, correct, tags`; `category_path` uses `/`-delimited names (e.g., `Security Awareness/Phishing`); `correct` is a comma-separated list of 1-based option indices; the importer resolves `category_path` to a `category_id` and returns a row error if the path is not found.
- [ ] AC-6: `GET /api/v1/questions/export` accepts the same filter query parameters as the list endpoint plus an `Accept` header (`text/csv` or `application/json`); it streams the response and sets `Content-Disposition: attachment; filename="questions-export-{date}.{ext}"`. When the optional `?ids=` query parameter is present (comma-separated UUID v4 list, max 100 IDs), the export returns only the specified questions, overriding any other filter params. When not present, the export uses the filter query params as documented above.
- [ ] AC-7: Exported JSON format is a JSON array of full question objects matching the `GET /api/v1/questions/:id` response shape (including translations and answer options), so the file can be re-imported without transformation.
- [ ] AC-8: Exported CSV contains all columns defined in AC-5; multi-locale translations are exported as additional column groups suffixed with the locale code (e.g., `stem_kk`, `stem_ru`, `explanation_kk`, `explanation_ru`, `option_1_kk`, `option_1_ru`).
- [ ] AC-9: The import endpoint requires role `examiner` or above; the export endpoint requires role `examiner` or above; both operations are recorded in the audit log with the actor ID and a summary (row count, format, filter parameters).
- [ ] AC-10: The service uses `pg_trgm` extension for trigram similarity; if the extension is not installed, the import proceeds without duplicate detection and logs a warning — it does not fail the request.

## Technical Specification

### Database Schema

```sql
-- Enable trigram extension for duplicate detection (idempotent)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Optional: GIN index on default-locale stems for fast trigram search
-- Applied to question_translations table after FR-BB22 migration runs
CREATE INDEX IF NOT EXISTS idx_question_translations_stem_trgm
    ON question_translations
    USING GIN (stem gin_trgm_ops)
    WHERE locale = 'kk';   -- index per locale; add one per active locale
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/questions/import` | examiner+ | Import questions (dry-run or commit) |
| GET | `/api/v1/questions/export` | examiner+ | Export questions as CSV or JSON |

#### Request / Response Shapes

**POST /api/v1/questions/import?dry_run=true**

Request: `multipart/form-data` with field `file` (CSV or JSON file upload).

```
Content-Type: multipart/form-data
```

**Dry-run Response (200)**
```json
{
  "data": {
    "dry_run": true,
    "valid_count": 47,
    "error_rows": [
      {
        "row": 12,
        "errors": [
          "type: invalid value 'checkbox' (must be single|multiple|truefalse|likert|shorttext)",
          "category_path: 'Security/Unknown' not found"
        ]
      },
      {
        "row": 35,
        "errors": ["correct: index 5 exceeds option count 3"]
      }
    ],
    "warning_rows": [
      {
        "row": 8,
        "similarity_match": {
          "question_id": "uuid-existing",
          "score": 0.87,
          "stem_preview": "Фишинг дегеніміз не?"
        }
      }
    ]
  },
  "error": null
}
```

**Commit Response (201)**
```json
{
  "data": {
    "dry_run": false,
    "imported_count": 47,
    "question_ids": ["uuid-new-1", "uuid-new-2", "..."]
  },
  "error": null
}
```

**Batch Too Large (413)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_BATCH_TOO_LARGE",
    "message": "Import batch exceeds maximum of 500 questions"
  }
}
```

**Commit Validation Failure (422)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_IMPORT_VALIDATION",
    "message": "2 rows failed validation; no questions were imported",
    "details": {
      "valid_count": 48,
      "error_rows": [
        { "row": 12, "errors": ["type: invalid value 'checkbox'"] }
      ],
      "warning_rows": []
    }
  }
}
```

**GET /api/v1/questions/export — CSV (200)**
```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="questions-export-2026-05-14.csv"

type,difficulty,category_path,default_locale,stem_kk,explanation_kk,stem_ru,explanation_ru,option_1_kk,option_1_ru,option_2_kk,option_2_ru,correct,tags
single,medium,Security Awareness/Phishing,kk,Фишинг дегеніміз...,Фишинг — алаяқтық.,Что такое фишинг?,Фишинг — мошенничество.,Алаяқ сілтеме,Фишинговая ссылка,Қауіпсіз сілтеме,Безопасная ссылка,1,gdpr;iso27001
```

**GET /api/v1/questions/export — JSON (200)**
```
Content-Type: application/json
Content-Disposition: attachment; filename="questions-export-2026-05-14.json"

[
  {
    "type": "single",
    "difficulty": "medium",
    "category_path": "Security Awareness/Phishing",
    "default_locale": "kk",
    "translations": { "kk": { "stem": "...", "explanation": "..." }, "ru": { "stem": "...", "explanation": "..." } },
    "answer_options": [
      { "sort_order": 1, "is_correct": true, "translations": { "kk": { "text": "..." }, "ru": { "text": "..." } } }
    ],
    "tags": ["gdpr", "iso27001"]
  }
]
```

## Notes
- CSV parsing must handle quoted fields with embedded commas and newlines (RFC 4180 compliance); use Go's `encoding/csv` package.
- `category_path` resolution walks the `categories` table by name at each level; the lookup is case-insensitive and trims whitespace.
- For performance, trigram similarity queries are run as a single batch (`ANY($1::text[])`) against existing stems rather than one query per row.
- The export endpoint must not load all rows into memory at once; use `sql.Rows` with a streaming encoder (`encoding/json` encoder writing directly to the `http.ResponseWriter`, or a buffered CSV writer) to support exports of 10,000+ questions without OOM risk.
- Import rows that fail validation are never partially committed; all-or-nothing semantics simplify error recovery for the user.
- JSON import format mirrors the export format, enabling round-trip import/export without transformation.
