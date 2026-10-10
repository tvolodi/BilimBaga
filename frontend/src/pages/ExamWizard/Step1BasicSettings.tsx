import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { DateTimePicker } from '@/components/ui/date-time-picker'
import {
  useCreateExam,
  useUpdateExam,
  useAddSection,
  type ExamDetail,
  type CreateExamPayload,
  type ExamApiError,
} from '@/api/exams'

// ---- Types ------------------------------------------------------------------

interface FormState {
  title: string
  description: string
  timeLimitMinutes: string
  passingScorePct: string
  maxAttempts: string
  availableFrom: string | null
  availableUntil: string | null
  shuffleQuestions: boolean
  shuffleOptions: boolean
  showAnswers: 'never' | 'after_completion' | 'after_all_attempts'
  onTabSwitch: 'log' | 'warn' | 'submit'
  certificateEnabled: boolean
}

interface ValidationErrors {
  title?: string
  timeLimitMinutes?: string
  passingScorePct?: string
  maxAttempts?: string
  availableUntil?: string
}

// ---- Toggle Switch component ------------------------------------------------

function Switch({
  checked,
  onChange,
  id,
  disabled,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  id?: string
  disabled?: boolean
}) {
  return (
    <button
      id={id}
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => !disabled && onChange(!checked)}
      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
        checked ? 'bg-primary' : 'bg-input'
      }`}
    >
      <span
        className={`inline-block h-4 w-4 transform rounded-full bg-background transition-transform ${
          checked ? 'translate-x-6' : 'translate-x-1'
        }`}
      />
    </button>
  )
}

// ---- Helpers ----------------------------------------------------------------

function formFromExam(exam: ExamDetail | null): FormState {
  if (!exam) {
    return {
      title: '',
      description: '',
      timeLimitMinutes: '60',
      passingScorePct: '70',
      maxAttempts: '1',
      availableFrom: null,
      availableUntil: null,
      shuffleQuestions: false,
      shuffleOptions: false,
      showAnswers: 'never',
      onTabSwitch: 'log',
      certificateEnabled: false,
    }
  }
  return {
    title: exam.title,
    description: exam.description ?? '',
    timeLimitMinutes: String(exam.time_limit_minutes),
    passingScorePct: String(exam.passing_score_pct),
    maxAttempts: String(exam.max_attempts),
    availableFrom: exam.available_from,
    availableUntil: exam.available_until,
    shuffleQuestions: exam.shuffle_questions,
    shuffleOptions: exam.shuffle_options,
    showAnswers: exam.show_answers,
    onTabSwitch: exam.on_tab_switch,
    certificateEnabled: exam.certificate_enabled,
  }
}

function validateForm(form: FormState): ValidationErrors {
  const errors: ValidationErrors = {}
  if (!form.title.trim()) {
    errors.title = 'required'
  }
  const tl = Number(form.timeLimitMinutes)
  if (!form.timeLimitMinutes || isNaN(tl) || tl < 1 || tl > 300) {
    errors.timeLimitMinutes = 'range'
  }
  const ps = Number(form.passingScorePct)
  if (!form.passingScorePct || isNaN(ps) || ps < 0 || ps > 100) {
    errors.passingScorePct = 'range'
  }
  const ma = Number(form.maxAttempts)
  if (!form.maxAttempts || isNaN(ma) || ma < 1 || ma > 10) {
    errors.maxAttempts = 'range'
  }
  if (form.availableFrom && form.availableUntil) {
    if (new Date(form.availableFrom) >= new Date(form.availableUntil)) {
      errors.availableUntil = 'order'
    }
  }
  return errors
}

// ---- Component --------------------------------------------------------------

interface Step1BasicSettingsProps {
  exam: ExamDetail | null
  onDone: (examId: string, sectionId: string | null) => void
  resolvedSectionId: string | null
}

export function Step1BasicSettings({ exam, onDone, resolvedSectionId }: Step1BasicSettingsProps) {
  const { t } = useTranslation()
  const [form, setForm] = useState<FormState>(() => formFromExam(exam))
  const [errors, setErrors] = useState<ValidationErrors>({})
  const [apiError, setApiError] = useState<string | null>(null)

  const isReadOnly = exam?.status === 'active'

  useEffect(() => {
    setForm(formFromExam(exam))
    // Intentionally keyed on the exam id only: re-sync the form when a different exam loads,
    // not on every refetch of the same exam (which would discard unsaved edits).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [exam?.id])

  const createExam = useCreateExam()
  const updateExam = useUpdateExam(exam?.id ?? '')
  const addSection = useAddSection()

  const isPending = createExam.isPending || updateExam.isPending

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
    setErrors((prev) => {
      const next = { ...prev }
      delete next[key as keyof ValidationErrors]
      return next
    })
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setApiError(null)

    // Active exams are read-only: skip save and navigate directly to next step
    if (isReadOnly && exam) {
      onDone(exam.id, resolvedSectionId)
      return
    }

    const validationErrors = validateForm(form)
    if (Object.keys(validationErrors).length > 0) {
      setErrors(validationErrors)
      return
    }

    const payload: CreateExamPayload = {
      title: form.title.trim(),
      description: form.description.trim() || null,
      time_limit_minutes: Number(form.timeLimitMinutes),
      passing_score_pct: Number(form.passingScorePct),
      max_attempts: Number(form.maxAttempts),
      available_from: form.availableFrom || null,
      available_until: form.availableUntil || null,
      shuffle_questions: form.shuffleQuestions,
      shuffle_options: form.shuffleOptions,
      show_answers: form.showAnswers,
      on_tab_switch: form.onTabSwitch,
      certificate_enabled: form.certificateEnabled,
    }

    try {
      if (exam) {
        await updateExam.mutateAsync(payload)
        onDone(exam.id, resolvedSectionId)
      } else {
        const created = await createExam.mutateAsync(payload)
        // Create a default section for rules
        let sectionId: string | null = null
        try {
          const section = await addSection.mutateAsync({ examId: created.id, sort_order: 0 })
          sectionId = section.id
        } catch {
          // Section creation is best-effort; rules can exist without a section
          sectionId = null
        }
        onDone(created.id, sectionId)
      }
    } catch (err) {
      const apiErr = err as ExamApiError
      setApiError(apiErr.message)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {isReadOnly && (
        <div className="rounded-md bg-bg-warning border border-warning px-4 py-3 text-sm text-warning">
          {t('exam.wizard.activeReadOnlyNotice')}
        </div>
      )}

      {apiError && (
        <div className="rounded-md bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive">
          {apiError}
        </div>
      )}

      {/* Title */}
      <div className="space-y-1.5">
        <Label htmlFor="title">{t('exam.field.title')} *</Label>
        <Input
          id="title"
          value={form.title}
          onChange={(e) => set('title', e.target.value)}
          placeholder={t('exam.field.title')}
          className={errors.title ? 'border-destructive' : ''}
          disabled={isReadOnly}
        />
        {errors.title && (
          <p className="text-xs text-destructive">{t('exam.field.title')} {t('exam.error.required')}</p>
        )}
      </div>

      {/* Description */}
      <div className="space-y-1.5">
        <Label htmlFor="description">{t('exam.field.description')}</Label>
        <textarea
          id="description"
          value={form.description}
          onChange={(e) => set('description', e.target.value)}
          rows={3}
          disabled={isReadOnly}
          className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-none"
        />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {/* Time limit */}
        <div className="space-y-1.5">
          <Label htmlFor="timeLimitMinutes">{t('exam.field.timeLimitMinutes')}</Label>
          <Input
            id="timeLimitMinutes"
            type="number"
            min={1}
            max={300}
            value={form.timeLimitMinutes}
            onChange={(e) => set('timeLimitMinutes', e.target.value)}
            className={errors.timeLimitMinutes ? 'border-destructive' : ''}
            disabled={isReadOnly}
          />
          {errors.timeLimitMinutes && (
            <p className="text-xs text-destructive">{t('exam.error.range', { min: 1, max: 300 })}</p>
          )}
        </div>

        {/* Passing score */}
        <div className="space-y-1.5">
          <Label htmlFor="passingScorePct">{t('exam.field.passingScorePct')}</Label>
          <Input
            id="passingScorePct"
            type="number"
            min={0}
            max={100}
            value={form.passingScorePct}
            onChange={(e) => set('passingScorePct', e.target.value)}
            className={errors.passingScorePct ? 'border-destructive' : ''}
            disabled={isReadOnly}
          />
          {errors.passingScorePct && (
            <p className="text-xs text-destructive">{t('exam.error.range', { min: 0, max: 100 })}</p>
          )}
        </div>

        {/* Max attempts */}
        <div className="space-y-1.5">
          <Label htmlFor="maxAttempts">{t('exam.field.maxAttempts')}</Label>
          <Input
            id="maxAttempts"
            type="number"
            min={1}
            max={10}
            value={form.maxAttempts}
            onChange={(e) => set('maxAttempts', e.target.value)}
            className={errors.maxAttempts ? 'border-destructive' : ''}
            disabled={isReadOnly}
          />
          {errors.maxAttempts && (
            <p className="text-xs text-destructive">{t('exam.error.range', { min: 1, max: 10 })}</p>
          )}
        </div>
      </div>

      {/* Availability window */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <DateTimePicker
          id="availableFrom"
          label={t('exam.field.availableFrom')}
          value={form.availableFrom}
          onChange={(v) => set('availableFrom', v)}
          disabled={isReadOnly}
        />
        <DateTimePicker
          id="availableUntil"
          label={t('exam.field.availableUntil')}
          value={form.availableUntil}
          onChange={(v) => set('availableUntil', v)}
          disabled={isReadOnly}
        />
      </div>
      {errors.availableUntil && (
        <p className="text-xs text-destructive -mt-4">{t('exam.error.availableUntilOrder')}</p>
      )}

      {/* Selects: show_answers, on_tab_switch */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label htmlFor="showAnswers">{t('exam.field.showAnswers')}</Label>
          <Select
            id="showAnswers"
            value={form.showAnswers}
            onChange={(e) => set('showAnswers', e.target.value as FormState['showAnswers'])}
            disabled={isReadOnly}
          >
            <option value="never">{t('exam.showAnswers.never')}</option>
            <option value="after_completion">{t('exam.showAnswers.after_completion')}</option>
            <option value="after_all_attempts">{t('exam.showAnswers.after_all_attempts')}</option>
          </Select>
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="onTabSwitch">{t('exam.field.onTabSwitch')}</Label>
          <Select
            id="onTabSwitch"
            value={form.onTabSwitch}
            onChange={(e) => set('onTabSwitch', e.target.value as FormState['onTabSwitch'])}
            disabled={isReadOnly}
          >
            <option value="log">{t('exam.onTabSwitch.log')}</option>
            <option value="warn">{t('exam.onTabSwitch.warn')}</option>
            <option value="submit">{t('exam.onTabSwitch.submit')}</option>
          </Select>
        </div>
      </div>

      {/* Toggles */}
      <div className="space-y-4">
        {(
          [
            { key: 'shuffleQuestions', labelKey: 'exam.field.shuffleQuestions' },
            { key: 'shuffleOptions', labelKey: 'exam.field.shuffleOptions' },
            { key: 'certificateEnabled', labelKey: 'exam.field.certificateEnabled' },
          ] as const
        ).map(({ key, labelKey }) => (
          <div key={key} className="flex items-center justify-between">
            <Label htmlFor={key}>{t(labelKey)}</Label>
            <Switch
              id={key}
              checked={form[key] as boolean}
              onChange={(v) => set(key, v)}
              disabled={isReadOnly}
            />
          </div>
        ))}
      </div>

      {/* Navigation */}
      <div className="flex justify-end pt-4">
        <Button type="submit" disabled={isPending}>
          {isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              {t('exam.wizard.saving')}
            </>
          ) : (
            t('exam.wizard.next')
          )}
        </Button>
      </div>
    </form>
  )
}
