import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { ApiError } from '@/api/auth'

interface LoginPayload {
  email: string
  password: string
}

interface LoginFormProps {
  onSubmit: (values: LoginPayload) => void
  isPending: boolean
  error: ApiError | null
}

function formatLockedUntil(message: string): string {
  const match = message.match(/(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?)/)
  if (!match) return message
  try {
    const date = new Date(match[1])
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(date)
  } catch {
    return match[1]
  }
}

export function LoginForm({ onSubmit, isPending, error }: LoginFormProps) {
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onSubmit({ email, password })
  }

  function renderError() {
    if (!error) return null
    let message: string
    if (error.code === 'ACCOUNT_LOCKED') {
      const until = formatLockedUntil(error.message)
      message = t('auth.login.errors.ACCOUNT_LOCKED', { until })
    } else if (error.code === 'INVALID_CREDENTIALS') {
      message = t('auth.login.errors.INVALID_CREDENTIALS')
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
        <Label htmlFor="email">{t('auth.login.emailLabel')}</Label>
        <Input
          id="email"
          type="email"
          autoComplete="username"
          placeholder={t('auth.login.emailPlaceholder')}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
      </div>
      <div className="space-y-1">
        <Label htmlFor="password">{t('auth.login.passwordLabel')}</Label>
        <div className="relative">
          <Input
            id="password"
            type={showPassword ? 'text' : 'password'}
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="pr-10"
            required
          />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            aria-label={t(showPassword ? 'auth.login.hidePassword' : 'auth.login.showPassword')}
            className="absolute inset-y-0 right-0 flex items-center px-3 text-muted-foreground hover:text-foreground"
          >
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
      </div>
      {renderError()}
      <Button type="submit" className="w-full" disabled={isPending}>
        {isPending ? '…' : t('auth.login.submitButton')}
      </Button>
      <p className="text-center text-sm">
        <Link to="/forgot-password" className="text-primary underline-offset-4 hover:underline">
          {t('auth.recovery.forgotLink')}
        </Link>
      </p>
    </form>
  )
}
