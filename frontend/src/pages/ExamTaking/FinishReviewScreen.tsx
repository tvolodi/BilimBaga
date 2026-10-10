import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { SessionQuestion } from '@/api/sessions'

interface FinishReviewScreenProps {
  questions: SessionQuestion[]
  answeredIds: Set<string>
  flaggedIds: Set<string>
  onGoBack: () => void
  onSubmitAnyway: () => void
}

export function FinishReviewScreen({
  questions,
  answeredIds,
  flaggedIds,
  onGoBack,
  onSubmitAnyway,
}: FinishReviewScreenProps) {
  const { t } = useTranslation()

  const unanswered = questions.filter((q) => !answeredIds.has(q.id))
  const flagged = questions.filter((q) => flaggedIds.has(q.id))

  const indexOf = (q: SessionQuestion) => questions.indexOf(q) + 1

  return (
    <div className="min-h-screen bg-background flex items-start justify-center py-12 px-4">
      <div className="w-full max-w-lg space-y-6">
        <h1 className="text-xl font-bold">{t('exam.taking.review.title')}</h1>

        {unanswered.length > 0 && (
          <section>
            <h2 className="text-sm font-semibold text-destructive mb-2">
              {t('exam.taking.review.unanswered')} ({unanswered.length})
            </h2>
            <div className="flex flex-wrap gap-2">
              {unanswered.map((q) => (
                <span
                  key={q.id}
                  className="inline-flex items-center justify-center w-8 h-8 rounded border border-border text-sm"
                >
                  {indexOf(q)}
                </span>
              ))}
            </div>
          </section>
        )}

        {flagged.length > 0 && (
          <section>
            <h2 className="text-sm font-semibold text-warning mb-2">
              {t('exam.taking.review.flagged')} ({flagged.length})
            </h2>
            <div className="flex flex-wrap gap-2">
              {flagged.map((q) => (
                <span
                  key={q.id}
                  className="inline-flex items-center justify-center w-8 h-8 rounded bg-bg-warning border border-warning text-sm font-medium"
                >
                  {indexOf(q)}
                </span>
              ))}
            </div>
          </section>
        )}

        {unanswered.length === 0 && flagged.length === 0 && (
          <p className="text-sm text-muted-foreground">{t('exam.taking.review.allAnswered')}</p>
        )}

        <div className="flex gap-3 pt-4">
          <Button variant="outline" className="min-h-11" onClick={onGoBack}>
            {t('exam.taking.review.goBack')}
          </Button>
          <Button className="min-h-11" onClick={onSubmitAnyway}>
            {t('exam.taking.review.submitAnyway')}
          </Button>
        </div>
      </div>
    </div>
  )
}
