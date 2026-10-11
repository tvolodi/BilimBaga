import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useGradingSession, useSubmitGrade } from '@/api/grading'
import type { GradingQuestion } from '@/api/grading'
import { QuestionGrader } from '@/components/grading/QuestionGrader'
import { GradingNavigation } from '@/components/grading/GradingNavigation'
import { SubmitAllGradesButton } from '@/components/grading/SubmitAllGradesButton'
import type { GradeMap } from '@/components/grading/SubmitAllGradesButton'

// Simple inline toast helper
function useToast() {
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null)

  function show(text: string, type: 'success' | 'error') {
    setMessage({ text, type })
    setTimeout(() => setMessage(null), 5000)
  }

  return { message, show }
}

function buildInitialGrades(questions: GradingQuestion[]): GradeMap {
  const map: GradeMap = {}
  for (const q of questions) {
    map[q.question_id] = {
      score_pct: q.current_score_pct,
      feedback: q.manual_feedback ?? '',
    }
  }
  return map
}

// A final session is one whose result is out (FR-BB47 AC-5): its graded answers are read-only.
const FINAL_SESSION_STATUSES: readonly string[] = ['submitted', 'auto_submitted']

/**
 * A grade is posted when the answer still waits, or when the examiner changed it. A graded answer that is unchanged is
 * not posted again: a final session refuses it with 409 SESSION_FINALISED (FR-BB73 AC-19).
 */
function needsPosting(q: GradingQuestion, grade: GradeMap[string] | undefined): boolean {
  if (!grade || grade.score_pct === null) return false
  if (q.grading_status === 'pending_manual') return true
  return grade.score_pct !== q.current_score_pct || grade.feedback !== (q.manual_feedback ?? '')
}

export function GradingDetailPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()

  const { data, isLoading, isError } = useGradingSession(sessionId ?? '')
  const submitGrade = useSubmitGrade(sessionId ?? '')

  const [grades, setGrades] = useState<GradeMap>({})
  const [currentIndex, setCurrentIndex] = useState(0)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [acceptingId, setAcceptingId] = useState<string | null>(null)

  // Initialise grades from API response
  useEffect(() => {
    if (data) {
      setGrades(buildInitialGrades(data.questions))
    }
  }, [data])

  const handleGradeChange = useCallback(
    (questionId: string, score: number | null, feedback: string) => {
      setGrades((prev) => ({ ...prev, [questionId]: { score_pct: score, feedback } }))
    },
    [],
  )

  async function handleSubmitAll() {
    if (!data) return
    const pending = data.questions.map((q, i) => ({ q, i })).filter(({ q }) => needsPosting(q, grades[q.question_id]))
    if (pending.length === 0) {
      toast.show(t('grading.nothing_to_submit'), 'success')
      return
    }
    setIsSubmitting(true)

    for (const { q, i } of pending) {
      const grade = grades[q.question_id]
      if (!grade || grade.score_pct === null) continue

      try {
        const result = await submitGrade.mutateAsync({
          questionId: q.question_id,
          scorePct: grade.score_pct,
          feedback: grade.feedback,
        })

        if (result.all_graded) {
          setIsSubmitting(false)
          // The success toast is rendered by the queue page (this page unmounts);
          // pass a flag via navigation state (FR-BB47 AC-9, #84).
          navigate('/admin/grading', { state: { gradingSuccess: true } })
          return
        }
      } catch {
        toast.show(t('grading.error_toast', { question: i + 1 }), 'error')
        setIsSubmitting(false)
        return
      }
    }

    setIsSubmitting(false)
  }

  // FR-BB73 AC-17: accepting the suggestion posts the suggested value as the grade.
  async function handleAccept(question: GradingQuestion) {
    if (question.ai_suggested_score_pct === null) return
    const feedback = grades[question.question_id]?.feedback ?? ''
    setAcceptingId(question.question_id)
    try {
      const result = await submitGrade.mutateAsync({
        questionId: question.question_id,
        scorePct: question.ai_suggested_score_pct,
        feedback,
        accepted: true,
      })
      if (result.all_graded) {
        navigate('/admin/grading', { state: { gradingSuccess: true } })
      }
    } catch {
      toast.show(t('grading.error_toast', { question: currentIndex + 1 }), 'error')
    } finally {
      setAcceptingId(null)
    }
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
  }

  if (isError || !data) {
    return <p className="text-sm text-destructive">{t('common.loadError')}</p>
  }

  const questions = data.questions
  const currentQuestion = questions[currentIndex]
  const isFinal = FINAL_SESSION_STATUSES.includes(data.session_status)
  // A final session has nothing to submit unless an AI grade can still be overridden (FR-BB73 AC-19).
  const canSubmit = !isFinal || questions.some((q) => q.grading_status === 'ai_graded')

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-xl font-semibold">{t('grading.detail_title')}</h1>
        <p className="text-sm text-muted-foreground">
          {data.employee_name} &mdash; {data.exam_title}
        </p>
        {isFinal && (
          <p className="mt-2 text-sm text-muted-foreground">{t('grading.suggestion.final_read_only')}</p>
        )}
      </div>

      {/* Toast */}
      {toast.message && (
        <div
          className={`px-4 py-3 rounded-md text-sm ${
            toast.message.type === 'error'
              ? 'bg-bg-danger border border-danger text-danger'
              : 'bg-bg-success border border-success text-success'
          }`}
        >
          {toast.message.text}
        </div>
      )}

      {/* Navigation */}
      <GradingNavigation
        current={currentIndex + 1}
        total={questions.length}
        onPrev={() => setCurrentIndex((i) => Math.max(0, i - 1))}
        onNext={() => setCurrentIndex((i) => Math.min(questions.length - 1, i + 1))}
      />

      {/* Grader */}
      {currentQuestion && (
        <QuestionGrader
          question={currentQuestion}
          score={grades[currentQuestion.question_id]?.score_pct ?? null}
          feedback={grades[currentQuestion.question_id]?.feedback ?? ''}
          onChange={(score, feedback) =>
            handleGradeChange(currentQuestion.question_id, score, feedback)
          }
          readOnly={isFinal && currentQuestion.grading_status === 'graded'}
          onAccept={() => handleAccept(currentQuestion)}
          isAccepting={acceptingId === currentQuestion.question_id}
        />
      )}

      {/* Submit */}
      {canSubmit && (
        <div className="pt-2">
          <SubmitAllGradesButton
            grades={grades}
            questionCount={questions.length}
            onSubmit={handleSubmitAll}
            isLoading={isSubmitting}
          />
        </div>
      )}
    </div>
  )
}
