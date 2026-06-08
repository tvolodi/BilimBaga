import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useChangePassword, type ApiError, type CurrentUser } from '@/api/auth'
import { ChangePasswordForm } from '@/components/auth/ChangePasswordForm'
import { Card } from '@/components/ui/card'

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

  async function handleSubmit(values: ChangePasswordPayload) {
    try {
      await changePassword.mutateAsync(values)
      // Update currentUser cache to clear force_password_change if it's still present.
      const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])
      if (user) {
        qc.setQueryData(['auth', 'currentUser'], { ...user, force_password_change: false })
      }
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
    <main id="main-content" tabIndex={-1} className="flex min-h-screen items-center justify-center bg-muted outline-none">
      <Card className="w-full max-w-md p-8">
        <h1 className="text-xl font-semibold text-center">
          {t('auth.changePassword.title')}
        </h1>
        <ChangePasswordForm
          onSubmit={handleSubmit}
          isPending={changePassword.isPending}
          error={changePassword.error as ApiError | null}
        />
      </Card>
    </main>
  )
}
