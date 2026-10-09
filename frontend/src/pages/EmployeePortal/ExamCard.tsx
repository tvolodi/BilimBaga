import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useCountdown } from '@/hooks/useCountdown'
import type { PortalExam, UserStatus } from '@/api/portal'

type BadgeVariant = 'secondary' | 'default' | 'success' | 'destructive' | 'warning'

const STATUS_VARIANT: Record<UserStatus, BadgeVariant> = {
  not_started: 'secondary',
  in_progress: 'default',
  passed: 'success',
  failed: 'destructive',
  expired: 'warning',
}

interface ExamCardProps {
  exam: PortalExam
  onStart: (examId: string) => void
}

/** Returns why the exam cannot be started right now (ISS-132), or null when its window is open. */
export function startWindowBlock(exam: PortalExam, now: Date = new Date()): 'notOpen' | 'closed' | null {
  if (exam.available_from && now < new Date(exam.available_from)) return 'notOpen'
  if (exam.available_until && now > new Date(exam.available_until)) return 'closed'
  return null
}

export function ExamCard({ exam, onStart }: ExamCardProps) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const countdown = useCountdown(exam.deadline)

  // ISS-038: when an active session exists, always show in-progress state regardless of user_status
  const displayStatus: UserStatus = exam.open_session_id ? 'in_progress' : exam.user_status

  const statusVariant = STATUS_VARIANT[displayStatus]
  const statusLabel = t(`portal.card.status.${displayStatus}`)
  const noAttemptsRemaining = displayStatus === 'failed' && exam.attempts_used >= exam.max_attempts

  function handleCta() {
    switch (displayStatus) {
      case 'not_started':
        onStart(exam.id)
        break
      case 'in_progress':
        navigate(`/portal/sessions/${exam.open_session_id}`)
        break
      case 'passed':
        navigate(`/portal/exams/${exam.id}/result`)
        break
      case 'failed':
        navigate(`/portal/exams/${exam.id}/result`)
        break
    }
  }

  const ctaLabel = (() => {
    switch (displayStatus) {
      case 'not_started':
        return t('portal.card.cta.start')
      case 'in_progress':
        return t('portal.card.cta.continue')
      case 'passed':
      case 'failed':
        return t('portal.card.cta.viewResult')
      default:
        return null
    }
  })()

  const windowBlock = displayStatus === 'not_started' ? startWindowBlock(exam) : null
  const ctaDisabled = (displayStatus === 'failed' && exam.show_answers === 'never') || windowBlock !== null
  const showCta = displayStatus !== 'expired' && ctaLabel !== null && !noAttemptsRemaining

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-semibold text-base leading-tight">{exam.title}</h3>
          <div className="flex flex-col items-end gap-1">
            <Badge variant={statusVariant}>{statusLabel}</Badge>
            {noAttemptsRemaining && (
              <span className="text-xs text-muted-foreground">{t('portal.card.noAttemptsRemaining')}</span>
            )}
          </div>
        </div>
        {exam.description && (
          <p className="text-sm text-muted-foreground line-clamp-2 mt-1">{exam.description}</p>
        )}
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-x-2 gap-y-1 text-xs text-muted-foreground mb-3">
          <span>{t('portal.card.timeLimit', { minutes: exam.time_limit_minutes })}</span>
          <span>·</span>
          <span>{t('portal.card.passingScore', { pct: exam.passing_score_pct })}</span>
          <span>·</span>
          <span>
            {t('portal.card.attempts', { used: exam.attempts_used, max: exam.max_attempts })}
          </span>
        </div>
        <div className="text-xs text-muted-foreground mb-4">
          {exam.deadline
            ? t('portal.card.deadline', { countdown })
            : t('portal.card.noDeadline')}
        </div>
        {windowBlock && (
          <p id={`start-reason-${exam.id}`} role="status" className="text-xs text-amber-800 dark:text-amber-400 mb-2">
            {windowBlock === 'notOpen'
              ? t('portal.card.windowNotOpen', {
                  date: new Date(exam.available_from as string).toLocaleString(i18n.language),
                })
              : t('portal.card.windowClosed')}
          </p>
        )}
        {showCta && (
          <Button
            className="w-full"
            variant={displayStatus === 'not_started' ? 'default' : 'outline'}
            disabled={ctaDisabled}
            aria-describedby={windowBlock ? `start-reason-${exam.id}` : undefined}
            onClick={handleCta}
          >
            {ctaLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
