---
description: "Use when writing, reviewing, or debugging frontend React/TypeScript code. Covers React Query patterns, shadcn/ui usage, Tailwind layout, i18n rules, routing conventions, and TypeScript standards. Auto-loaded for all files under frontend/."
applyTo: "frontend/**"
---

# Frontend Conventions — React 18 + TypeScript

## Tech Stack

- **React 18** + **TypeScript** — strict mode
- **Vite** — build tool
- **Tailwind CSS** — utility-first styling
- **shadcn/ui** — component primitives (built on Radix UI)
- **TanStack Query (React Query v5)** — all server state
- **React Router v6** — client-side routing
- **react-i18next** — internationalization

## API Calls

All API calls go through `frontend/src/api/{domain}.ts`:

```typescript
// WRONG — raw fetch in a component
useEffect(() => {
  fetch('/api/v1/users').then(...)
}, [])

// RIGHT — React Query hook wrapping an API function
// 1. Define in src/api/users.ts:
export const usersApi = {
  list: async (params: ListParams): Promise<PaginatedUsers> => {
    const res = await fetch(`/api/v1/users?${toQueryString(params)}`)
    if (!res.ok) throw await res.json()
    return res.json()
  }
}

// 2. Use in a component:
const { data, isLoading, error } = useQuery({
  queryKey: ['users', params],
  queryFn: () => usersApi.list(params),
})
```

- React Query for ALL server state — no manual `useEffect` fetch loops.
- Mutations via `useMutation`; always call `queryClient.invalidateQueries(...)` after success.
- Never put backend URL, port, or API key in frontend source.
- All API requests go to `/api/*` — Vite proxies to `http://localhost:8080` in development.

## TypeScript

- Strict mode is on. No `any` types on API response shapes.
- Define response types alongside the API client function:

```typescript
export interface User {
  id: string        // UUID
  email: string
  fullName: string
  role: UserRole
  status: 'active' | 'inactive'
  createdAt: string  // UTC ISO 8601
}
```

## Components

- One component per file.
- Reusable primitives: use **shadcn/ui** (`Button`, `Input`, `Dialog`, `Table`, `Select`, `Badge`, etc.).
- Layout and spacing: **Tailwind** utility classes.
- Loading, error, and empty states must be explicitly handled:

```tsx
if (isLoading) return <Skeleton />
if (error) return <ErrorAlert error={error} />
if (!data?.length) return <EmptyState />
return <DataTable data={data} />
```

## Routing

```
/                      → redirect to /login or /portal based on auth
/login                 → LoginPage (public)
/change-password       → ChangePasswordPage (force_password_change gate)
/portal                → employee portal shell
/portal/exams          → employee exam list
/portal/exams/:id      → exam detail / session
/admin                 → admin shell (role guard: super_admin, department_admin, examiner)
/admin/users           → user management
/admin/departments     → department management
/admin/questions       → question bank
/admin/exams           → exam configuration
/admin/reports         → analytics
/admin/settings        → branding settings
/verify/:code          → public certificate verification
```

Route guards:
- `<RequireAuth>` — redirects to `/login` if unauthenticated.
- `<RequireRole roles={['super_admin', 'department_admin']}>` — redirects to `/portal` if wrong role.

## Internationalization (i18n)

- **Zero hardcoded user-visible strings** in components.
- All text via `useTranslation()`:

```tsx
// WRONG
<h1>User List</h1>

// RIGHT (add key to all three locale files first)
const { t } = useTranslation()
<h1>{t('users.list_title')}</h1>
```

- Locale files: `src/locales/en.json`, `kk.json`, `ru.json`.
- When adding a new key: add it to **all three files simultaneously**.
- Key structure: `{domain}.{key}` e.g. `auth.login_title`, `exam.time_remaining`.

## Forms

- Controlled inputs: `value` + `onChange` on every input.
- Form validation with **zod** + **react-hook-form**.
- Error messages are i18n strings, not hardcoded.

## Dev Commands

```bash
cd frontend && npm run dev       # dev server, port 5173
cd frontend && npm run build     # production build
cd frontend && npm test          # run tests
cd frontend && npm run lint      # ESLint check
```
