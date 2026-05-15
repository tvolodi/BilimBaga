import { useParams, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useSessionResult } from '@/api/sessions'
import { usePortalExam } from '@/api/portal'
import { useTenantConfig } from '@/api/useTenantConfig'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { ScoreDial } from '@/components/results/ScoreDial'
import { PassFailBanner } from '@/components/results/PassFailBanner'
import { TimeTakenBadge } from '@/components/results/TimeTakenBadge'
import { SectionScores } from '@/components/results/SectionScores'
import { QuestionBreakdownTable } from '@/components/results/QuestionBreakdownTable'
import { ResultActions } from '@/components/results/ResultActions'

export function ResultPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const { t } = useTranslation()
  const { data: tenantConfig } = useTenantConfig()

  const {
    data: result,
    isLoading: resultLoading,
    isError: resultError,
  } = useSessionResult(sessionId ?? '')

  const {
    data: exam,
    isLoading: examLoading,
  } = usePortalExam(result?.exam_id)

  if (!sessionId) return <Navigate to="/portal" replace />

  if (resultLoading || (result && examLoading)) {
    return <FullPageSpinner />
  }

  if (resultError || !result) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <p className="text-destructive text-sm">{t('common.loadError')}</p>
      </div>
    )
  }

  const primaryColor = tenantConfig?.primary_color ?? '#6366f1'
  const isPending = result.score_pct === null
  const canRetake =
    !!exam &&
    result.attempt_number < exam.max_attempts &&
    exam.user_status !== 'expired'
  const certificateEnabled = exam?.certificate_enabled ?? false

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="w-full max-w-2xl space-y-6">
        {/* Title */}
        <div className="text-center">
          <h1 className="text-2xl font-bold">{t('result.title')}</h1>
          <p className="text-muted-foreground mt-1">{result.exam_title}</p>
          <p className="text-xs text-muted-foreground mt-1">
            {t('result.attempt_number')}: {result.attempt_number}
          </p>
        </div>

        {isPending ? (
          /* Pending manual review notice */
          <div className="rounded-lg border border-yellow-400 bg-yellow-50 px-4 py-4 text-center text-sm text-yellow-800">
            {t('result.pending_grading')}
          </div>
        ) : (
          /* Score dial + pass/fail */
          <div className="flex flex-col items-center gap-4">
            <div>
              <p className="text-center text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wide">
                {t('result.score_label')}
              </p>
              <ScoreDial scorePct={result.score_pct!} primaryColor={primaryColor} />
            </div>
            <PassFailBanner passed={result.passed} />
          </div>
        )}

        {/* Time taken */}
        <div className="text-center">
          <TimeTakenBadge seconds={result.time_taken_seconds} />
        </div>

        {/* Section scores */}
        <SectionScores sections={result.per_section_scores} />

        {/* Question breakdown */}
        <QuestionBreakdownTable breakdown={result.per_question_breakdown} />

        {/* Actions */}
        <ResultActions
          sessionId={result.session_id}
          passed={isPending ? false : result.passed}
          certificateEnabled={isPending ? false : certificateEnabled}
          canRetake={isPending ? false : canRetake}
          examId={result.exam_id}
        />
      </div>
    </div>
  )
}
