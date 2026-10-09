import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useTenantConfig } from '@/api/useTenantConfig'
import { useLogin, type ApiError } from '@/api/auth'
import { TenantLogo } from '@/components/TenantLogo'
import { LoginForm } from '@/components/auth/LoginForm'
import { LanguageSelector } from '@/components/auth/LanguageSelector'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { Card } from '@/components/ui/card'

interface LoginPayload {
  email: string
  password: string
}

export function LoginPage() {
  const { data: config, isLoading: configLoading } = useTenantConfig()
  const login = useLogin()
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()
  // FR-BB115 AC-7: success notice after a completed password reset.
  const resetNotice = (location.state as { passwordReset?: boolean } | null)?.passwordReset === true

  if (configLoading) return <FullPageSpinner />

  async function handleSubmit(values: LoginPayload) {
    try {
      const result = await login.mutateAsync(values)
      if (result.user.force_password_change) {
        navigate('/change-password')
      } else if (result.user.role === 'employee') {
        navigate('/portal')
      } else {
        navigate('/admin')
      }
    } catch {
      // error displayed inline via login.error
    }
  }

  return (
    <main id="main-content" tabIndex={-1} className="flex min-h-screen items-center justify-center bg-muted outline-none">
      <Card className="w-full max-w-md p-8">
        <TenantLogo appName={config?.app_name} />
        <h1 className="mt-4 text-xl font-semibold text-center">
          {t('auth.login.title', { appName: config?.app_name ?? 'BilimBaga' })}
        </h1>
        {resetNotice && (
          <p
            role="status"
            className="mt-4 rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-800"
          >
            {t('auth.recovery.resetSuccess')}
          </p>
        )}
        <LoginForm
          onSubmit={handleSubmit}
          isPending={login.isPending}
          error={login.error as ApiError | null}
        />
        <LanguageSelector availableLocales={config?.available_locales ?? ['kk', 'ru', 'en']} />
      </Card>
    </main>
  )
}
