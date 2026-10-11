import { useParams, Navigate } from 'react-router-dom'
import { useExamHistory, type ExamHistorySession } from '@/api/sessions'
import { FullPageSpinner } from '@/components/FullPageSpinner'

// Every attempt fits in one page: the cap is above any max_attempts (FR-BB322 AC-14).
const ALL_ATTEMPTS = 100

/**
 * The session the exam result should open: the one waiting for grading (FR-BB322 AC-14), otherwise the newest attempt.
 * The history is listed oldest first, so "newest" compares started_at.
 */
export function pickResultSession(sessions: ExamHistorySession[]): ExamHistorySession | undefined {
  const waiting = sessions.find((s) => s.status === 'grading_pending')
  if (waiting) return waiting
  return sessions.reduce<ExamHistorySession | undefined>((latest, s) => {
    if (!latest) return s
    return new Date(s.started_at).getTime() > new Date(latest.started_at).getTime() ? s : latest
  }, undefined)
}

/**
 * Redirects /portal/exams/:examId/result → /portal/sessions/:sessionId/result.
 * Used by ExamCard "View result" CTA for completed (passed/failed) exams and by the waiting CTA.
 */
export function ExamResultRedirectPage() {
  const { examId = '' } = useParams<{ examId: string }>()
  const { data, isLoading } = useExamHistory(examId, ALL_ATTEMPTS)

  if (!examId) return <Navigate to="/portal" replace />

  if (isLoading) return <FullPageSpinner />

  const session = pickResultSession(data?.sessions ?? [])
  if (!session?.session_id) {
    return <Navigate to="/portal/results" replace />
  }

  return <Navigate to={`/portal/sessions/${session.session_id}/result`} replace />
}
