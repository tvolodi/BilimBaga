# Function: IMPLEMENT_FIX

> Apply a targeted, minimal bug fix. Used by Issue Resolution and Code Fixer.

## Metadata

| Property | Value |
|----------|-------|
| **Category** | Implementation |
| **Used By** | Issue Resolution, Code Fixer |
| **Depends On** | Root cause identified; DESIGN_SOLUTION output (optional) |

---

## Purpose

Apply a surgical fix to the identified root cause. Minimize blast radius — change only what is necessary to resolve the issue.

---

## Rules

- Change only files directly related to the root cause.
- Do not refactor surrounding code unless it is the direct cause of the bug.
- Do not introduce new dependencies.
- Do not change function signatures unless the signature is the bug.
- Preserve all existing behavior beyond the bug fix.

---

## Steps

1. **Confirm root cause** before editing: re-read the relevant function to ensure the fix plan is correct.

2. **Apply the fix**:
   - Edit the minimum number of lines needed.
   - Preserve surrounding code style and indentation.

3. **Check for the same pattern elsewhere**:
   - Search for the same pattern in other files.
   - If the bug is a copy-paste pattern (e.g. all handlers missing error wrapping), fix all instances.

4. **Compile check** (Go):
   ```bash
   cd backend && go build ./...
   ```
   Fix any new compile errors before continuing.

5. **Manual verification**:
   - Reproduce the original bug scenario.
   - Confirm the bug no longer occurs.
   - Confirm no adjacent behavior broke.

---

## Common BilimBaga Bug Patterns

| Pattern | Fix |
|---------|-----|
| Missing `%w` in error wrap | `fmt.Errorf("funcName: %w", err)` |
| Handler not checking session ownership | Add `WHERE session_id = ? AND user_id = ?` |
| Grading using wrong question version | Join on `session_questions.question_version_id` |
| JWT not validated on a route | Add `jwtMiddleware` to route group |
| Option IDs not validated against session | Query `session_questions` before accepting answer |
| Missing audit log write | Call `audit.Write(ctx, ...)` after state change |
| Refresh token not rotated on use | Delete old token record; insert new one |
| Score computation divides by zero | Guard: `if maxPossible == 0 { return 0 }` |
| React Query key stale after mutation | Add `queryClient.invalidateQueries(...)` after mutation |
| i18n key missing in one locale | Add the key to all three locale files simultaneously |
