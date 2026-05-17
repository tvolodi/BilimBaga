import { useParams, Navigate } from 'react-router-dom'
import { useExamHistory } from '@/api/sessions'
import { FullPageSpinner } from '@/components/FullPageSpinner'

/**
 * Redirects /portal/exams/:examId/result → /portal/sessions/:latestSessionId/result.
 * Used by ExamCard "View result" CTA for completed (passed/failed) exams.
 */
export function ExamResultRedirectPage() {
  const { examId = '' } = useParams<{ examId: string }>()
  const { data, isLoading } = useExamHistory(examId)

  if (!examId) return <Navigate to="/portal" replace />

  if (isLoading) return <FullPageSpinner />

  const latestSession = data?.sessions?.[0]
  if (!latestSession?.session_id) {
    return <Navigate to="/portal/results" replace />
  }

  return <Navigate to={`/portal/sessions/${latestSession.session_id}/result`} replace />
}
