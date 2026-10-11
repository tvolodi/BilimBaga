import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { GradingQuestion } from '@/api/grading'

interface QuestionGraderProps {
  question: GradingQuestion
  score: number | null
  feedback: string
  onChange: (score: number | null, feedback: string) => void
  /** A graded answer of a final session is shown, not edited (FR-BB47 AC-5, FR-BB73 AC-19). */
  readOnly?: boolean
  /** Posts the suggested value as the grade (FR-BB73 AC-17). Shown only when the answer has a suggestion. */
  onAccept?: () => void
  isAccepting?: boolean
}

function AIReasoningSection({ reasoning }: { reasoning: string }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <div className="mt-2">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="text-xs text-info hover:underline focus:outline-none"
      >
        {open ? t('grading.ai_reasoning_hide') : t('grading.ai_reasoning_show')}
      </button>
      {open && (
        <div className="mt-1 rounded border border-info bg-bg-info px-3 py-2 text-sm text-info">
          <p className="font-medium mb-1">{t('grading.ai_reasoning_label')}</p>
          <p className="whitespace-pre-wrap">{reasoning}</p>
        </div>
      )}
    </div>
  )
}

/** The AI suggestion of a waiting answer. Score, flags and reasoning are text only: React escapes them (FR-BB73 AC-18). */
function SuggestionPanel({
  question,
  suggestedPct,
  onAccept,
  isAccepting,
}: {
  question: GradingQuestion
  suggestedPct: number
  onAccept?: () => void
  isAccepting: boolean
}) {
  const { t } = useTranslation()
  return (
    <section className="rounded border border-info bg-bg-info px-3 py-3 space-y-2 text-sm text-info">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="font-medium">{t('grading.suggestion.title')}</p>
        <p>{t('grading.suggestion.score', { score: suggestedPct })}</p>
      </div>
      {question.ai_flags && question.ai_flags.length > 0 && (
        <ul className="flex flex-wrap gap-1.5" aria-label={t('grading.suggestion.flags_label')}>
          {question.ai_flags.map((flag) => (
            <li key={flag}>
              <Badge variant="outline" className="border-info text-info">
                {t(`grading.flag.${flag}`)}
              </Badge>
            </li>
          ))}
        </ul>
      )}
      {question.ai_reasoning && (
        <div>
          <p className="font-medium mb-1">{t('grading.ai_reasoning_label')}</p>
          <p className="whitespace-pre-wrap">{question.ai_reasoning}</p>
        </div>
      )}
      <div className="flex flex-wrap items-center gap-3">
        {onAccept && (
          <Button type="button" size="sm" onClick={onAccept} disabled={isAccepting}>
            {isAccepting ? t('grading.suggestion.accepting') : t('grading.suggestion.accept')}
          </Button>
        )}
        <p className="text-xs">{t('grading.suggestion.waiting_note')}</p>
      </div>
    </section>
  )
}

export function QuestionGrader({
  question,
  score,
  feedback,
  onChange,
  readOnly = false,
  onAccept,
  isAccepting = false,
}: QuestionGraderProps) {
  const { t } = useTranslation()

  const isInvalid = score !== null && (score < 0 || score > 100)
  const suggestedPct = question.grading_status === 'pending_manual' ? question.ai_suggested_score_pct : null

  function handleSliderChange(e: React.ChangeEvent<HTMLInputElement>) {
    onChange(Number(e.target.value), feedback)
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const val = e.target.value
    if (val === '') {
      onChange(null, feedback)
      return
    }
    const num = Number(val)
    onChange(isNaN(num) ? null : num, feedback)
  }

  function handleFeedbackChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    onChange(score, e.target.value)
  }

  return (
    <div className="space-y-5">
      {/* Question stem */}
      <div>
        <p className="text-sm font-medium text-muted-foreground mb-1">{t('grading.question_indicator', { current: '', total: '' })}</p>
        <p className="text-base font-medium">{question.stem}</p>
        {question.grading_status === 'graded' && (
          <Badge variant="secondary" className="mt-1">
            {t('grading.already_graded')}
          </Badge>
        )}
        {question.grading_status === 'ai_graded' && (
          <Badge variant="secondary" className="mt-1 bg-bg-info text-info">
            {t('grading.ai_graded')}
          </Badge>
        )}
        {question.grading_status === 'ai_graded' && question.current_score_pct !== null && (
          <p className="mt-1 text-sm text-info">{t('grading.suggestion.ai_score', { score: question.current_score_pct })}</p>
        )}
        {question.grading_status === 'ai_graded' && question.ai_reasoning && (
          <AIReasoningSection reasoning={question.ai_reasoning} />
        )}
      </div>

      {/* AI suggestion of a waiting answer (FR-BB73 AC-18) */}
      {suggestedPct !== null && (
        <SuggestionPanel
          question={question}
          suggestedPct={suggestedPct}
          onAccept={readOnly ? undefined : onAccept}
          isAccepting={isAccepting}
        />
      )}

      {/* Employee answer */}
      <div>
        <label className="block text-sm font-medium mb-1">{t('grading.answer_label')}</label>
        <textarea
          readOnly
          value={question.text_answer}
          className="w-full rounded-md border bg-muted px-3 py-2 text-sm resize-none min-h-[80px]"
        />
      </div>

      {/* Score */}
      <div>
        <label className="block text-sm font-medium mb-2">{t('grading.score_label')}</label>
        <div className="flex items-center gap-4">
          <input
            type="range"
            min={0}
            max={100}
            step={1}
            value={score ?? 0}
            onChange={handleSliderChange}
            disabled={readOnly}
            className="flex-1 h-2 accent-primary cursor-pointer disabled:cursor-not-allowed disabled:opacity-50"
          />
          <Input
            type="number"
            min={0}
            max={100}
            value={score ?? ''}
            onChange={handleInputChange}
            disabled={readOnly}
            className={`w-20 ${isInvalid ? 'border-destructive focus-visible:ring-destructive' : ''}`}
          />
        </div>
        {isInvalid && (
          <p className="text-destructive text-xs mt-1">{t('grading.score_error')}</p>
        )}
      </div>

      {/* Feedback */}
      <div>
        <label className="block text-sm font-medium mb-1">{t('grading.feedback_label')}</label>
        <textarea
          value={feedback}
          onChange={handleFeedbackChange}
          readOnly={readOnly}
          placeholder={t('grading.feedback_placeholder')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm resize-none min-h-[80px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring read-only:bg-muted"
        />
      </div>
    </div>
  )
}
