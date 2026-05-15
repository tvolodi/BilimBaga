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
    setIsSubmitting(true)

    for (let i = 0; i < data.questions.length; i++) {
      const q = data.questions[i]
      const grade = grades[q.question_id]
      if (!grade || grade.score_pct === null) continue

      try {
        const result = await submitGrade.mutateAsync({
          questionId: q.question_id,
          scorePct: grade.score_pct,
          feedback: grade.feedback,
        })

        if (result.all_graded) {
          toast.show(t('grading.success_toast'), 'success')
          setIsSubmitting(false)
          navigate('/admin/grading')
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

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
  }

  if (isError || !data) {
    return <p className="text-sm text-destructive">{t('common.loadError')}</p>
  }

  const questions = data.questions
  const currentQuestion = questions[currentIndex]

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-xl font-semibold">{t('grading.detail_title')}</h1>
        <p className="text-sm text-muted-foreground">
          {data.employee_name} &mdash; {data.exam_title}
        </p>
      </div>

      {/* Toast */}
      {toast.message && (
        <div
          className={`px-4 py-3 rounded-md text-sm ${
            toast.message.type === 'error'
              ? 'bg-red-50 border border-red-200 text-red-800'
              : 'bg-green-50 border border-green-200 text-green-800'
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
        />
      )}

      {/* Submit */}
      <div className="pt-2">
        <SubmitAllGradesButton
          grades={grades}
          questionCount={questions.length}
          onSubmit={handleSubmitAll}
          isLoading={isSubmitting}
        />
      </div>
    </div>
  )
}
