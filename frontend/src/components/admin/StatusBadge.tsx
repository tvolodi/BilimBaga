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
          ? 'bg-bg-success text-success'
          : 'bg-muted text-muted-foreground',
        className,
      )}
    >
      {t(`users.status.${status}`)}
    </span>
  )
}
