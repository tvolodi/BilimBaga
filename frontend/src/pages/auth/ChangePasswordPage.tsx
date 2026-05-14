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

export function ChangePasswordPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const changePassword = useChangePassword()

  async function handleSubmit(values: ChangePasswordPayload) {
    try {
      await changePassword.mutateAsync(values)
      const user = qc.getQueryData<CurrentUser>(['auth', 'currentUser'])
      if (user) {
        qc.setQueryData(['auth', 'currentUser'], { ...user, force_password_change: false })
      }
      if (user?.role === 'employee') {
        navigate('/portal')
      } else {
        navigate('/admin')
      }
    } catch {
      // error displayed inline via changePassword.error
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted">
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
    </div>
  )
}
