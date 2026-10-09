import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { useMyPermissions } from '@/hooks/useMyPermissions'
import { ADMIN_LANDING_ORDER, can } from '@/lib/routeRoles'

/**
 * Index of /admin. Built-in roles go to the dashboard (unchanged). A custom role goes to the
 * first admin page its permissions allow, or sees a "no access" notice, so a role without
 * reports:read can never bounce between /admin and /admin/dashboard (FR-BB117 AC-16).
 */
export function AdminHome() {
  const { t } = useTranslation()
  const { isCustom, permissions, isLoading } = useMyPermissions()
  if (!isCustom) return <Navigate to="dashboard" replace />
  if (isLoading) return <FullPageSpinner />
  const target = ADMIN_LANDING_ORDER.find((a) => can(permissions, a.permission))
  if (target) return <Navigate to={target.path} replace />
  return (
    <div role="status" className="max-w-md space-y-2 py-8">
      <h1 className="text-xl font-semibold">{t('roles.noAccess.title')}</h1>
      <p className="text-sm text-muted-foreground">{t('roles.noAccess.body')}</p>
    </div>
  )
}
