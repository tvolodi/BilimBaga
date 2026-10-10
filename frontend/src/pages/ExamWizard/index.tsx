import { useState, useCallback } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useExam } from '@/api/exams'
import { ExamNotFoundState } from '@/components/ExamNotFoundState'
import { isNotFoundError } from '@/lib/apiRetry'
import { Step1BasicSettings } from './Step1BasicSettings'
import { Step2QuestionRules } from './Step2QuestionRules'
import { Step3Assignments } from './Step3Assignments'
import { Step4Review } from './Step4Review'

// ---- Types ------------------------------------------------------------------

export interface WizardExamId {
  examId: string
  sectionId: string | null
}

// ---- Helpers ----------------------------------------------------------------

function StepIndicator({ currentStep, totalSteps, labels }: {
  currentStep: number
  totalSteps: number
  labels: string[]
}) {
  return (
    <nav className="flex items-center gap-0 mb-8">
      {Array.from({ length: totalSteps }).map((_, i) => {
        const step = i + 1
        const isActive = step === currentStep
        const isDone = step < currentStep
        return (
          <div key={step} className="flex items-center">
            <div className={`flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium transition-colors ${
              isActive
                ? 'bg-primary text-primary-foreground'
                : isDone
                ? 'bg-primary/20 text-foreground'
                : 'bg-muted text-muted-foreground'
            }`}>
              {step}
            </div>
            <span className={`ml-2 text-sm hidden sm:inline ${
              isActive ? 'text-foreground font-medium' : 'text-muted-foreground'
            }`}>
              {labels[i]}
            </span>
            {i < totalSteps - 1 && (
              <div className={`w-8 h-px mx-2 ${isDone ? 'bg-primary' : 'bg-border'}`} />
            )}
          </div>
        )
      })}
    </nav>
  )
}

// ---- Main component ---------------------------------------------------------

interface ExamWizardProps {
  examId?: string
}

export function ExamWizard({ examId: examIdProp }: ExamWizardProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [searchParams] = useSearchParams()

  const initialStep = examIdProp
    ? Math.max(1, Math.min(4, Number(searchParams.get('step')) || 1))
    : 1

  const [step, setStep] = useState(initialStep)
  const [resolvedExamId, setResolvedExamId] = useState<string | null>(examIdProp ?? null)
  const [resolvedSectionId, setResolvedSectionId] = useState<string | null>(null)

  // In edit mode, pre-load the exam to populate forms
  const { data: exam, isLoading: examLoading, error: examError } = useExam(resolvedExamId)

  const isEditMode = !!examIdProp

  const stepLabels = [
    t('exam.wizard.step1.title'),
    t('exam.wizard.step2.title'),
    t('exam.wizard.step3.title'),
    t('exam.wizard.step4.title'),
  ]

  // Called by Step1 after create/update succeeds; advances to step 2
  const handleStep1Done = useCallback((eid: string, sid: string | null) => {
    setResolvedExamId(eid)
    setResolvedSectionId(sid)
    setStep(2)
  }, [])

  const currentUser = qc.getQueryData<{ id: string; email: string; role: string }>(['auth', 'currentUser'])
  const canMutateAssignments =
    currentUser?.role === 'admin' ||
    currentUser?.role === 'super_admin' ||
    currentUser?.role === 'department_admin'

  // #470: an unknown exam id in edit mode gets the not-found state, not a wizard with no data.
  if (isEditMode && examError && isNotFoundError(examError)) return <ExamNotFoundState />

  if (examLoading && isEditMode) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-sm text-muted-foreground">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">
        {isEditMode ? t('exam.wizard.editTitle') : t('exam.wizard.createTitle')}
      </h1>

      <StepIndicator currentStep={step} totalSteps={4} labels={stepLabels} />

      {step === 1 && (
        <Step1BasicSettings
          exam={exam ?? null}
          onDone={handleStep1Done}
          resolvedSectionId={resolvedSectionId}
        />
      )}

      {step === 2 && resolvedExamId && (
        <Step2QuestionRules
          examId={resolvedExamId}
          sectionId={resolvedSectionId}
          exam={exam}
          onBack={() => setStep(1)}
          onNext={() => setStep(3)}
        />
      )}

      {step === 3 && resolvedExamId && (
        <Step3Assignments
          examId={resolvedExamId}
          canMutate={canMutateAssignments}
          onBack={() => setStep(2)}
          onNext={() => setStep(4)}
        />
      )}

      {step === 4 && resolvedExamId && (
        <Step4Review
          examId={resolvedExamId}
          onBack={() => setStep(3)}
          onPublished={() => navigate(`/admin/exams`)}
        />
      )}
    </div>
  )
}

// ---- Routable wrapper -------------------------------------------------------

export function ExamWizardCreatePage() {
  return <ExamWizard />
}

export function ExamWizardEditPage() {
  const { id } = useParams<{ id: string }>()
  return <ExamWizard examId={id} />
}
