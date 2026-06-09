import { useParams, Navigate, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useSession } from '@/api/sessions'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { ExamLayout } from './ExamLayout'

export function ExamTakingPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data: session, isLoading, isError } = useSession(sessionId ?? '')

  if (!sessionId) return <Navigate to="/portal" replace />

  if (isLoading) return <FullPageSpinner />

  if (isError || !session) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <p className="text-destructive text-sm">{t('common.loadError')}</p>
      </div>
    )
  }

  if (session.status === 'submitted' || session.status === 'auto_submitted') {
    return <Navigate to={`/portal/sessions/${session.session_id}/result`} replace />
  }

  return (
    <ExamLayout
      session={session}
      onSubmitSuccess={(result) => navigate(`/portal/sessions/${result.session_id}/result`)}
    />
  )
}
