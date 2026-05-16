import { useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  useSaveAnswer,
  useSubmitSession,
  type ResumeSessionResponse,
  type SavedAnswer,
  type SubmitResult,
} from '@/api/sessions'
import { useCountdownTimer } from '@/hooks/useCountdownTimer'
import { useTabSwitchDetection } from '@/hooks/useTabSwitchDetection'
import { ExamTopBar } from './ExamTopBar'
import { QuestionNavigator } from './QuestionNavigator'
import { QuestionDisplay } from './QuestionDisplay'
import { FinishReviewScreen } from './FinishReviewScreen'
import { SubmitConfirmModal } from './SubmitConfirmModal'
import { TabSwitchWarningModal } from './TabSwitchWarningModal'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'
type Screen = 'exam' | 'review'

interface ExamLayoutProps {
  session: ResumeSessionResponse
  onSubmitSuccess: (result: SubmitResult) => void
}

export function ExamLayout({ session, onSubmitSuccess }: ExamLayoutProps) {
  const { t } = useTranslation()

  // Timer
  const totalSeconds = Math.max(
    1,
    Math.floor(
      (new Date(session.expires_at).getTime() - new Date(session.started_at).getTime()) / 1000,
    ),
  )
  const { remaining, syncFromServer } = useCountdownTimer(
    session.remaining_seconds,
    session.expires_at,
  )

  // Local answers & flags
  const [localAnswers, setLocalAnswers] = useState<Record<string, SavedAnswer>>(
    () => session.answers,
  )
  const [flags, setFlags] = useState<Set<string>>(new Set<string>())

  // Save status
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const saveStatusTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const saveDebounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Screen / modal state
  const [screen, setScreen] = useState<Screen>('exam')
  const [showSubmitConfirm, setShowSubmitConfirm] = useState(false)
  const [showTabWarning, setShowTabWarning] = useState(false)
  const [showTimesUp, setShowTimesUp] = useState(false)
  const [showNavigatorMobile, setShowNavigatorMobile] = useState(false)

  // Submitted result (for auto-submit via tab switch) - reserved for future use
  // const [submitResult, setSubmitResult] = useState<SubmitResult | null>(null)

  // Mutations
  const saveAnswer = useSaveAnswer(session.session_id)
  const submitSession = useSubmitSession(session.session_id)

  // Refs for scrolling
  const questionRefs = useRef<Record<string, HTMLDivElement | null>>({})

  // Auto-submit on timer expiry (retry up to 3 times every 5s)
  const doAutoSubmitRef = useRef<((retryCount: number) => void) | undefined>(undefined)
  doAutoSubmitRef.current = (retryCount: number) => {
    submitSession.mutate(undefined, {
      onSuccess: (result) => {
        setShowTimesUp(false)
        onSubmitSuccess(result)
      },
      onError: () => {
        if (retryCount < 2) {
          setTimeout(() => doAutoSubmitRef.current?.(retryCount + 1), 5000)
        }
      },
    })
  }

  const handleTimerExpire = useCallback(() => {
    setShowTimesUp(true)
    doAutoSubmitRef.current?.(0)
  }, [])

  // Tab switch detection
  const handleTabWarn = useCallback(() => setShowTabWarning(true), [])
  const handleTabAutoSubmit = useCallback(() => {
    // The session was auto-submitted server-side due to tab switch policy
    // Re-fetch session or go to result with pending state
    setShowTabWarning(false)
    const pending: SubmitResult = {
      session_id: session.session_id,
      status: 'auto_submitted',
      submitted_at: new Date().toISOString(),
      score_pct: null,
      passed: null,
    }
    onSubmitSuccess(pending)
  }, [session.session_id, onSubmitSuccess])

  useTabSwitchDetection(session.session_id, handleTabWarn, handleTabAutoSubmit)

  // Reset save status auto-dismiss helper
  function scheduleSaveStatusReset(ms: number) {
    if (saveStatusTimerRef.current) clearTimeout(saveStatusTimerRef.current)
    saveStatusTimerRef.current = setTimeout(() => setSaveStatus('idle'), ms)
  }

  // Core save trigger
  const triggerSave = useCallback(
    (questionId: string, answer: SavedAnswer) => {
      setSaveStatus('saving')
      saveAnswer.mutate(
        {
          questionId,
          answer: {
            selected_option_ids: answer.selected_option_ids,
            text_answer: answer.text_answer,
            time_spent_seconds: 0,
          },
        },
        {
          onSuccess: (data) => {
            setSaveStatus('saved')
            syncFromServer(data.remaining_seconds)
            scheduleSaveStatusReset(2000)
          },
          onError: () => {
            setSaveStatus('error')
            scheduleSaveStatusReset(3000)
          },
        },
      )
    },
    [saveAnswer, syncFromServer],
  )

  // Immediate save (radio / checkbox / likert)
  const handleOptionAnswer = useCallback(
    (questionId: string, selectedOptionIds: string[]) => {
      const answer: SavedAnswer = {
        selected_option_ids: selectedOptionIds,
        text_answer: null,
        time_spent_seconds: 0,
        saved_at: '',
      }
      setLocalAnswers((prev) => ({ ...prev, [questionId]: answer }))
      triggerSave(questionId, answer)
    },
    [triggerSave],
  )

  // Debounced save (short text, 800ms)
  const handleTextAnswer = useCallback(
    (questionId: string, textAnswer: string) => {
      const answer: SavedAnswer = {
        selected_option_ids: [],
        text_answer: textAnswer,
        time_spent_seconds: 0,
        saved_at: '',
      }
      setLocalAnswers((prev) => ({ ...prev, [questionId]: answer }))
      if (saveDebounceTimerRef.current) clearTimeout(saveDebounceTimerRef.current)
      saveDebounceTimerRef.current = setTimeout(() => triggerSave(questionId, answer), 800)
    },
    [triggerSave],
  )

  const handleAnswer = useCallback(
    (questionId: string, optionIds: string[], textAnswer: string | null) => {
      if (textAnswer !== null) {
        handleTextAnswer(questionId, textAnswer)
      } else {
        handleOptionAnswer(questionId, optionIds)
      }
    },
    [handleTextAnswer, handleOptionAnswer],
  )

  const toggleFlag = useCallback((questionId: string) => {
    setFlags((prev) => {
      const next = new Set(prev)
      if (next.has(questionId)) next.delete(questionId)
      else next.add(questionId)
      return next
    })
  }, [])

  const scrollToQuestion = useCallback((questionId: string) => {
    questionRefs.current[questionId]?.scrollIntoView({ behavior: 'smooth' })
    setShowNavigatorMobile(false)
  }, [])

  // Derived state
  const isAnswered = (qId: string) => {
    const ans = localAnswers[qId]
    if (!ans) return false
    return (
      ans.selected_option_ids.length > 0 ||
      (ans.text_answer !== null && ans.text_answer.trim() !== '')
    )
  }
  const answeredCount = session.questions.filter((q) => isAnswered(q.id)).length

  const handleFinishClick = () => setScreen('review')

  const handleSubmitConfirm = () => {
    setShowSubmitConfirm(false)
    submitSession.mutate(undefined, {
      onSuccess: (result) => {
        onSubmitSuccess(result)
      },
    })
  }

  if (screen === 'review') {
    return (
      <>
        <FinishReviewScreen
          questions={session.questions}
          answeredIds={new Set(session.questions.filter((q) => isAnswered(q.id)).map((q) => q.id))}
          flaggedIds={flags}
          onGoBack={() => setScreen('exam')}
          onSubmitAnyway={() => setShowSubmitConfirm(true)}
        />
        <SubmitConfirmModal
          open={showSubmitConfirm}
          isSubmitting={submitSession.isPending}
          onConfirm={handleSubmitConfirm}
          onCancel={() => setShowSubmitConfirm(false)}
        />
      </>
    )
  }

  return (
    <div className="flex flex-col h-screen bg-background overflow-hidden">
      {/* Top bar */}
      <ExamTopBar
        examTitle={session.exam_title}
        remaining={remaining}
        totalSeconds={totalSeconds}
        answeredCount={answeredCount}
        totalCount={session.questions.length}
        onTimerExpire={handleTimerExpire}
        saveStatus={saveStatus}
        onOpenNavigator={() => setShowNavigatorMobile(true)}
      />

      {/* Main area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Questions */}
        <main id="main-content" tabIndex={-1} className="flex-1 overflow-y-auto p-4 md:p-6 space-y-6 outline-none">
          {session.questions.map((question) => (
            <div
              key={question.id}
              ref={(el) => {
                questionRefs.current[question.id] = el
              }}
            >
              <QuestionDisplay
                question={question}
                savedAnswer={localAnswers[question.id]}
                onAnswer={(optionIds, textAnswer) =>
                  handleAnswer(question.id, optionIds, textAnswer)
                }
                isFlagged={flags.has(question.id)}
                onToggleFlag={() => toggleFlag(question.id)}
              />
            </div>
          ))}

          {/* Finish button */}
          <div className="flex justify-end pt-4 pb-8">
            <Button onClick={handleFinishClick} size="lg">
              {t('exam.taking.finishButton')}
            </Button>
          </div>
        </main>

        {/* Desktop navigator sidebar */}
        <aside className="hidden md:flex flex-col w-56 border-l bg-muted/30 p-4 overflow-y-auto">
          <QuestionNavigator
            questions={session.questions}
            answeredIds={new Set(session.questions.filter((q) => isAnswered(q.id)).map((q) => q.id))}
            flaggedIds={flags}
            onNavigate={scrollToQuestion}
          />
        </aside>
      </div>

      {/* Mobile navigator drawer */}
      {showNavigatorMobile && (
        <div className="fixed inset-0 z-50 flex flex-col justify-end md:hidden">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => setShowNavigatorMobile(false)}
          />
          <div className="relative bg-background rounded-t-2xl p-4 max-h-[60vh] overflow-y-auto">
            <QuestionNavigator
              questions={session.questions}
              answeredIds={
                new Set(session.questions.filter((q) => isAnswered(q.id)).map((q) => q.id))
              }
              flaggedIds={flags}
              onNavigate={scrollToQuestion}
            />
          </div>
        </div>
      )}

      {/* Tab switch warning */}
      <TabSwitchWarningModal open={showTabWarning} onClose={() => setShowTabWarning(false)} />

      {/* Time's up modal */}
      <Dialog open={showTimesUp} onOpenChange={() => {}}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('exam.taking.timesUp.title')}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">{t('exam.taking.timesUp.message')}</p>
          {submitSession.isError && (
            <p className="text-sm text-amber-600">{t('exam.taking.timesUp.retrying')}</p>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
