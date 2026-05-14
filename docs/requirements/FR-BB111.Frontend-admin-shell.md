# FR-BB111 — Frontend: Admin Shell

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB111 |
| Phase | 1 — Foundation |
| Priority | 3 |
| Status | Draft |
| Depends On | FR-BB110, FR-BB17, FR-BB18 |

## Description
Provides the persistent chrome — sidebar navigation, top bar, and breadcrumbs — that wraps every admin-facing page. Route guards enforce authentication and role boundaries so that unauthenticated users are redirected to login and employees cannot reach admin pages. The User List page and User Create/Edit drawer are delivered as the first functional admin screens, exercising the shell's data-fetching and navigation patterns.

## Acceptance Criteria
- [ ] AC-1: The admin layout renders a collapsible sidebar with navigation links: Dashboard, Users, Departments, Questions, Exams, Reports, Settings; active link is visually highlighted.
- [ ] AC-2: The top bar displays the authenticated user's `full_name`, a role badge (coloured by role), and a logout button that calls `POST /api/v1/auth/logout` and redirects to `/auth/login`.
- [ ] AC-3: A `<Breadcrumb />` component reflects the current route hierarchy (e.g. "Admin › Users › Edit").
- [ ] AC-4: Any navigation to `/admin/*` without a valid access token immediately redirects to `/auth/login`; the originally requested URL is stored so the user is redirected back after login.
- [ ] AC-5: Any navigation to `/admin/*` by a user with `role = 'employee'` immediately redirects to `/portal`.
- [ ] AC-6: The Users list page (`/admin/users`) renders a table with columns: Full Name, Email, Department, Role, Status, Actions; supports client-side column sorting.
- [ ] AC-7: The Users list page includes filter controls for Department (select), Role (select), and Status (toggle); applying filters calls the API with the corresponding query parameters.
- [ ] AC-8: A "New User" button opens the User Create Drawer; on successful save the users list query is invalidated so the new user appears without a full page reload.
- [ ] AC-9: The User Edit Drawer is opened from the "Edit" action in the table row; the form pre-fills all editable fields (`full_name`, `department_id`, `role_id`).
- [ ] AC-10: A "Bulk Import" button opens the Import Modal (two-step CSV flow); on commit the users list query is invalidated.
- [ ] AC-11: The sidebar can be collapsed to icon-only mode; the state persists in `localStorage['sidebar-collapsed']`.
- [ ] AC-12: All user-visible strings in the admin shell (navigation labels, column headers, button labels, status badges) come from i18n locale files; no hardcoded English text in component files.

## Technical Specification

### Frontend Components

#### File layout

```
src/
  layouts/
    AdminLayout.tsx         # sidebar + topbar + breadcrumb shell
  components/
    admin/
      Sidebar.tsx           # collapsible nav with icon set
      TopBar.tsx            # user info, role badge, logout
      Breadcrumb.tsx        # route-aware breadcrumb
      RoleBadge.tsx         # coloured badge per role
      StatusBadge.tsx       # active / inactive badge
  pages/
    admin/
      users/
        UsersListPage.tsx
        UserCreateDrawer.tsx
        UserEditDrawer.tsx
        ImportModal.tsx
      departments/
        DepartmentsPage.tsx   (shell placeholder — FR-BB17 full impl)
  api/
    users.ts                # useUsers, useUser, useCreateUser, useUpdateUser,
                            #   useDeactivateUser, useResetPassword, useImportUsers
    departments.ts          # useDepartments
```

#### `AdminLayout.tsx` — structure

```typescript
export function AdminLayout() {
  const { data: user } = useCurrentUser();   // GET /api/v1/users/me
  const navigate = useNavigate();
  const [collapsed, setCollapsed] = useLocalStorage('sidebar-collapsed', false);

  if (!user) {
    return <Navigate to="/auth/login" state={{ from: location }} replace />;
  }
  if (user.role === 'employee') {
    return <Navigate to="/portal" replace />;
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar user={user} />
        <main className="flex-1 overflow-y-auto p-6">
          <Breadcrumb />
          <Outlet />
        </main>
      </div>
    </div>
  );
}
```

#### Sidebar nav items

```typescript
const NAV_ITEMS = [
  { key: 'dashboard',   icon: LayoutDashboard, path: '/admin',             labelKey: 'nav.dashboard'   },
  { key: 'users',       icon: Users,           path: '/admin/users',        labelKey: 'nav.users'       },
  { key: 'departments', icon: Building2,        path: '/admin/departments',  labelKey: 'nav.departments' },
  { key: 'questions',   icon: FileQuestion,     path: '/admin/questions',    labelKey: 'nav.questions'   },
  { key: 'exams',       icon: ClipboardList,    path: '/admin/exams',        labelKey: 'nav.exams'       },
  { key: 'reports',     icon: BarChart2,        path: '/admin/reports',      labelKey: 'nav.reports'     },
  { key: 'settings',    icon: Settings,         path: '/admin/settings',     labelKey: 'nav.settings'    },
];
```

#### `UsersListPage.tsx` — key data flow

```typescript
export function UsersListPage() {
  const { t } = useTranslation();
  const [filters, setFilters] = useState<UserFilters>({});
  const [page, setPage] = useState(1);
  const { data, isLoading } = useUsers({ ...filters, page, page_size: 20 });

  return (
    <div>
      <PageHeader title={t('users.listTitle')}>
        <Button onClick={openImportModal}>{t('users.import')}</Button>
        <Button onClick={openCreateDrawer}>{t('users.create')}</Button>
      </PageHeader>
      <UserFiltersBar filters={filters} onChange={setFilters} />
      <DataTable columns={columns} data={data?.items ?? []} isLoading={isLoading} />
      <Pagination page={page} total={data?.pagination.total ?? 0} pageSize={20} onChange={setPage} />
    </div>
  );
}
```

#### `useUsers` hook (`src/api/users.ts`)

```typescript
export function useUsers(params: UserListParams) {
  return useQuery({
    queryKey: ['users', params],
    queryFn: async () => {
      const qs = new URLSearchParams(
        Object.entries(params)
          .filter(([, v]) => v !== undefined && v !== '')
          .map(([k, v]) => [k, String(v)])
      );
      const res = await apiFetch(`/api/v1/users?${qs}`);
      return res.data as UserListResponse;
    },
  });
}
```

#### i18n keys required (excerpt `en.json`)

```json
{
  "nav": {
    "dashboard":   "Dashboard",
    "users":       "Users",
    "departments": "Departments",
    "questions":   "Questions",
    "exams":       "Exams",
    "reports":     "Reports",
    "settings":    "Settings"
  },
  "users": {
    "listTitle":   "Users",
    "create":      "New User",
    "import":      "Bulk Import",
    "columns": {
      "fullName":   "Full Name",
      "email":      "Email",
      "department": "Department",
      "role":       "Role",
      "status":     "Status",
      "actions":    "Actions"
    },
    "roles": {
      "super_admin":       "Super Admin",
      "department_admin":  "Dept. Admin",
      "examiner":          "Examiner",
      "employee":          "Employee"
    },
    "status": {
      "active":   "Active",
      "inactive": "Inactive"
    }
  }
}
```

## Notes
- `useCurrentUser()` calls `GET /api/v1/users/me` and is cached by React Query; it is the single source of truth for the current user's identity throughout the admin shell.
- The route guard pattern uses React Router's `<Navigate state={{ from: location }} />` so the post-login redirect works correctly.
- shadcn/ui `Sheet` component is used for both create and edit drawers to ensure consistent slide-over behaviour and focus management.
- Role badge colours: `super_admin` = purple, `department_admin` = blue, `examiner` = teal, `employee` = gray — defined as Tailwind `data-role` variant classes, not hardcoded `style` attributes.
- Navigation items for which the user lacks `read` permission (based on their role) should be hidden from the sidebar to reduce confusion; this is a UX convenience, not a security control (the backend enforces permissions).
