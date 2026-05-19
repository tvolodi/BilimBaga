---
id: ISS-016
title: Save on active question archives it and leaves user on stale URL
status: resolved
severity: high
layer: frontend, backend
module: questions
tags: [CreateVersionFull, versioning, navigate, ErrInvalidInput, 422]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: [ISS-012, ISS-005]
regression_test: backend/internal/questions/handler_test.go
---

## Symptom

When editing a question with **active** status and pressing the Save button:
1. The question badge changes to **"В архиве"** (archived) immediately after saving.
2. A save-error toast appears: **"Не удалось сохранить. Попробуйте ещё раз."**

URL at time of failure: `/admin/questions/9332444d-f6e3-42ce-a6ac-884d4239af94/edit`

## Root Cause

### Part 1 — Frontend doesn't navigate to the new version URL

The backend's `UpdateQuestion` service uses two strategies based on the question's current status:
- **draft / review** → update in place (same ID)
- **active** → call `CreateVersionFull`: archives the old question (`status='archived'`) and inserts a **new** question row with a new UUID and `status='draft'`

The handler returns the **new** question (new `id`) in its response. However, `handleSaveDraft` and `doAutoSave` in `QuestionEditorPage.tsx` ignored the returned question entirely:

```ts
// Before fix — result discarded
await updateQuestion.mutateAsync({ id, payload: buildUpdatePayload(form) })
```

After the mutation, `useUpdateQuestion.onSuccess` invalidated `['questions', OLD_id]`, which triggered a refetch of the **archived** original. The `question?.status` became `'archived'`, so the badge showed "В архиве". The user remained on the old URL. Any subsequent Save press then sent a PUT to the archived question, which the backend rejected (`ErrInvalidInput` → 500).

### Part 2 — Backend returned 500 for a client error

`ErrInvalidInput` (status is not updatable) fell through to the generic 500 handler in `handler.go::Update`, rather than returning a semantically correct 4xx response.

## Fix Applied

### Frontend (`QuestionEditorPage.tsx`)
`handleSaveDraft` and `doAutoSave` now capture the returned `QuestionDetail`. When `updated.id !== id` (active-question versioning path), they call `navigate` to redirect the user to the new draft version's edit page:

```ts
const updated = await updateQuestion.mutateAsync({ id, payload: buildUpdatePayload(form) })
setIsDirty(false)
setAutoSaveStatus('saved')
if (updated.id !== id) {
  navigate(`/admin/questions/${updated.id}/edit`, { replace: true })
  return
}
```

`navigate` was also added to the `useCallback` dependency array for `doAutoSave`.

### Frontend (`api/questions.ts`)
`useUpdateQuestion.onSuccess` now invalidates the new question's cache entry when the returned ID differs from the mutation variable ID:

```ts
onSuccess: (result, { id }) => {
  queryClient.invalidateQueries({ queryKey: ['questions', id] })
  if (result?.id && result.id !== id) {
    queryClient.invalidateQueries({ queryKey: ['questions', result.id] })
  }
  queryClient.invalidateQueries({ queryKey: ['questions'] })
},
```

### Backend (`handler.go`)
Added explicit handling for `ErrInvalidInput` in the `Update` handler to return **422** instead of **500**:

```go
if errors.Is(err, ErrInvalidInput) {
    api.WriteError(w, http.StatusUnprocessableEntity, "ERR_INVALID_STATUS", "question status does not allow updates")
    return
}
```

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/pages/admin/questions/QuestionEditorPage.tsx` | `handleSaveDraft` and `doAutoSave` navigate to new version URL after active-question update |
| `frontend/src/api/questions.ts` | `useUpdateQuestion.onSuccess` invalidates new question cache when ID changes |
| `backend/internal/questions/handler.go` | Handle `ErrInvalidInput` → 422 instead of 500 |
| `backend/internal/questions/handler_test.go` | Add `TestQHandlerUpdate_ArchivedQuestion_Returns422` regression test |

## Regression Test

`backend/internal/questions/handler_test.go` — `TestQHandlerUpdate_ArchivedQuestion_Returns422`

Mocks `UpdateQuestion` returning `ErrInvalidInput` (archived question) and asserts the handler responds with HTTP 422.

## Resolution Results

- Tests: all passed (25 backend packages, questions package ran fresh)
- Migration applied: no
- Build clean: yes (TypeScript no errors, `go test ./...` all green)

## Recurrence Log

| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Bug report ISS-016 | Fixed frontend navigation + backend 422 response |
