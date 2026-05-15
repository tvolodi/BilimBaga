import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { SaveStatus } from './ExamLayout'

interface SaveIndicatorProps {
  status: SaveStatus
}

export function SaveIndicator({ status }: SaveIndicatorProps) {
  const { t } = useTranslation()

  if (status === 'idle') return null

  return (
    <span
      className={cn(
        'text-xs font-medium transition-opacity',
        status === 'saving' && 'text-muted-foreground',
        status === 'saved' && 'text-green-600 dark:text-green-400',
        status === 'error' && 'text-destructive',
      )}
      aria-live="polite"
    >
      {status === 'saving' && t('exam.taking.save.saving')}
      {status === 'saved' && t('exam.taking.save.saved')}
      {status === 'error' && t('exam.taking.save.error')}
    </span>
  )
}
