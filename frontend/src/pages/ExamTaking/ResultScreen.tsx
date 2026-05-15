import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import type { SubmitResult } from '@/api/sessions'

interface ResultScreenProps {
  result: SubmitResult
  examTitle: string
  certificateEnabled: boolean
}

export function ResultScreen({ result, examTitle, certificateEnabled: _certificateEnabled }: ResultScreenProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const isPending = result.passed === null
  const isPassed = result.passed === true
  const isFailed = result.passed === false

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="w-full max-w-sm text-center space-y-6">
        <h1 className="text-xl font-semibold text-muted-foreground">{examTitle}</h1>

        {isPending && (
          <div className="space-y-2">
            <div className="text-4xl">⏳</div>
            <p className="text-lg font-semibold">{t('exam.taking.result.pending')}</p>
          </div>
        )}

        {isPassed && (
          <div className="space-y-2">
            <div className="text-4xl">✅</div>
            <p className="text-2xl font-bold text-green-600 dark:text-green-400">
              {t('exam.taking.result.passed')}
            </p>
            {result.score_pct !== null && (
              <p className="text-muted-foreground">
                {t('exam.taking.result.score', { score: Math.round(result.score_pct) })}
              </p>
            )}
          </div>
        )}

        {isFailed && (
          <div className="space-y-2">
            <div className="text-4xl">❌</div>
            <p className="text-2xl font-bold text-destructive">
              {t('exam.taking.result.failed')}
            </p>
            {result.score_pct !== null && (
              <p className="text-muted-foreground">
                {t('exam.taking.result.score', { score: Math.round(result.score_pct) })}
              </p>
            )}
          </div>
        )}

        <div className="flex flex-col gap-2 pt-4">
          <Button onClick={() => navigate('/portal')}>
            {t('exam.taking.result.backToPortal')}
          </Button>
        </div>
      </div>
    </div>
  )
}
