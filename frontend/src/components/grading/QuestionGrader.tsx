import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import type { GradingQuestion } from '@/api/grading'

interface QuestionGraderProps {
  question: GradingQuestion
  score: number | null
  feedback: string
  onChange: (score: number | null, feedback: string) => void
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

export function QuestionGrader({ question, score, feedback, onChange }: QuestionGraderProps) {
  const { t } = useTranslation()

  const isInvalid = score !== null && (score < 0 || score > 100)

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
        {question.grading_status === 'ai_graded' && question.ai_reasoning && (
          <AIReasoningSection reasoning={question.ai_reasoning} />
        )}
      </div>

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
            className="flex-1 h-2 accent-primary cursor-pointer"
          />
          <Input
            type="number"
            min={0}
            max={100}
            value={score ?? ''}
            onChange={handleInputChange}
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
          placeholder={t('grading.feedback_placeholder')}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm resize-none min-h-[80px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>
    </div>
  )
}
