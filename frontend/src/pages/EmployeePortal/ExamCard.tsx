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

export function ExamCard({ exam, onStart }: ExamCardProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const countdown = useCountdown(exam.deadline)

  const statusVariant = STATUS_VARIANT[exam.user_status]
  const statusLabel = t(`portal.card.status.${exam.user_status}`)

  function handleCta() {
    switch (exam.user_status) {
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
    switch (exam.user_status) {
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

  const ctaDisabled = exam.user_status === 'failed' && exam.show_answers === 'never'
  const showCta = exam.user_status !== 'expired' && ctaLabel !== null

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-semibold text-base leading-tight">{exam.title}</h3>
          <Badge variant={statusVariant}>{statusLabel}</Badge>
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
        {showCta && (
          <Button
            className="w-full"
            variant={exam.user_status === 'not_started' ? 'default' : 'outline'}
            disabled={ctaDisabled}
            onClick={handleCta}
          >
            {ctaLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
