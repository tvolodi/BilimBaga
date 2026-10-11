import { useEffect, useRef, useState } from 'react'
import { useParams, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useSessionResult } from '@/api/sessions'
import { usePortalExam } from '@/api/portal'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { Button } from '@/components/ui/button'
import { ScoreDial } from '@/components/results/ScoreDial'
import { PassFailBanner } from '@/components/results/PassFailBanner'
import { TimeTakenBadge } from '@/components/results/TimeTakenBadge'
import { SectionScores } from '@/components/results/SectionScores'
import { QuestionBreakdownTable } from '@/components/results/QuestionBreakdownTable'
import { ResultActions } from '@/components/results/ResultActions'

export function ResultPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()

  const {
    data: result,
    isLoading: resultLoading,
    isError: resultError,
    pollingStopped,
    refresh,
  } = useSessionResult(sessionId ?? '')

  const {
    data: exam,
    isLoading: examLoading,
  } = usePortalExam(result?.exam_id)

  // FR-BB322 AC-7: when a waiting result turns final, announce it and refresh the lists that show the attempt.
  const waitedRef = useRef(false)
  const [announceReady, setAnnounceReady] = useState(false)
  useEffect(() => {
    if (!result) return
    if (result.status === 'grading_pending') {
      waitedRef.current = true
      return
    }
    if (!waitedRef.current) return
    waitedRef.current = false
    setAnnounceReady(true)
    void qc.invalidateQueries({ queryKey: ['portal', 'exams'] })
    void qc.invalidateQueries({ queryKey: ['my-results'] })
    void qc.invalidateQueries({ queryKey: ['exam-history', result.exam_id] })
    void qc.invalidateQueries({ queryKey: ['session-result', result.session_id] })
  }, [result, qc])

  if (!sessionId) return <Navigate to="/portal" replace />

  if (resultLoading || (result && examLoading)) {
    return <FullPageSpinner />
  }

  if (resultError || !result) {
    return (
      <main id="main-content" tabIndex={-1} className="min-h-screen flex items-center justify-center bg-background outline-none">
        <p className="text-destructive text-sm">{t('common.loadError')}</p>
      </main>
    )
  }

  const isPending = result.status === 'grading_pending'
  const canRetake =
    !!exam &&
    result.attempt_number < exam.max_attempts &&
    exam.user_status !== 'expired'
  const certificateEnabled = exam?.certificate_enabled ?? false

  return (
    <main id="main-content" tabIndex={-1} className="min-h-screen bg-background flex items-center justify-center p-6 outline-none">
      <div className="w-full max-w-2xl space-y-6">
        {/* Title */}
        <div className="text-center">
          <h1 className="text-2xl font-bold">{t('result.title')}</h1>
          <p className="text-muted-foreground mt-1">{result.exam_title}</p>
          <p className="text-xs text-muted-foreground mt-1">
            {t('result.attempt_number')}: {result.attempt_number}
          </p>
        </div>

        {/* Result ready: announced once for assistive technology (AC-7) */}
        <div aria-live="polite" className="sr-only">
          {announceReady ? t('result.ready') : ''}
        </div>

        {isPending ? (
          /* FR-BB322 AC-5: what happens next, how far it has got, and when the answers were sent */
          <div className="rounded-lg border border-warning bg-bg-warning px-4 py-4 text-center text-sm text-warning space-y-2">
            <p>{result.grading_expected_by === 'ai' ? t('result.pending_ai') : t('result.pending_examiner')}</p>
            <p>
              {t('result.pending_count', {
                pending: result.pending_question_count,
                total: result.total_gradable_count,
              })}
            </p>
            {result.submitted_at && (
              <p className="text-xs">
                {t('result.pending_submitted', {
                  date: new Date(result.submitted_at).toLocaleString(i18n.language),
                })}
              </p>
            )}
            {pollingStopped && (
              /* AC-6: polling stopped after 15 minutes; Refresh fetches once and starts the 15 minutes again */
              <div className="space-y-2 pt-1">
                <p>{t('result.pending_stopped')}</p>
                <Button variant="outline" size="sm" onClick={() => void refresh()}>
                  {t('result.refresh')}
                </Button>
              </div>
            )}
          </div>
        ) : (
          /* Score dial + pass/fail */
          <div className="flex flex-col items-center gap-4">
            <div>
              <p className="text-center text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wide">
                {t('result.score_label')}
              </p>
              <ScoreDial scorePct={result.score_pct!} />
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
    </main>
  )
}
