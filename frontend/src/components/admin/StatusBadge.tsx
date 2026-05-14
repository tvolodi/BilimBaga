import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

interface StatusBadgeProps {
  status: 'active' | 'inactive'
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const { t } = useTranslation()

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold',
        status === 'active'
          ? 'bg-green-100 text-green-800'
          : 'bg-gray-100 text-gray-600',
        className,
      )}
    >
      {t(`users.status.${status}`)}
    </span>
  )
}
