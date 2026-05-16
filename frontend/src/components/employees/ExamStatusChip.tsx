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
          className: 'bg-green-50 text-green-700 border-green-200',
        }
      : passed === false
        ? {
            icon: XCircle,
            className: 'bg-red-50 text-red-700 border-red-200',
          }
        : {
            icon: Minus,
            className: 'bg-gray-50 text-gray-500 border-gray-200',
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
