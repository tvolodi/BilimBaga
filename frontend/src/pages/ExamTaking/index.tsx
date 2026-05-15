import { useState } from 'react'
import { useParams, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useSession, type SubmitResult } from '@/api/sessions'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { ExamLayout } from './ExamLayout'
import { ResultScreen } from './ResultScreen'

export function ExamTakingPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const { t } = useTranslation()
  const [submitResult, setSubmitResult] = useState<SubmitResult | null>(null)

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

  if (submitResult) {
    return (
      <ResultScreen
        result={submitResult}
        examTitle={session.exam_title}
        certificateEnabled={session.certificate_enabled}
      />
    )
  }

  if (session.status === 'submitted' || session.status === 'auto_submitted') {
    return (
      <ResultScreen
        result={{
          session_id: session.session_id,
          status: session.status,
          submitted_at: session.expires_at,
          score_pct: null,
          passed: null,
        }}
        examTitle={session.exam_title}
        certificateEnabled={session.certificate_enabled}
      />
    )
  }

  return (
    <ExamLayout
      session={session}
      onSubmitSuccess={setSubmitResult}
    />
  )
}
