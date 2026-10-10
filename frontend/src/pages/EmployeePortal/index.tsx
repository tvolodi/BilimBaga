import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { usePortalExams, useCreateSession } from '@/api/portal'
import type { PortalExam, PortalApiError } from '@/api/portal'
import { ExamCard } from './ExamCard'
import { ExamCardSkeleton } from './ExamCardSkeleton'
import { StartExamModal } from './StartExamModal'
import { EmptyPortal } from './EmptyPortal'
import { requestDocumentFullscreen } from '@/lib/fullscreen'

export function EmployeePortal() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: exams, isLoading, isError } = usePortalExams()
  const [selectedExam, setSelectedExam] = useState<PortalExam | null>(null)

  const createSession = useCreateSession(selectedExam?.id ?? '')

  function handleStart(examId: string) {
    const exam = exams?.find((e) => e.id === examId) ?? null
    setSelectedExam(exam)
  }

  function handleConfirm() {
    // FR-BB319 AC-8: first statement, inside the user's click, so the browser allows fullscreen.
    requestDocumentFullscreen()
    createSession.mutate(undefined, {
      onSuccess: (data) => {
        setSelectedExam(null)
        navigate(`/portal/sessions/${data.session_id}`)
      },
      onError: () => {
        // The exam list may be stale (e.g. a session is already open): refresh it so the card reflects reality.
        void queryClient.invalidateQueries({ queryKey: ['portal', 'exams'] })
      },
    })
  }

  // Map a backend error code to a translated message; unknown codes fall back to a generic one.
  const startError = createSession.error as PortalApiError | null
  const startErrorMessage = startError
    ? t(`portal.startError.${startError.code}`, { defaultValue: t('portal.startError.generic') })
    : null

  function handleClose() {
    setSelectedExam(null)
    createSession.reset()
  }

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-2xl font-bold mb-8">{t('portal.title')}</h1>

        {isError && (
          <p className="text-destructive text-sm mb-6">{t('common.loadError')}</p>
        )}

        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[0, 1, 2].map((i) => (
              <ExamCardSkeleton key={i} />
            ))}
          </div>
        ) : exams && exams.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {exams.map((exam) => (
              <ExamCard key={exam.id} exam={exam} onStart={handleStart} />
            ))}
          </div>
        ) : (
          <EmptyPortal />
        )}

        {selectedExam && (
          <StartExamModal
            exam={selectedExam}
            open
            onClose={handleClose}
            onConfirm={handleConfirm}
            isLoading={createSession.isPending}
            errorMessage={startErrorMessage}
          />
        )}
      </div>
    </div>
  )
}
