---
id: ISS-017
title: Question tag field cleared after save — tags not persisted
status: resolved
severity: high
layer: backend
module: questions
tags: [tag_ids, tags, QuestionDetail, json, detailToForm]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: []
regression_test: backend/internal/questions/handler_test.go
---

## Symptom
When editing a question in the admin question editor (`/admin/questions/{id}/edit`), adding tags
and then clicking Save causes the tag field to appear empty after the save completes. The entered
tags are not visible and are not persisted.

## Root Cause
JSON key mismatch between the backend response and the frontend interface.

The backend `QuestionDetail` struct was serialised with `json:"tags"`:
```go
Tags []string `json:"tags"`
```

The frontend `QuestionDetail` TypeScript interface expected the key `tag_ids`:
```typescript
tag_ids: string[]
```

`detailToForm` read `q.tag_ids ?? []`. Because the backend JSON never contained `tag_ids` (only
`tags`), `q.tag_ids` was always `undefined` — so the tag array was always reset to `[]` on every
load/refetch, making all entered tags disappear.

The tags **were** being saved correctly to `question_tags` on every PUT request (the `UpdateInPlace`
code path correctly calls `applySubObjects` with `input.TagIDs`). The only problem was the
read-back — the JSON key mismatch meant the frontend could never see the saved tags.

## Fix Applied
Renamed the field in `QuestionDetail` from `Tags` to `TagIDs` with the JSON tag `"tag_ids"`:

```go
// before
Tags []string `json:"tags"`

// after
TagIDs []string `json:"tag_ids"`
```

Updated all three `QuestionDetail` struct literals that assigned to the old `Tags` field:
- `repository.go` `CreateFull` → `TagIDs: detail.tags`
- `repository.go` `GetWithDetails` → `TagIDs: tags`
- `service_test.go` mock `CreateFull` → `TagIDs: input.TagIDs`
- `service_test.go` mock `GetWithDetails` → `TagIDs: []string{}`
- `handler_test.go` `sampleDetail` → `TagIDs: []string{}`

`QuestionListItem.Tags` (tag *names* in list responses) and `ExportRow.Tags` (tag names in
export) were deliberately left unchanged — they use `json:"tags"` correctly.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/questions/model.go` | `Tags []string \`json:"tags"\`` → `TagIDs []string \`json:"tag_ids"\`` in `QuestionDetail` |
| `backend/internal/questions/repository.go` | Two `QuestionDetail` literals: `Tags:` → `TagIDs:` |
| `backend/internal/questions/service_test.go` | Two mock `QuestionDetail` literals: `Tags:` → `TagIDs:` |
| `backend/internal/questions/handler_test.go` | `sampleDetail`: `Tags:` → `TagIDs:`; new regression test |

## Regression Test
`TestQHandlerGet_ResponseContainsTagIDs` in `backend/internal/questions/handler_test.go`.

Asserts that `GET /api/v1/questions/:id` response JSON contains a `tag_ids` key (not `tags`) and
that the value matches the tag IDs set on the `QuestionDetail`.

## Resolution Results
- Tests: 70 passed, 0 failed
- Migration applied: no (no schema change required)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Bug report: tags cleared after save | Fixed JSON key mismatch; regression test added |
