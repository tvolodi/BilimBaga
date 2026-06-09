# FR-BB24 — Translation API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB24 |
| Phase | 2 — Content Management |
| Priority | 2 |
| Status | uat-verified |
| Depends On | FR-BB23 |

## Description
Provides dedicated endpoints for managing per-locale translations of question stems, explanations, and answer option texts independently of the core question structure. A single upsert endpoint replaces all text for a locale atomically, enabling translators to work locale-by-locale without touching the structural metadata. The API enforces that the default locale translation cannot be deleted and that only locales declared in the tenant's configuration are accepted.

## Acceptance Criteria
- [x] AC-1: `GET /api/v1/questions/:id/translations` returns a map of all locale codes to their translation objects (stem, explanation, answer option texts); locales with no translation are absent from the map; the response includes a `locale_coverage` summary listing present and missing locales relative to the tenant's `available_locales`.
- [x] AC-2: `PUT /api/v1/questions/:id/translations/:locale` performs an atomic upsert: it creates or replaces `question_translations` and all `answer_translations` for the given locale in a single transaction; partial updates (e.g., providing only some option translations) return `422` with an error listing the missing option IDs.
- [x] AC-3: `PUT /api/v1/questions/:id/translations/:locale` rejects a locale that is not present in the tenant's `available_locales` configuration with `422` and error code `ERR_UNSUPPORTED_LOCALE`.
- [x] AC-4: `DELETE /api/v1/questions/:id/translations/:locale` removes the `question_translations` row and all associated `answer_translations` for that locale; if the locale equals `default_locale`, it returns `409` with error code `ERR_CANNOT_DELETE_DEFAULT_LOCALE`.
- [x] AC-5: The `locale_coverage` computed field returned by `GET /api/v1/questions` (FR-BB23) and `GET /api/v1/questions/:id` is consistent with the translation rows — any discrepancy caused by a direct DB manipulation must be detectable by an integration test.
- [x] AC-6: All translation write operations (PUT, DELETE) require role `examiner` or above; read operations require any authenticated user.
- [x] AC-7: An audit log entry is written for every PUT and DELETE, capturing `entity_type = 'question_translation'`, `entity_id` (question ID + locale), and the before/after text diff.
- [x] AC-8: `PUT /api/v1/questions/:id/translations/:locale` returns the full updated translation object in the response body, not just a success acknowledgement.
- [x] AC-9: Attempting to write a translation for a non-existent question ID returns `404` with error code `ERR_QUESTION_NOT_FOUND`.
- [x] AC-10: The upsert is idempotent — submitting the same translation payload twice produces no error and does not increment any version counter or trigger additional audit entries beyond the second write.

## Technical Specification

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/questions/:id/translations` | Any authenticated | All locale translations for a question |
| PUT | `/api/v1/questions/:id/translations/:locale` | examiner+ | Upsert all text for a locale |
| DELETE | `/api/v1/questions/:id/translations/:locale` | examiner+ | Remove a non-default locale translation |

#### Request / Response Shapes

**GET /api/v1/questions/:id/translations — Response (200)**
```json
{
  "data": {
    "question_id": "uuid-q1",
    "default_locale": "kk",
    "locale_coverage": {
      "present": ["kk", "ru"],
      "missing": ["en"]
    },
    "translations": {
      "kk": {
        "locale": "kk",
        "stem": "Вирустық бағдарлама дегеніміз не?",
        "explanation": "Вирус — зиянды бағдарлама.",
        "options": [
          { "option_id": "uuid-opt1", "text": "Зиянды бағдарлама" },
          { "option_id": "uuid-opt2", "text": "Жүйелік файл" }
        ],
        "updated_at": "2026-05-14T10:00:00Z"
      },
      "ru": {
        "locale": "ru",
        "stem": "Что такое вирусная программа?",
        "explanation": "Вирус — вредоносная программа.",
        "options": [
          { "option_id": "uuid-opt1", "text": "Вредоносная программа" },
          { "option_id": "uuid-opt2", "text": "Системный файл" }
        ],
        "updated_at": "2026-05-13T08:30:00Z"
      }
    }
  },
  "error": null
}
```

**PUT /api/v1/questions/:id/translations/:locale — Request**
```json
{
  "stem": "What is a computer virus?",
  "explanation": "A virus is malicious software that replicates itself.",
  "options": [
    { "option_id": "uuid-opt1", "text": "Malicious software" },
    { "option_id": "uuid-opt2", "text": "System file" }
  ]
}
```

**PUT /api/v1/questions/:id/translations/:locale — Response (200)**
```json
{
  "data": {
    "locale": "en",
    "stem": "What is a computer virus?",
    "explanation": "A virus is malicious software that replicates itself.",
    "options": [
      { "option_id": "uuid-opt1", "text": "Malicious software" },
      { "option_id": "uuid-opt2", "text": "System file" }
    ],
    "updated_at": "2026-05-14T11:00:00Z"
  },
  "error": null
}
```

**PUT — Unsupported Locale (422)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_UNSUPPORTED_LOCALE",
    "message": "Locale 'fr' is not in tenant available_locales"
  }
}
```

**PUT — Missing Option Translations (422)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_MISSING_OPTION_TRANSLATIONS",
    "message": "Translation missing for option IDs: ['uuid-opt2']"
  }
}
```

**DELETE /api/v1/questions/:id/translations/:locale — Response (204)**
_(empty body)_

**DELETE — Default Locale Attempt (409)**
```json
{
  "data": null,
  "error": {
    "code": "ERR_CANNOT_DELETE_DEFAULT_LOCALE",
    "message": "Cannot delete the default locale translation ('kk')"
  }
}
```

### Go Package / Layer

| Concern | Location |
|---------|----------|
| Package | `internal/questions` |
| HTTP handlers | `translation_handler.go` — thin; no SQL, no business logic |
| Business logic | `translation_service.go` — locale validation, option completeness check, coverage computation, before/after diff for audit |
| SQL / persistence | `translation_repository.go` — all queries against `question_translations` and `answer_translations`; upsert and delete wrapped in transactions |
| Wiring | `cmd/api/main.go` — `NewTranslationRepository` → `NewTranslationService` → `NewTranslationHandler`; `tenantSvc` satisfies `TenantLocaleProvider` |

## Notes
- The upsert for `answer_translations` should use `INSERT ... ON CONFLICT (option_id, locale) DO UPDATE SET text = EXCLUDED.text` to avoid DELETE + INSERT patterns that would invalidate FK references during the transaction.
- `available_locales` is stored in the `tenant_config` table (FR-BB13); the translation service reads it once per request (or from a short-lived cache) to validate the `:locale` parameter.
- Translation completeness for an exam session is checked at exam start time (Phase 3) — the API itself does not block activating a question that has incomplete translations in non-default locales.
- The `locale_coverage.missing` field is computed as `SET(available_locales) - SET(present_locales)`, giving translators a clear to-do list.
