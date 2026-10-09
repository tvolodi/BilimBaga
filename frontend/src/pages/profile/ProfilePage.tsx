import { useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useUpdateMyLocale } from '@/api/users'
import { RoleBadge } from '@/components/admin/RoleBadge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { buttonVariants } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { SUPPORTED_LOCALES, applyLocale, isSupportedLocale } from '@/lib/locale'
import { cn } from '@/lib/utils'

type Notice = { kind: 'success' | 'error'; text: string } | null

/**
 * FR-BB116: the self-service profile. Shows the caller's identity (read-only), lets the caller
 * choose the UI and email language, and links to change the password. Rendered inside the
 * admin or portal layout.
 */
export function ProfilePage() {
  const { t, i18n } = useTranslation()
  const me = useMe()
  const updateLocale = useUpdateMyLocale()
  const [notice, setNotice] = useState<Notice>(null)

  const currentLocale = isSupportedLocale(i18n.language) ? i18n.language : 'en'

  function handleLanguageChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const next = e.target.value
    if (!isSupportedLocale(next) || next === currentLocale) return
    const previous = currentLocale
    // AC-6: the UI switches immediately; the change is then persisted.
    applyLocale(i18n, next)
    setNotice(null)
    updateLocale.mutate(next, {
      onSuccess: () => setNotice({ kind: 'success', text: t('profile.language_saved') }),
      onError: () => {
        applyLocale(i18n, previous)
        setNotice({ kind: 'error', text: t('profile.language_save_failed') })
      },
    })
  }

  if (me.isPending) {
    return (
      <p role="status" className="p-6 text-sm text-muted-foreground">
        {t('common.loading')}
      </p>
    )
  }
  if (me.isError || !me.data) {
    return (
      <p role="alert" className="p-6 text-sm text-destructive">
        {t('profile.load_failed')}
      </p>
    )
  }

  const user = me.data
  const fieldLabelClass = 'text-sm text-muted-foreground'

  return (
    <div className="mx-auto w-full max-w-2xl space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">{t('profile.title')}</h1>

      <Card>
        <CardHeader>
          <CardTitle>{t('profile.identity_heading')}</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-2">
            <div>
              <dt className={fieldLabelClass}>{t('profile.full_name')}</dt>
              <dd className="mt-1 break-words font-medium">{user.full_name}</dd>
            </div>
            <div>
              <dt className={fieldLabelClass}>{t('profile.email')}</dt>
              <dd className="mt-1 break-all font-medium">{user.email}</dd>
            </div>
            <div>
              <dt className={fieldLabelClass}>{t('profile.department')}</dt>
              <dd className="mt-1 break-words font-medium">
                {user.department_name ?? t('profile.no_department')}
              </dd>
            </div>
            <div>
              <dt className={fieldLabelClass}>{t('profile.role')}</dt>
              <dd className="mt-1">
                <RoleBadge role={user.role_name} />
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('profile.preferences_heading')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="profile-language">{t('profile.language')}</Label>
            <Select
              id="profile-language"
              value={currentLocale}
              onChange={handleLanguageChange}
              aria-describedby="profile-language-hint"
              className="max-w-xs"
            >
              {SUPPORTED_LOCALES.map((locale) => (
                <option key={locale.code} value={locale.code}>
                  {locale.label}
                </option>
              ))}
            </Select>
            <p id="profile-language-hint" className="text-sm text-muted-foreground">
              {t('profile.language_hint')}
            </p>
          </div>
          {notice && (
            <p
              role={notice.kind === 'error' ? 'alert' : 'status'}
              className={cn('text-sm', notice.kind === 'error' ? 'text-destructive' : 'text-emerald-700')}
            >
              {notice.text}
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('profile.security_heading')}</CardTitle>
        </CardHeader>
        <CardContent>
          <Link to="/change-password" className={buttonVariants({ variant: 'outline' })}>
            {t('profile.change_password')}
          </Link>
        </CardContent>
      </Card>
    </div>
  )
}

/** FR-BB116: /profile forwards to the profile page of the caller's own layout. */
export function ProfileRedirect() {
  const { t } = useTranslation()
  const me = useMe()
  if (me.isPending) return <FullPageSpinner />
  if (me.isError) {
    return (
      <p role="alert" className="p-6 text-sm text-destructive">
        {t('profile.load_failed')}
      </p>
    )
  }
  if (!me.data) return <Navigate to="/login" replace />
  return <Navigate to={me.data.role_name === 'employee' ? '/portal/profile' : '/admin/profile'} replace />
}
