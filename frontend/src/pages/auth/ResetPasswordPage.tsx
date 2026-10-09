import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff } from 'lucide-react'
import { useCompletePasswordReset } from '@/api/recovery'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

/** FR-BB115 AC-7: public reset page. Reads `token` from the query string. */
export function ResetPasswordPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const reset = useCompletePasswordReset()

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [mismatch, setMismatch] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (password !== confirm) {
      setMismatch(true)
      return
    }
    setMismatch(false)
    try {
      await reset.mutateAsync({ token, new_password: password })
      navigate('/login', { replace: true, state: { passwordReset: true } })
    } catch {
      // error rendered below from reset.error
    }
  }

  const code = reset.error?.code
  const tokenProblem = token === '' || code === 'INVALID_TOKEN'

  function errorMessage(): string | null {
    if (mismatch) return t('auth.recovery.errors.mismatch')
    if (!reset.error || code === 'INVALID_TOKEN') return null
    if (code === 'VALIDATION_ERROR') return t('auth.recovery.errors.VALIDATION_ERROR')
    if (code === 'RATE_LIMITED') return t('auth.recovery.errors.RATE_LIMITED')
    return t('auth.recovery.errors.network')
  }
  const error = errorMessage()

  return (
    <main
      id="main-content"
      tabIndex={-1}
      className="flex min-h-screen items-center justify-center bg-muted px-4 outline-none"
    >
      <Card className="w-full max-w-md p-6 sm:p-8">
        <h1 className="text-xl font-semibold text-center">{t('auth.recovery.reset.title')}</h1>
        {tokenProblem ? (
          <div className="mt-6 space-y-4">
            <p role="alert" className="text-sm text-destructive">
              {t('auth.recovery.errors.INVALID_TOKEN')}
            </p>
            <Link
              to="/forgot-password"
              className="block text-sm text-primary underline-offset-4 hover:underline"
            >
              {t('auth.recovery.reset.requestNewLink')}
            </Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="mt-6 space-y-4">
            <div className="space-y-1">
              <Label htmlFor="reset-new-password">{t('auth.recovery.reset.newPasswordLabel')}</Label>
              <div className="relative">
                <Input
                  id="reset-new-password"
                  type={showNew ? 'text' : 'password'}
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  aria-describedby="reset-password-hint"
                  className="pr-10"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowNew((v) => !v)}
                  aria-label={t(
                    showNew ? 'auth.recovery.reset.hidePassword' : 'auth.recovery.reset.showPassword',
                  )}
                  className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
                >
                  {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              <p id="reset-password-hint" className="text-xs text-muted-foreground">
                {t('auth.recovery.reset.passwordHint')}
              </p>
            </div>
            <div className="space-y-1">
              <Label htmlFor="reset-confirm-password">
                {t('auth.recovery.reset.confirmPasswordLabel')}
              </Label>
              <div className="relative">
                <Input
                  id="reset-confirm-password"
                  type={showConfirm ? 'text' : 'password'}
                  autoComplete="new-password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  aria-invalid={mismatch || undefined}
                  className="pr-10"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowConfirm((v) => !v)}
                  aria-label={t(
                    showConfirm
                      ? 'auth.recovery.reset.hidePassword'
                      : 'auth.recovery.reset.showPassword',
                  )}
                  className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
                >
                  {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>
            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
            <Button type="submit" className="w-full" disabled={reset.isPending}>
              {reset.isPending ? '…' : t('auth.recovery.reset.submitButton')}
            </Button>
          </form>
        )}
      </Card>
    </main>
  )
}
