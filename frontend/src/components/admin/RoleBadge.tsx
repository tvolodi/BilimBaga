import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

interface RoleBadgeProps {
  role: string
  className?: string
}

const roleStyles: Record<string, string> = {
  super_admin: 'bg-purple-100 text-purple-800',
  department_admin: 'bg-blue-100 text-blue-800',
  examiner: 'bg-teal-100 text-teal-800',
  employee: 'bg-gray-100 text-gray-700',
}

export function RoleBadge({ role, className }: RoleBadgeProps) {
  const { t } = useTranslation()
  const style = roleStyles[role] ?? 'bg-gray-100 text-gray-700'
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
