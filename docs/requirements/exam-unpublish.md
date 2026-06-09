# FR-BB318 — Unpublish Exam

**Status**: uat-verified  
**Updated**: 2026-05-21 (revision 2)  
**Phase**: 3 (enhancement to 3.12)  
**Depends on**: FR-BB31, FR-BB32, FR-BB312, FR-BB35

## Summary

HR Admins and Super Admins can revert a published (`active`) exam back to `draft` status so they can edit it. This "Unpublish" action is distinct from archiving — the exam is not retired; it returns to a fully editable draft state and can be re-published. Unpublishing is blocked when the exam has at least one in-progress (live) session to prevent disrupting test-takers. Completed session records are unaffected and remain historically intact. While the exam is in `draft` status, the employee portal will not surface it, placing all outstanding assignments effectively on hold without requiring any schema change to the assignments table.

## Scope

| Layer | Items |
|-------|-------|
| Database | No schema changes — reuses `exams`, `exam_sessions` tables already present |
| API endpoints | `POST /api/v1/exams/{id}/unpublish` — transitions exam from `active` → `draft` |
| Frontend pages/components | `src/pages/ExamWizard/Step4Review.tsx` — Unpublish button; `src/pages/admin/ExamsListPage.tsx` — Unpublish action in row action menu |
| i18n keys | `exam.unpublish.*` keys in `kk`, `ru`, `en` locale files |

## Acceptance Criteria

- **AC1**: `POST /api/v1/exams/{id}/unpublish` returns `200 OK` with `{ data: { id, status: "draft" }, error: null }` when the exam is `active` and has no in-progress sessions. The exam `status` column is updated to `"draft"` in the database.
- **AC2**: If the exam has one or more sessions with `status = 'in_progress'`, the endpoint returns `409 Conflict` with `{ data: null, error: { code: "ERR_ACTIVE_SESSIONS", message: "..." } }` and makes no database change.
- **AC3**: If the exam `status` is not `active` (i.e., it is already `draft` or `archived`), the endpoint returns `409 Conflict` with `{ data: null, error: { code: "ERR_INVALID_TRANSITION", message: "..." } }`.
- **AC4**: The endpoint requires `exams:write` permission. A caller without that permission receives `403 Forbidden`.
- **AC5**: After a successful unpublish, the exam is fully editable: `PUT /api/v1/exams/{id}` no longer returns `ERR_NOT_DRAFT`, and all existing exam fields, sections, and question rules are preserved unchanged.
- **AC6**: After a successful unpublish, the employee portal (`GET /portal/exams`) no longer lists the exam for any employee, and no new sessions can be started for it, because the portal query filters on `status = 'active'`. No changes to the `exam_assignments` table are made.
- **AC7**: Completed sessions (status `submitted` or `auto_submitted`) for the exam are not deleted or modified; their scores and certificates remain queryable.
- **AC8**: On the Exam Wizard Step 4 Review page, when `exam.status === 'active'`, an "Unpublish" button is rendered next to the existing status label. The button is not rendered for `draft` or `archived` exams.
- **AC9**: Clicking the Unpublish button on Step 4 calls the unpublish endpoint, shows a loading state, and on success a `toast.success(t('exam.unpublish.success'))` is displayed and `navigate(\`/admin/exams/${examId}/edit\`)` is called. On failure (e.g., `ERR_ACTIVE_SESSIONS`), a descriptive toast error is shown and the user stays on Step 4.
- **AC10**: In the ExamsListPage admin action menu, an "Unpublish" menu item is visible only for rows where `status === 'active'`. Clicking it triggers the same unpublish API call; on success the row's status badge updates to `draft` (via React Query cache invalidation).
- **AC11**: A successful unpublish is recorded in the audit log with action `"exam.unpublish"` and the exam ID as the resource.
- **AC12**: A shadcn/ui `<Dialog>` with the `exam.unpublish.confirm` title and `exam.unpublish.confirmDetail` body is shown before the unpublish mutation fires; the user must click a confirm button to proceed.

## Technical Notes

### Database

No DDL changes required. The `exams` table already has a `status` column constrained to the Postgres ENUM `(draft, active, archived)`. The `active → draft` transition is valid within this ENUM and does not require a migration.

The "assignments on hold" behavior is implicit: the portal repository query already filters by `WHERE e.status = 'active'` (see `backend/internal/portal/repository.go` lines ~91 and ~119), so a `draft` exam is invisible to employees and no session can be started for it.

**Active-session check query (run inside the unpublish service method before updating)**:

```sql
SELECT COUNT(*) FROM exam_sessions
WHERE exam_id = $1 AND status = 'in_progress';
```

If `count > 0`, return `ErrActiveSessionsExist` (new sentinel error) and abort.

### API Contract

**Request**

```
POST /api/v1/exams/{id}/unpublish
Authorization: Bearer <token>
(no request body)
```

**Success response** (`200 OK`)

```json
{
  "data": { "id": "<uuid>", "status": "draft" },
  "error": null
}
```

**Error responses**

| Status | `error.code` | Condition |
|--------|--------------|-----------|
| 404 | `ERR_NOT_FOUND` | Exam ID does not exist |
| 409 | `ERR_ACTIVE_SESSIONS` | Exam has one or more in-progress sessions |
| 409 | `ERR_INVALID_TRANSITION` | Exam is not `active` (already draft or archived) |
| 401 | `UNAUTHORIZED` | Missing or invalid JWT |
| 403 | `FORBIDDEN` | Caller lacks `exams:write` permission |

### Go Implementation Notes

**Package**: `internal/exams`

**`model.go`** — add sentinel error:
```go
var ErrActiveSessionsExist = errors.New("exam has active in-progress sessions")
```

**`service.go`** — extend `validTransitions`:  
The current map is `map[string]string{"draft": "active", "active": "archived"}`. Do not touch `validTransitions` for unpublish — `TransitionStatus` is a generic forward-only helper and should not be reused for a guarded reverse transition. Instead, add a dedicated service method:

```go
func (s *service) UnpublishExam(ctx context.Context, id string) (*Exam, error)
```

Implementation steps:
1. Fetch exam via `s.repo.GetByID(ctx, id)` — propagate `ErrNotFound` on miss.
2. If `e.Status != "active"`, return `ErrInvalidTransition`.
3. Call `s.repo.CountActiveSessionsForExam(ctx, id)` — if `count > 0`, return `ErrActiveSessionsExist`.
4. Call `s.repo.UpdateStatus(ctx, id, "draft")` — reuses the existing repo method; no new method needed.
5. Call `s.repo.GetByID(ctx, id)` to fetch and return the updated exam.

Add `UnpublishExam(ctx context.Context, id string) (*Exam, error)` to the `Service` interface.

**`repository.go`** — add one method to the `Repository` interface and implement on `postgresRepository`:

```go
// CountActiveSessionsForExam returns the number of in-progress sessions for the exam.
CountActiveSessionsForExam(ctx context.Context, examID string) (int, error)
```

`CountActiveSessionsForExam` runs:
```sql
SELECT COUNT(*) FROM exam_sessions WHERE exam_id = $1 AND status = 'in_progress'
```

> **Note**: `SetStatus` was considered but removed to avoid duplication. The service uses the existing `UpdateStatus(ctx, id, status string) error` + `GetByID(ctx, id string) (*Exam, error)` methods in sequence — status update then re-fetch. No new repository method is required for the write path.

**`handler.go`** — add `Unpublish` handler mirroring the existing `Publish` handler:

```go
func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    exam, err := h.svc.UnpublishExam(r.Context(), id)
    if err != nil {
        switch {
        case errors.Is(err, ErrNotFound):
            api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "exam not found")
        case errors.Is(err, ErrInvalidTransition):
            api.WriteError(w, http.StatusConflict, "ERR_INVALID_TRANSITION", err.Error())
        case errors.Is(err, ErrActiveSessionsExist):
            api.WriteError(w, http.StatusConflict, "ERR_ACTIVE_SESSIONS", err.Error())
        default:
            api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to unpublish exam")
        }
        return
    }
    h.writer.Write(r.Context(), r, "exam.unpublish", "exam", &exam.ID, map[string]any{
        "status": exam.Status,
    })
    api.WriteJSON(w, http.StatusOK, map[string]any{
        "data":  map[string]string{"id": exam.ID, "status": exam.Status},
        "error": nil,
    })
}
```

**`internal/router/router.go`** — register the route alongside `/publish` and `/archive`:

```go
r.With(rbac.RequirePermission(rbacCache, "exams", "write")).
    Post("/exams/{id}/unpublish", examsHandler.Unpublish)
```

No new middleware is needed. Existing JWT + RBAC middleware applies.

### Frontend Implementation Notes

**API hook** — add to `src/api/exams.ts`:

```ts
export function useUnpublishExam() {
  const qc = useQueryClient()
  return useMutation<{ id: string; status: string }, ExamApiError, string>({
    mutationFn: (examId) => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<{ id: string; status: string }>(
        `/api/v1/exams/${examId}/unpublish`,
        token,
        { method: 'POST' },
      )
    },
    onSuccess: (_data, examId) => {
      qc.invalidateQueries({ queryKey: ['exams'] })
      qc.invalidateQueries({ queryKey: ['exams', examId] })
    },
  })
}
```

> **Note**: Uses the file-local `examsFetch(url, token, options)` — not `apiFetch` — matching the pattern of `usePublishExam` and all other mutations in this file. `examsFetch` sends `credentials: 'include'` and throws `ExamApiError` on failure; no import of `apiFetch` is needed.

**Step4Review** (`src/pages/ExamWizard/Step4Review.tsx`):
- Import `useUnpublishExam` and `useNavigate`.
- Render an "Unpublish" `<Button variant="outline">` (shadcn/ui) when `exam.status === 'active'`, placed adjacent to the existing "Published" status badge.
- On click: call `unpublishMutation.mutate(examId)`.
  - While pending: show `<Loader2 className="animate-spin" />` and disable the button.
  - On success: `navigate(`/admin/exams/${examId}/edit`)`.
  - On error: display a `toast.error(t('exam.unpublish.errorActiveSessions'))` when `error.code === 'ERR_ACTIVE_SESSIONS'`, or a generic `toast.error(t('exam.unpublish.errorGeneric'))` otherwise.
- The existing Publish button logic is not modified.

**ExamsListPage** (`src/pages/admin/ExamsListPage.tsx`):
- Import `useUnpublishExam`.
- In the row action `<DropdownMenu>` for each exam, add an "Unpublish" `<DropdownMenuItem>` conditionally rendered when `exam.status === 'active'`.
- On click: call `unpublishMutation.mutate(exam.id)`. On error, show a toast; on success, React Query cache invalidation (already wired in the hook) will update the row status badge automatically.

**i18n keys** — add to `src/locales/en.json`, `ru.json`, and `kk.json` under the `exam.unpublish` namespace:

| Key | English | Russian | Kazakh |
|-----|---------|---------|--------|
| `exam.unpublish.button` | `"Unpublish"` | `"Снять с публикации"` | `"Жариялауды алу"` |
| `exam.unpublish.confirm` | `"Revert to Draft?"` | `"Вернуть в черновик?"` | `"Жобаға қайтару?"` |
| `exam.unpublish.confirmDetail` | `"The exam will return to draft and employees won't be able to take it until it is re-published."` | `"Экзамен вернётся в черновик и сотрудники не смогут проходить его до повторной публикации."` | `"Емтихан жобаға оралады және қайта жарияланғанша қызметкерлер оны өткізе алмайды."` |
| `exam.unpublish.success` | `"Exam reverted to draft."` | `"Экзамен переведён в черновик."` | `"Емтихан жобаға аударылды."` |
| `exam.unpublish.errorActiveSessions` | `"Cannot unpublish: there are active test sessions in progress."` | `"Нельзя снять с публикации: есть активные сеансы тестирования."` | `"Жариялауды алу мүмкін емес: белсенді тестілеу сеанстары бар."` |
| `exam.unpublish.errorGeneric` | `"Failed to unpublish exam."` | `"Не удалось снять экзамен с публикации."` | `"Емтиханды жариялаудан алу сәтсіз аяқталды."` |

## Out of Scope

- Unpublishing `archived` exams — `archived` is a terminal state with no allowed outgoing transitions.
- Automatically re-assigning or notifying employees when the exam is re-published after an unpublish.
- Bulk unpublish of multiple exams in a single request.
- Force-unpublish overriding the active-sessions guard (admin override) — the guard is a hard business rule.
- Deleting or voiding in-progress sessions as part of unpublish.
- Any change to the `exam_assignments` table structure or status concept — the portal filter on `e.status = 'active'` provides the necessary "on hold" behavior without schema changes.

## Test Strategy

**Backend unit tests** (`service_test.go`):
- `TestUnpublishExam_Success`: active exam, zero in-progress sessions → returns updated exam with `status = "draft"`.
- `TestUnpublishExam_BlockedByActiveSessions`: active exam with one in-progress session → returns `ErrActiveSessionsExist`, no DB update.
- `TestUnpublishExam_NotActive_Draft`: exam already in draft → returns `ErrInvalidTransition`.
- `TestUnpublishExam_NotActive_Archived`: archived exam → returns `ErrInvalidTransition`.
- `TestUnpublishExam_NotFound`: non-existent ID → returns `ErrNotFound`.

**Backend handler tests** (`handler_test.go`):
- `TestUnpublishHandler_200`: mocked service returns exam → HTTP 200 with correct JSON shape.
- `TestUnpublishHandler_409_ActiveSessions`: service returns `ErrActiveSessionsExist` → HTTP 409, `ERR_ACTIVE_SESSIONS` code.
- `TestUnpublishHandler_409_InvalidTransition`: service returns `ErrInvalidTransition` → HTTP 409, `ERR_INVALID_TRANSITION` code.
- `TestUnpublishHandler_404`: service returns `ErrNotFound` → HTTP 404.

**Frontend component tests**:
- `Step4Review`: when `exam.status === 'active'`, Unpublish button renders; when `'draft'` or `'archived'`, it does not render.
- `Step4Review`: on successful mutation, `navigate` is called with the edit route.
- `Step4Review`: on `ERR_ACTIVE_SESSIONS` error, the active-sessions toast message is shown.
- `ExamsListPage`: Unpublish menu item visible for active row, hidden for draft/archived rows.

**Integration (if harness available)**:
- Insert an active exam with an in-progress session; call `POST /exams/{id}/unpublish`; assert 409.
- Insert an active exam with only submitted sessions; call unpublish; assert 200 and `status = 'draft'` in DB; assert portal no longer returns the exam.
