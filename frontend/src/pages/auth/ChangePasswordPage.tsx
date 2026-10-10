import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useChangePassword, type ApiError, type CurrentUser } from '@/api/auth'
import { ChangePasswordForm } from '@/components/auth/ChangePasswordForm'
import { Card } from '@/components/ui/card'
import { clearPasswordChangeRequired, PASSWORD_CHANGE_FLAG_KEY } from '@/lib/passwordChangeRequired'
import { AuthShell } from '@/components/auth/AuthShell'

interface ChangePasswordPayload {
  current_password: string
  new_password: string
}

/** Decode the `role` claim from a JWT without signature verification. */
function jwtRole(token: string): string | undefined {
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    return payload?.role as string | undefined
  } catch {
    return undefined
  }
}

export function ChangePasswordPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const changePassword = useChangePassword()
  // ISS-160: show why the user landed here when the change is mandatory.
  const required =
    qc.getQueryData<boolean>(PASSWORD_CHANGE_FLAG_KEY) === true ||
    qc.getQueryData<CurrentUser>(['auth', 'currentUser'])?.force_password_change === true

  async function handleSubmit(values: ChangePasswordPayload) {
    try {
      await changePassword.mutateAsync(values)
      // Update currentUser cache to clear force_password_change if it's still present.
      const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])
      if (user) {
        qc.setQueryData(['auth', 'currentUser'], { ...user, force_password_change: false })
      }
      clearPasswordChangeRequired(qc)
      // ISS-027: Decode role from JWT (same pattern as RequireRole after ISS-019).
      // Do NOT rely on currentUser.role — it may have been garbage-collected by TanStack Query.
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      const role = token ? jwtRole(token) : undefined
      if (role === 'employee') {
        navigate('/portal')
      } else {
        navigate('/admin')
      }
    } catch {
      // error displayed inline via changePassword.error
    }
  }

  return (
    <AuthShell>
      <main id="main-content" tabIndex={-1} className="flex min-h-screen items-center justify-center bg-muted outline-none">
        <Card className="w-full max-w-md p-8">
          <h1 className="text-xl font-semibold text-center">
            {t('auth.changePassword.title')}
          </h1>
          {required && (
            <p role="status" className="mt-4 text-center text-sm text-muted-foreground">
              {t('auth.changePassword.required')}
            </p>
          )}
          <ChangePasswordForm
            onSubmit={handleSubmit}
            isPending={changePassword.isPending}
            error={changePassword.error as ApiError | null}
          />
        </Card>
      </main>
    </AuthShell>
  )
}
