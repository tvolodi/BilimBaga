# Code Review ISS-138 - FR-BB117 role management frontend

Branch: swarm/138-roles-frontend, commit 5dddc0c (diff vs origin/main). Reviewer: Code Reviewer subagent. Static review only; no tests were run.

Result: PASS (0 Critical, 0 High, 3 Medium, 4 Low)

## Findings

- [Medium] frontend/src/components/ui/dialog.tsx (effect, `previouslyFocused = document.activeElement`) - a11y focus return: the trigger is captured inside a passive `useEffect`, which runs after children have committed. A child with `autoFocus` (RoleFormDialog create mode: `autoFocus={!isEdit}` on `#role-name`) has already taken focus, so `previouslyFocused` is the input, not the trigger. On close, focus is "restored" to an unmounted node and is lost (it goes to body). Focus return therefore works only for dialogs without autoFocus (edit, view, delete). Suggestion: capture the opener when `open` flips to true during render (ref updated in render or via a `useState` initialiser keyed on `open`), or in a `useLayoutEffect` placed before the children mount. Add a test for Tab/Shift+Tab wrap and focus return, including an autoFocus child.
- [Medium] frontend/src/components/admin/AdminHome.tsx + hooks/useMyPermissions.ts - error path: if `/users/me` fails for a custom role, `permissions` is `undefined` and `isLoading` is false, so AdminHome shows the "no access" notice (`roles.noAccess.*`) instead of a load error with retry. Access is correctly denied (no over-grant, no redirect loop), but the message is misleading for a transient failure. Suggestion: expose `isError` from the hook and show a `common.loadError`-style message.
- [Medium] frontend/src/api/roles.ts `roleErrorKey` - the ROLE_IN_USE count is parsed with a regex from the server's English message (`role is assigned to %d user(s)`); the backend sends no `details.count` (handler.go:46). It is correctly mapped by code, no raw text is displayed, and a unit test covers it, but any rewording of the backend message silently renders count 0. Suggestion: have the backend send `details: {count}` and read `err.details.count` first, keeping the regex only as a fallback.
- [Low] frontend/src/App.tsx `users/:userId/record` - maps to `PERM.reports` only; the page also shows user data (users:read). Harmless today because the backend is authoritative; consider requiring both.
- [Low] frontend/src/components/ui/dialog.tsx - focus-trap selector does not exclude hidden elements (`hidden`, `display:none`, `aria-hidden`) or `tabindex="-1"` on non-listed elements. Fine for current dialogs; could pick a hidden node as first/last in future ones. The initial focus is not moved into the dialog when no child has autoFocus (the trap pulls it in on the first Tab).
- [Low] frontend/src/pages/admin/roles/RolesPage.tsx - 375px layout: the table has 5 visible columns at phone width (description hidden below sm). It depends on the shared `Table` wrapper's overflow handling, which I could not verify visually; the matrix grid (`sm:grid-cols-2`, `min-w-0`, `break-words`) and dialog (`mx-4`, `max-h-[90vh]`, `overflow-y-auto`) are fine.
- [Low] Sidebar `end: true` for the Dashboard link at `/admin` and the `permission: reports:read` mapping: a custom role without reports:read sees no Dashboard link (correct), but `/admin/dashboard` is still reachable only via guard (also correct). No action; noted for completeness.

## Focus area results

1. Permission fallback - no over-grant found.
   - Built-in roles: `RequireRole` returns children iff `roles.includes(role)`; otherwise built-ins hit the unchanged redirect branch (`unauthorizedRedirect` or `/portal`|`/admin`). `useMe` is `enabled` only for custom roles, so built-in behaviour and request count are unchanged.
   - Stale `/users/me` cache: permissions are trusted only when `data.role_name === jwt role`; a mismatch yields `undefined` (deny) and `isLoading` stays true while refetching. Tested.
   - Error/loading: unknown permissions deny; `can()` returns false when undefined; a spinner shows only while loading. Tested for error.
   - Redirect loops: custom role denied on a page goes to `/admin`; AdminHome routes to the first permitted page or renders a notice (no Navigate when nothing is permitted). A role lacking reports:read does not bounce between /admin and /admin/dashboard. Custom roles never go to `/login` via `unauthorizedRedirect`. Tested.
   - Custom role named like a built-in: `isCustomRole` is a plain membership check against BUILTIN_ROLES and names are unique and pattern-constrained, so a built-in name always takes the built-in path. `roles:*`/`tenant:manage` are non-assignable, so a custom role cannot reach `/admin/roles` or settings (settings uses RequireSuperAdmin, unchanged).
   - `allowCustomRole` shell: admits only custom roles (employee is not custom and is redirected; tested). Every admin child route was checked: all carry RequireRole (permission-aware) or RequireSuperAdmin; the formerly unguarded `users`, `departments`, `exams/:id/analytics` routes now have guards with identical role lists for built-ins. The permission strings used all exist in the seeded catalogue (users:read, departments:read, questions:read/write, categories:manage, tags:read, exams:read/write, grading:read, reports:read, audit:read, roles:read, tenant:manage).
   - Backend remains authoritative; the code comments say so.
2. Error mapping - by `err.code` to `roles.errors.<CODE>`; unknown codes and non-Error values map to `roles.errors.generic`. Server `message` is never rendered (only parsed for a number). ROLE_NAME_TAKEN is shown inline on the name field (`role="alert"`, aria-describedby). All six keys plus generic and nameInvalid exist in the locale files. Delete errors are shown in the dialog.
3. i18n - key sets for en/ru/kk are identical (script-compared: no missing or extra keys). Every `t('...')` key used in the new components exists. Aria-labels (`roles.dismiss`, `roles.actions.*Role`) are translated. The only literal is the dismiss glyph "x" (multiplication sign), which is a symbol, acceptable. Permission and resource labels fall back to the raw `resource:action` key for catalogue entries lacking a translation (en has all 12 resources).
4. a11y - inputs have `Label htmlFor`; the matrix uses fieldset/legend with a label per checkbox and a hint linked by aria-describedby; Dialog has role=dialog, aria-modal, labelledby/describedby; Escape and pending-guard work; the `onOpenChange` ref avoids re-running the effect (no focus theft on re-render). Existing dialogs keep working (same props, the only behavioural additions are the Tab trap and focus return; the effect deps changed to `[open]` only, which is safe since the latest callback is read from a ref). See the Medium finding on focus return with autoFocus.
5. Tests - good coverage: permissionGuards.test.tsx (RequireRole fallback incl. stale cache, error, shell, built-in unchanged, role-list snapshots, Sidebar, AdminHome), PermissionMatrix, RolesPage (list, system protected, create, edit, delete, in-use, client validation), roleErrorKey. Gaps: no test of the Dialog focus trap/return (the changed shared component); AC-15 (new role appears in user drawers) is only covered by the invalidation of `['users','roles']`, not asserted in a test.

## AC coverage (frontend)

- AC-11: covered (Sidebar roles item, super_admin only; RequireRole roles=['super_admin'])
- AC-12: covered (table, badges, View for system rows, no Delete, Create button)
- AC-13: covered (dialog, name disabled on edit, matrix from catalogue, non-assignable disabled, inline errors, invalidation, toast)
- AC-14: covered (confirm dialog, ROLE_IN_USE translated with count, role remains)
- AC-15: covered by implementation (invalidates `['users','roles']`), not directly tested
- AC-16: covered (see focus area 1)
- AC-17: covered (locales identical, labels, trap), with the focus-return caveat above
- AC-18 (frontend part): RolesPage.test.tsx, Sidebar and RequireRole permission tests present (in permissionGuards.test.tsx)

Summary: No Critical or High issues; the permission fallback fails closed and built-in role behaviour is unchanged; address the Dialog focus-return capture and the in-use count source as follow-ups.

## Resolution (implementer)
Medium 1 fixed (opener captured during render + dialog.test.tsx). Medium 2 fixed (AdminHome shows load error when permissions unknown, test added). Medium 3 accepted: backend sends no details.count; regex fallback documented, follow-up backend note. Lows accepted.
