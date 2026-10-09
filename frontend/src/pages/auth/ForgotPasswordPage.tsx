import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForgotPassword } from '@/api/recovery'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/** FR-BB115 AC-7: public "forgot password" page (outside RequireAuth). */
export function ForgotPasswordPage() {
  const { t } = useTranslation()
  const forgot = useForgotPassword()
  const [email, setEmail] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    forgot.mutate({ email: email.trim() })
  }

  // The confirmation is neutral: it is shown for any accepted request, so the page never
  // reveals whether the address belongs to an account. A throttled request is also neutral.
  const showConfirmation = forgot.isSuccess || forgot.error?.code === 'RATE_LIMITED'
  const showError = forgot.isError && !showConfirmation

  return (
    <main
      id="main-content"
      tabIndex={-1}
      className="flex min-h-screen items-center justify-center bg-muted px-4 outline-none"
    >
      <Card className="w-full max-w-md p-6 sm:p-8">
        <h1 className="text-xl font-semibold text-center">{t('auth.recovery.forgot.title')}</h1>
        {showConfirmation ? (
          <p role="status" className="mt-6 text-sm">
            {t('auth.recovery.forgot.confirmation')}
          </p>
        ) : (
          <form onSubmit={handleSubmit} className="mt-6 space-y-4">
            <p className="text-sm text-muted-foreground">{t('auth.recovery.forgot.intro')}</p>
            <div className="space-y-1">
              <Label htmlFor="forgot-email">{t('auth.recovery.forgot.emailLabel')}</Label>
              <Input
                id="forgot-email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            {showError && (
              <p role="alert" className="text-sm text-destructive">
                {t('auth.recovery.errors.network')}
              </p>
            )}
            <Button type="submit" className="w-full" disabled={forgot.isPending}>
              {forgot.isPending ? '…' : t('auth.recovery.forgot.submitButton')}
            </Button>
          </form>
        )}
        <p className="mt-6 text-center text-sm">
          <Link to="/login" className="text-primary underline-offset-4 hover:underline">
            {t('auth.recovery.forgot.backToLogin')}
          </Link>
        </p>
      </Card>
    </main>
  )
}
