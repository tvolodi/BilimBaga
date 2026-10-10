import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

interface RoleBadgeProps {
  role: string
  className?: string
}

const roleStyles: Record<string, string> = {
  super_admin: 'bg-primary text-primary-foreground',
  department_admin: 'bg-primary/10 text-foreground',
  examiner: 'bg-bg-info text-info',
  employee: 'bg-muted text-muted-foreground',
}

export function RoleBadge({ role, className }: RoleBadgeProps) {
  const { t } = useTranslation()
  const style = roleStyles[role] ?? 'bg-muted text-muted-foreground'
  const label = t(`users.roles.${role}`, { defaultValue: role })

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
        style,
        className,
      )}
    >
      {label}
    </span>
  )
}
