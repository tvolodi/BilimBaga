import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { SessionQuestion } from '@/api/sessions'

interface QuestionNavigatorProps {
  questions: SessionQuestion[]
  answeredIds: Set<string>
  flaggedIds: Set<string>
  onNavigate: (questionId: string) => void
}

export function QuestionNavigator({
  questions,
  answeredIds,
  flaggedIds,
  onNavigate,
}: QuestionNavigatorProps) {
  const { t } = useTranslation()

  return (
    <div>
      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
        {t('exam.taking.navigator.title')}
      </p>
      <div className="grid grid-cols-5 gap-1.5">
        {questions.map((q, idx) => {
          const isFlagged = flaggedIds.has(q.id)
          const isAnswered = answeredIds.has(q.id)

          return (
            <button
              key={q.id}
              onClick={() => onNavigate(q.id)}
              className={cn(
                'w-8 h-8 rounded text-xs font-medium border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                isFlagged
                  ? 'bg-bg-warning border-warning text-warning hover:bg-warning/20'
                  : isAnswered
                    ? 'bg-info border-info text-primary-foreground hover:bg-info/90'
                    : 'bg-background border-border text-foreground hover:bg-muted',
              )}
              title={`${t('exam.taking.navigator.title')} ${idx + 1}`}
            >
              {idx + 1}
            </button>
          )
        })}
      </div>

      {/* Legend */}
      <div className="mt-4 space-y-1.5">
        <LegendItem color="bg-background border border-border" label={t('exam.taking.navigator.unanswered')} />
        <LegendItem color="bg-info" label={t('exam.taking.navigator.answered')} />
        <LegendItem color="bg-warning" label={t('exam.taking.navigator.flagged')} />
      </div>
    </div>
  )
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <span className={cn('inline-block w-3 h-3 rounded-sm', color)} />
      {label}
    </div>
  )
}
