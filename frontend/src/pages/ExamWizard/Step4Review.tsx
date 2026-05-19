import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, AlertTriangle, CheckCircle2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { useExam, usePublishExam, useEligibleCounts, type PublishValidationDetail, type ExamApiError } from '@/api/exams'

// ---- Types ------------------------------------------------------------------

interface Step4ReviewProps {
  examId: string
  onBack: () => void
  onPublished: () => void
}

// ---- Review row -------------------------------------------------------------

function ReviewRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between py-2 border-b last:border-0">
      <span className="text-sm text-muted-foreground w-40 shrink-0">{label}</span>
      <span className="text-sm font-medium text-right">{value}</span>
    </div>
  )
}

// ---- Component --------------------------------------------------------------

export function Step4Review({ examId, onBack, onPublished }: Step4ReviewProps) {
  const { t } = useTranslation()
  const { data: exam, isLoading } = useExam(examId)
  const publishExam = usePublishExam(examId)
  const { data: eligibleData, isLoading: eligibleLoading, isError: eligibleError } = useEligibleCounts(examId)

  const [confirmOpen, setConfirmOpen] = useState(false)
  const [successBanner, setSuccessBanner] = useState(false)
  const [unsatisfiedRules, setUnsatisfiedRules] = useState<PublishValidationDetail[] | null>(null)
  const [generalError, setGeneralError] = useState<string | null>(null)

  async function handlePublishConfirm() {
    setUnsatisfiedRules(null)
    setGeneralError(null)
    try {
      await publishExam.mutateAsync()
      setConfirmOpen(false)
      setSuccessBanner(true)
      setTimeout(onPublished, 1500)
    } catch (err) {
      setConfirmOpen(false)
      const apiErr = err as ExamApiError
      if (apiErr.httpStatus === 422 && apiErr.unsatisfiedRules) {
        setUnsatisfiedRules(apiErr.unsatisfiedRules)
      } else {
        setGeneralError(apiErr.message)
      }
    }
  }

  if (isLoading || !exam) {
    return (
      <div className="flex justify-center py-16">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  function formatDateTime(iso: string | null) {
    if (!iso) return '—'
    return new Date(iso).toLocaleString()
  }

  const isPublished = exam.status === 'active'

  return (
    <div className="space-y-6">
      <h2 className="text-base font-medium">{t('exam.wizard.step4.title')}</h2>

      {/* Success */}
      {successBanner && (
        <div className="flex items-center gap-2 rounded-md bg-green-50 border border-green-200 px-4 py-3 text-sm text-green-800">
          <CheckCircle2 size={16} />
          {t('exam.publishSuccess')}
        </div>
      )}

      {/* 422 validation error */}
      {unsatisfiedRules && unsatisfiedRules.length > 0 && (
        <div className="rounded-md bg-amber-50 border border-amber-200 px-4 py-3 text-sm text-amber-800 space-y-2">
          <div className="flex items-center gap-2 font-medium">
            <AlertTriangle size={16} />
            {t('exam.wizard.validationWarning')}
          </div>
          <ul className="list-disc list-inside space-y-1">
            {unsatisfiedRules.map((r) => (
              <li key={r.rule_id}>
                Rule {r.rule_id.slice(0, 8)}…: needs {r.required}, only {r.available} available
              </li>
            ))}
          </ul>
          {unsatisfiedRules.some((r) => r.available === 0) && (
            <p className="text-xs mt-1 opacity-80">{t('exam.wizard.validationWarningHint')}</p>
          )}
          <button
            type="button"
            className="text-xs underline mt-1 text-amber-700 hover:text-amber-900"
            onClick={() => setUnsatisfiedRules(null)}
          >
            {t('exam.wizard.retryPublish')}
          </button>
        </div>
      )}

      {/* General error */}
      {generalError && (
        <div className="rounded-md bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive">
          {generalError}
        </div>
      )}

      {/* Summary */}
      <div className="rounded-lg border p-4">
        <h3 className="text-sm font-semibold mb-3">{t('exam.wizard.step1.title')}</h3>
        <ReviewRow label={t('exam.field.title')} value={exam.title} />
        <ReviewRow label={t('exam.field.description')} value={exam.description ?? '—'} />
        <ReviewRow label={t('exam.field.timeLimitMinutes')} value={`${exam.time_limit_minutes} min`} />
        <ReviewRow label={t('exam.field.passingScorePct')} value={`${exam.passing_score_pct}%`} />
        <ReviewRow label={t('exam.field.maxAttempts')} value={exam.max_attempts} />
        <ReviewRow label={t('exam.field.availableFrom')} value={formatDateTime(exam.available_from)} />
        <ReviewRow label={t('exam.field.availableUntil')} value={formatDateTime(exam.available_until)} />
        <ReviewRow
          label={t('exam.field.shuffleQuestions')}
          value={exam.shuffle_questions ? '✓' : '—'}
        />
        <ReviewRow
          label={t('exam.field.shuffleOptions')}
          value={exam.shuffle_options ? '✓' : '—'}
        />
        <ReviewRow
          label={t('exam.field.showAnswers')}
          value={t(`exam.showAnswers.${exam.show_answers}`)}
        />
        <ReviewRow
          label={t('exam.field.onTabSwitch')}
          value={t(`exam.onTabSwitch.${exam.on_tab_switch}`)}
        />
        <ReviewRow
          label={t('exam.field.certificateEnabled')}
          value={exam.certificate_enabled ? '✓' : '—'}
        />
      </div>

      <div className="rounded-lg border p-4">
        <h3 className="text-sm font-semibold mb-3">{t('exam.wizard.step2.title')}</h3>
        {exam.rules.length === 0 ? (
          <p className="text-sm text-muted-foreground">—</p>
        ) : (
          <ul className="space-y-1">
            {exam.rules.map((r, i) => {
              const countEntry = eligibleData?.counts.find((c) => c.rule_id === r.id)
              const eligible = countEntry?.eligible
              let badge: React.ReactNode = null
              if (eligibleLoading) {
                badge = <Loader2 className="inline h-3 w-3 animate-spin ml-2 text-muted-foreground" />
              } else if (eligibleError) {
                badge = (
                  <span className="inline-flex items-center gap-1 ml-2 text-xs text-destructive">
                    <AlertTriangle className="h-3 w-3" />
                    {t('exam.wizard.step4.eligibleCountError')}
                  </span>
                )
              } else if (eligible !== undefined) {
                const badgeClass =
                  eligible >= r.count
                    ? 'bg-green-100 text-green-800'
                    : eligible > 0
                      ? 'bg-amber-100 text-amber-800'
                      : 'bg-red-100 text-red-800'
                badge = (
                  <Badge className={`ml-2 text-xs ${badgeClass}`}>
                    {eligible} {t('exam.wizard.step4.eligibleCount')}
                  </Badge>
                )
              }
              return (
                <li key={r.id} className="text-sm flex items-center">
                  <span>
                    Rule {i + 1}: {r.mode}, {r.count} questions
                    {r.difficulty && `, ${r.difficulty}`}
                  </span>
                  {badge}
                </li>
              )
            })}
          </ul>
        )}
      </div>

      {/* Navigation */}
      <div className="flex justify-between pt-4">
        <Button type="button" variant="outline" onClick={onBack}>
          {t('exam.wizard.back')}
        </Button>
        {!isPublished && (
          <Button type="button" onClick={() => setConfirmOpen(true)} disabled={publishExam.isPending || !!unsatisfiedRules}>
            {publishExam.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t('exam.wizard.publishing')}
              </>
            ) : (
              t('exam.wizard.publish')
            )}
          </Button>
        )}
        {isPublished && (
          <div className="flex items-center gap-2 text-sm text-green-700">
            <CheckCircle2 size={16} />
            Published
          </div>
        )}
      </div>

      {/* Confirm dialog */}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('exam.wizard.publish')}</DialogTitle>
            <DialogDescription>
              {t('exam.wizard.step4.publishConfirm')}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setConfirmOpen(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={handlePublishConfirm} disabled={publishExam.isPending}>
              {publishExam.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {t('exam.wizard.publishing')}
                </>
              ) : (
                t('exam.wizard.publish')
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
