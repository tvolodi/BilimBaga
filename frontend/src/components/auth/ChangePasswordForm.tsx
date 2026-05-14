import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
        <p role="alert" className="text-sm text-destructive">
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
        <Input
          id="current-password"
          type="password"
          autoComplete="current-password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          required
        />
      </div>
      <div className="space-y-1">
        <Label htmlFor="new-password">{t('auth.changePassword.newPasswordLabel')}</Label>
        <Input
          id="new-password"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          required
        />
      </div>
      <div className="space-y-1">
        <Label htmlFor="confirm-password">{t('auth.changePassword.confirmPasswordLabel')}</Label>
        <Input
          id="confirm-password"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
        />
      </div>
      {renderError()}
      <Button type="submit" className="w-full" disabled={isPending}>
        {isPending ? '…' : t('auth.changePassword.submitButton')}
      </Button>
    </form>
  )
}
