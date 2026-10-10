import { useTranslation } from 'react-i18next'
import { CheckCircle, XCircle, Minus } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ExamStatusChipProps {
  title: string
  passed: boolean | null
  attempts: number
}

export function ExamStatusChip({ title, passed, attempts }: ExamStatusChipProps) {
  const { t } = useTranslation()

  const tooltip = `${title} — ${t('employee_record.attempts', { count: attempts })}`

  const config =
    passed === true
      ? {
          icon: CheckCircle,
          className: 'bg-bg-success text-success border-transparent',
        }
      : passed === false
        ? {
            icon: XCircle,
            className: 'bg-bg-danger text-danger border-transparent',
          }
        : {
            icon: Minus,
            className: 'bg-muted text-muted-foreground border-border',
          }

  const { icon: Icon, className } = config

  return (
    <span
      title={tooltip}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium',
        className,
      )}
    >
      <Icon size={12} aria-hidden />
      <span className="truncate max-w-[10rem]">{title}</span>
    </span>
  )
}
