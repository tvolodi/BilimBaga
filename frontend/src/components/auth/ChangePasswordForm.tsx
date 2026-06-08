import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { ApiError } from '@/api/auth'

interface ChangePasswordPayload {
  current_password: string
  new_password: string
}

interface ChangePasswordFormProps {
  onSubmit: (values: ChangePasswordPayload) => void
  isPending: boolean
  error: ApiError | null
}

export function ChangePasswordForm({ onSubmit, isPending, error }: ChangePasswordFormProps) {
  const { t } = useTranslation()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [mismatch, setMismatch] = useState(false)
  const [showCurrent, setShowCurrent] = useState(false)
  const [showNew, setShowNew] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      setMismatch(true)
      return
    }
    setMismatch(false)
    onSubmit({ current_password: currentPassword, new_password: newPassword })
  }

  function renderError() {
    if (mismatch) {
      return (
        <p id="confirm-password-error" role="alert" className="text-sm text-destructive">
          {t('auth.changePassword.errors.mismatch')}
        </p>
      )
    }
    if (!error) return null
    let message: string
    if (error.code === 'INVALID_CREDENTIALS') {
      message = t('auth.changePassword.errors.INVALID_CREDENTIALS')
    } else if (error.code === 'VALIDATION_ERROR') {
      message = t('auth.changePassword.errors.VALIDATION_ERROR')
    } else {
      message = t('auth.login.errors.network')
    }
    return (
      <p role="alert" className="text-sm text-destructive">
        {message}
      </p>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 mt-6">
      <div className="space-y-1">
        <Label htmlFor="current-password">{t('auth.changePassword.currentPasswordLabel')}</Label>
        <div className="relative">
          <Input
            id="current-password"
            type={showCurrent ? 'text' : 'password'}
            autoComplete="off"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            className="pr-10"
            required
          />
          <button
            type="button"
            onClick={() => setShowCurrent((v) => !v)}
            aria-label={t(showCurrent ? 'auth.changePassword.hidePassword' : 'auth.changePassword.showPassword')}
            className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
          >
            {showCurrent ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
      </div>
      <div className="space-y-1">
        <Label htmlFor="new-password">{t('auth.changePassword.newPasswordLabel')}</Label>
        <div className="relative">
          <Input
            id="new-password"
            type={showNew ? 'text' : 'password'}
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            className="pr-10"
            required
          />
          <button
            type="button"
            onClick={() => setShowNew((v) => !v)}
            aria-label={t(showNew ? 'auth.changePassword.hidePassword' : 'auth.changePassword.showPassword')}
            className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
          >
            {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
      </div>
      <div className="space-y-1">
        <Label htmlFor="confirm-password">{t('auth.changePassword.confirmPasswordLabel')}</Label>
        <div className="relative">
          <Input
            id="confirm-password"
            type={showConfirm ? 'text' : 'password'}
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className="pr-10"
            aria-describedby={mismatch ? 'confirm-password-error' : undefined}
            aria-invalid={mismatch || undefined}
            required
          />
          <button
            type="button"
            onClick={() => setShowConfirm((v) => !v)}
            aria-label={t(showConfirm ? 'auth.changePassword.hidePassword' : 'auth.changePassword.showPassword')}
            className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
          >
            {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
      </div>
      {renderError()}
      <Button type="submit" className="w-full" disabled={isPending}>
        {isPending ? '…' : t('auth.changePassword.submitButton')}
      </Button>
    </form>
  )
}
