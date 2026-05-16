import { useTranslation } from 'react-i18next'
import { CountdownTimer } from './CountdownTimer'
import { SaveIndicator } from './SaveIndicator'
import type { SaveStatus } from './ExamLayout'
import { Button } from '@/components/ui/button'

interface ExamTopBarProps {
  examTitle: string
  remaining: number
  totalSeconds: number
  answeredCount: number
  totalCount: number
  onTimerExpire: () => void
  saveStatus: SaveStatus
  onOpenNavigator: () => void
  /** FR-BB72: when true, show "Questions answered: N" instead of "N of M" */
  adaptive?: boolean
}

export function ExamTopBar({
  examTitle,
  remaining,
  totalSeconds,
  answeredCount,
  totalCount,
  onTimerExpire,
  saveStatus,
  onOpenNavigator,
  adaptive = false,
}: ExamTopBarProps) {
  const { t } = useTranslation()

  return (
    <header className="flex items-center justify-between gap-4 px-4 py-2 border-b bg-background shadow-sm shrink-0 flex-wrap">
      <h1 className="text-base font-semibold truncate max-w-[200px] md:max-w-sm">{examTitle}</h1>

      <div className="flex items-center gap-3">
        <SaveIndicator status={saveStatus} />

        <CountdownTimer
          remainingSeconds={remaining}
          totalSeconds={totalSeconds}
          onExpire={onTimerExpire}
        />

        <span className="text-sm text-muted-foreground hidden sm:inline">
          {adaptive
            ? t('session.questionsAnswered', { count: answeredCount })
            : t('exam.taking.progress', { answered: answeredCount, total: totalCount })}
        </span>

        {/* Mobile navigator trigger — hidden in adaptive mode (no question list) */}
        {!adaptive && (
          <Button
            variant="outline"
            size="sm"
            className="md:hidden"
            onClick={onOpenNavigator}
            aria-label={t('exam.taking.navigator.title')}
          >
            {t('exam.taking.navigator.title')}
          </Button>
        )}
      </div>
    </header>
  )
}
