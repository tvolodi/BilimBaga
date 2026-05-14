# FR-BB110 — Frontend: Auth Screens

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB110 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Ready |
| Depends On | FR-BB14, FR-BB13 |

## Description
Delivers the login screen and forced-password-change screen — the two entry points that every user passes through before reaching any protected area of the application. Tenant branding (logo, colours) is fetched before render and applied via CSS custom properties so the screens feel native to the organisation. All strings are externalised to i18n locale files and the language selector on the login page allows switching before authentication.

## Scope

| Layer    | Artifact |
|----------|----------|
| Frontend | `src/pages/auth/LoginPage.tsx`, `src/pages/auth/ChangePasswordPage.tsx`, `src/api/auth.ts`, `src/components/auth/LoginForm.tsx`, `src/components/auth/ChangePasswordForm.tsx`, `src/components/auth/LanguageSelector.tsx`, `src/components/TenantLogo.tsx` (extend), `src/locales/kk.json`, `src/locales/ru.json`, `src/locales/en.json` |
| Backend  | No changes — using existing endpoints from FR-BB14 |
| Database | No changes |
| i18n     | `auth.*` key namespace added to `kk.json`, `ru.json`, `en.json` |

## Out of Scope

- Registration / self-sign-up flow
- Multi-factor authentication (MFA)
- Social / SSO login (OAuth2, SAML)
- Account unlock self-service
- Password-strength policy enforcement UI
- Password reset via email link

## Acceptance Criteria
- [ ] AC-1: The login page displays the tenant logo (from `GET /api/v1/tenant/logo`) at the top; if no logo is configured (204 response), a text fallback using `app_name` is shown.
- [ ] AC-2: The login page contains an email field, a password field (masked), a language selector, and a "Sign in" button; all labels and placeholder text come from the active i18n locale file.
- [ ] AC-3: On successful login, the access token is stored in React Query's client state (not `localStorage`); the user is redirected to `/admin` (admin roles) or `/portal` (employee role).
- [ ] AC-4: When the server returns `force_password_change: true`, the user is immediately redirected to `/change-password` before any other page loads.
- [ ] AC-5: The forced-password-change screen requires the user to enter their current password and a new password (with confirmation); on success the user is redirected to the appropriate landing page.
- [ ] AC-6: The error state `INVALID_CREDENTIALS` displays an inline error message without clearing the email field.
- [ ] AC-7: The error state `ACCOUNT_LOCKED` displays the lock message and the `locked_until` timestamp in the user's locale and time zone.
- [ ] AC-8: Network errors (fetch failure, 5xx) display a generic "Service unavailable, try again later" message and do not expose internal error details to the user.
- [ ] AC-9: Tenant config is fetched via `useTenantConfig()` before the login form renders; primary and accent colours are applied as CSS custom properties on `:root`.
- [ ] AC-10: The language selector persists the user's choice to `localStorage` and applies it on subsequent visits before authentication.

## Technical Specification

### Frontend Components

#### File layout

```
src/
  pages/
    auth/
      LoginPage.tsx
      ChangePasswordPage.tsx
  api/
    auth.ts            # React Query mutations: useLogin, useChangePassword, useLogout
  components/
    auth/
      LoginForm.tsx
      ChangePasswordForm.tsx
      LanguageSelector.tsx
  locales/
    kk.json            # Kazakh strings
    ru.json            # Russian strings
    en.json            # English strings
```

> **Note — existing hook**: `useTenantConfig()` already exists at `src/api/useTenantConfig.ts` (queryKey `['tenant','config']`, `staleTime: Infinity`). Import from there — do **not** create `src/api/tenant.ts`.

> **Note — existing component**: `src/components/TenantLogo.tsx` already exists. Extend it to accept an optional `appName?: string` prop; when the logo `<img>` fails to load, display `appName` as the text fallback instead of the generic i18n placeholder. Do **not** create `src/components/auth/TenantLogo.tsx`; import `TenantLogo` from `../../components/TenantLogo` in `LoginPage.tsx`.

#### `useLogin` mutation (`src/api/auth.ts`)

```typescript
import { useMutation, useQueryClient } from '@tanstack/react-query';

interface ApiError {
  code: string;
  message: string;
  details?: Record<string, string>;
}

// Note: If a shared `ApiError` type already exists in `src/api/types.ts`, import from there instead of redefining.

interface LoginPayload {
  email: string;
  password: string;
}

interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: {
    id: string;
    full_name: string;
    email: string;
    role: string;
    force_password_change: boolean;
  };
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation<LoginResponse, ApiError, LoginPayload>({
    mutationFn: async (payload) => {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        credentials: 'include',   // send/receive cookies
      });
      const json = await res.json();
      if (!res.ok) throw json.error;
      return json.data;
    },
    onSuccess: (data) => {
      qc.setQueryData(['auth', 'currentUser'], data.user);
      qc.setQueryData(['auth', 'accessToken'], data.access_token);
    },
  });
}
```

#### `LoginPage.tsx` — key behaviour

```typescript
// Key imports:
// import { useTenantConfig } from '../../api/useTenantConfig';
// import { TenantLogo } from '../../components/TenantLogo';

// CSS brand colours are already applied globally by TenantProvider in App.tsx — no local useEffect needed.
export function LoginPage() {
  const { data: config, isLoading: configLoading } = useTenantConfig();
  const login = useLogin();
  const navigate = useNavigate();
  const { t } = useTranslation();

  if (configLoading) return <FullPageSpinner />;

  const handleSubmit = async (values: LoginPayload) => {
    try {
      const result = await login.mutateAsync(values);
      if (result.user.force_password_change) {
        navigate('/change-password');
      } else if (result.user.role === 'employee') {
        navigate('/portal');
      } else {
        navigate('/admin');
      }
    } catch (err: any) {
      // error displayed inline by LoginForm via login.error
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted">
      <Card className="w-full max-w-md p-8">
        <TenantLogo appName={config?.app_name} />
        <LoginForm onSubmit={handleSubmit} isPending={login.isPending} error={login.error} />
        <LanguageSelector availableLocales={config?.available_locales ?? ['kk', 'ru', 'en']} />
      </Card>
    </div>
  );
}
```

#### i18n keys required (example — `en.json` excerpt)

```json
{
  "auth": {
    "login": {
      "title": "Sign in to {{appName}}",
      "emailLabel": "Email",
      "emailPlaceholder": "you@example.com",
      "passwordLabel": "Password",
      "submitButton": "Sign in",
      "errors": {
        "INVALID_CREDENTIALS": "Incorrect email or password.",
        "ACCOUNT_LOCKED": "Account locked until {{until}}. Contact your administrator.",
        "network": "Service unavailable. Please try again later."
      }
    },
    "changePassword": {
      "title": "Set your new password",
      "currentPasswordLabel": "Current password",
      "newPasswordLabel": "New password",
      "confirmPasswordLabel": "Confirm new password",
      "submitButton": "Change password",
      "errors": {
        "INVALID_CREDENTIALS": "Current password is incorrect.",
        "VALIDATION_ERROR": "New password does not meet requirements.",
        "mismatch": "Passwords do not match."
      }
    }
  }
}
```

#### Route configuration

```typescript
// In App.tsx / router setup
<Routes>
  <Route path="/login"           element={<LoginPage />} />
  <Route path="/change-password" element={<RequireAuth><ChangePasswordPage /></RequireAuth>} />
  <Route path="/admin/*"         element={<RequireRole roles={['super_admin','department_admin','examiner']}><AdminShell /></RequireRole>} />
  <Route path="/portal/*"        element={<RequireRole roles={['employee']}><EmployeePortal /></RequireRole>} />
  <Route path="/"                element={<Navigate to="/login" replace />} />
</Routes>
```

## Test Strategy

| Test type | Scope | Tool |
|-----------|-------|------|
| Unit | `useLogin` mutation (success, INVALID_CREDENTIALS, ACCOUNT_LOCKED, network error) | Vitest + MSW |
| Unit | `useTenantConfig` query (success, 204 logo fallback, network error degraded UI) | Vitest + MSW |
| Component | `LoginForm` renders error variants inline, does not clear email on INVALID_CREDENTIALS | React Testing Library |
| Component | `LanguageSelector` persists choice to localStorage; applies i18n change immediately | React Testing Library |
| Component | `ChangePasswordForm` shows mismatch error client-side | React Testing Library |
| e2e | Full login → redirect flow (admin role → /admin; employee role → /portal; force_password_change → /change-password) | Playwright |

## Notes
- The access token must never be written to `localStorage` or `sessionStorage` to mitigate XSS token theft. Store it only in React Query's in-memory client state; use the httpOnly refresh cookie for persistence across tab refreshes.
- On page refresh, the app should call `POST /api/v1/auth/refresh` silently to restore the access token from the cookie before rendering protected routes.
- `LanguageSelector` saves selection to `localStorage['i18n-lang']` and calls `i18n.changeLanguage()` immediately; `i18next` picks up the persisted value on next load via the `languageDetector` plugin configured with `localStorage` order.
- The `locked_until` timestamp must be formatted in the user's browser time zone using `Intl.DateTimeFormat`, not displayed as raw UTC.
- Tenant config fetch failure (network error) should not block the login form from rendering — show a degraded UI with default colours.
