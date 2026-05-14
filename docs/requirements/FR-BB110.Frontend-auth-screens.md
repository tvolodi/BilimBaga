# FR-BB110 — Frontend: Auth Screens

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB110 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB14, FR-BB13 |

## Description
Delivers the login screen and forced-password-change screen — the two entry points that every user passes through before reaching any protected area of the application. Tenant branding (logo, colours) is fetched before render and applied via CSS custom properties so the screens feel native to the organisation. All strings are externalised to i18n locale files and the language selector on the login page allows switching before authentication.

## Acceptance Criteria
- [ ] AC-1: The login page displays the tenant logo (from `GET /api/v1/tenant/logo`) at the top; if no logo is configured (204 response), a text fallback using `app_name` is shown.
- [ ] AC-2: The login page contains an email field, a password field (masked), a language selector, and a "Sign in" button; all labels and placeholder text come from the active i18n locale file.
- [ ] AC-3: On successful login, the access token is stored in React Query's client state (not `localStorage`); the user is redirected to `/admin` (admin roles) or `/portal` (employee role).
- [ ] AC-4: When the server returns `force_password_change: true`, the user is immediately redirected to `/auth/change-password` before any other page loads.
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
    tenant.ts          # React Query query: useTenantConfig
  components/
    auth/
      LoginForm.tsx
      ChangePasswordForm.tsx
      LanguageSelector.tsx
      TenantLogo.tsx
  locales/
    kk.json            # Kazakh strings
    ru.json            # Russian strings
    en.json            # English strings
```

#### `useTenantConfig` hook (`src/api/tenant.ts`)

```typescript
import { useQuery } from '@tanstack/react-query';

interface TenantConfig {
  app_name: string;
  primary_color: string;
  accent_color: string;
  default_locale: string;
  available_locales: string[];
}

export function useTenantConfig() {
  return useQuery<TenantConfig>({
    queryKey: ['tenant', 'config'],
    queryFn: async () => {
      const res = await fetch('/api/v1/tenant/config');
      if (!res.ok) throw new Error('Failed to load tenant config');
      const json = await res.json();
      return json.data;
    },
    staleTime: Infinity,  // config is stable for the session
  });
}
```

#### `useLogin` mutation (`src/api/auth.ts`)

```typescript
import { useMutation, useQueryClient } from '@tanstack/react-query';

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
export function LoginPage() {
  const { data: config, isLoading: configLoading } = useTenantConfig();
  const login = useLogin();
  const navigate = useNavigate();
  const { t } = useTranslation();

  // Apply brand colours before render
  useEffect(() => {
    if (!config) return;
    document.documentElement.style.setProperty('--color-primary', config.primary_color);
    document.documentElement.style.setProperty('--color-accent', config.accent_color);
  }, [config]);

  if (configLoading) return <FullPageSpinner />;

  const handleSubmit = async (values: LoginPayload) => {
    try {
      const result = await login.mutateAsync(values);
      if (result.user.force_password_change) {
        navigate('/auth/change-password');
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
  <Route path="/auth/login"           element={<LoginPage />} />
  <Route path="/auth/change-password" element={<RequireAuth><ChangePasswordPage /></RequireAuth>} />
  <Route path="/admin/*"              element={<RequireRole roles={['super_admin','department_admin','examiner']}><AdminShell /></RequireRole>} />
  <Route path="/portal/*"             element={<RequireRole roles={['employee']}><EmployeePortal /></RequireRole>} />
  <Route path="/"                     element={<Navigate to="/auth/login" replace />} />
</Routes>
```

## Notes
- The access token must never be written to `localStorage` or `sessionStorage` to mitigate XSS token theft. Store it only in React Query's in-memory client state; use the httpOnly refresh cookie for persistence across tab refreshes.
- On page refresh, the app should call `POST /api/v1/auth/refresh` silently to restore the access token from the cookie before rendering protected routes.
- `LanguageSelector` saves selection to `localStorage['i18n-lang']` and calls `i18n.changeLanguage()` immediately; `i18next` picks up the persisted value on next load via the `languageDetector` plugin configured with `localStorage` order.
- The `locked_until` timestamp must be formatted in the user's browser time zone using `Intl.DateTimeFormat`, not displayed as raw UTC.
- Tenant config fetch failure (network error) should not block the login form from rendering — show a degraded UI with default colours.
