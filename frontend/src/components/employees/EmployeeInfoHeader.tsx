import { useTranslation } from 'react-i18next'
import { RoleBadge } from '@/components/admin/RoleBadge'
import { StatusBadge } from '@/components/admin/StatusBadge'
import type { User } from '@/api/users'

interface EmployeeInfoHeaderProps {
  user: User
}

export function EmployeeInfoHeader({ user }: EmployeeInfoHeaderProps) {
  const { t } = useTranslation()

  const initials = user.full_name
    .split(' ')
    .map((n) => n[0] ?? '')
    .join('')
    .toUpperCase()
    .slice(0, 2)

  return (
    <div className="flex items-start gap-4 p-6 rounded-lg border bg-card">
      {/* Avatar with initials */}
      <div className="flex-shrink-0 w-16 h-16 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-xl font-bold select-none">
        {initials}
      </div>

      {/* Info column */}
      <div className="flex-1 space-y-2">
        <h2 className="text-xl font-semibold">{user.full_name}</h2>

        <div className="flex flex-wrap items-center gap-2">
          <RoleBadge role={user.role_name} />
          <StatusBadge status={user.status} />
        </div>

        <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm">
          <span>
            <span className="text-muted-foreground">{t('users.columns.department')}: </span>
            <span>{user.department_name ?? '—'}</span>
          </span>
          <span>
            <a href={`mailto:${user.email}`} className="text-primary hover:underline">
              {user.email}
            </a>
          </span>
        </div>
      </div>
    </div>
  )
}
