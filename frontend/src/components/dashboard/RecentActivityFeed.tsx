import { useTranslation } from 'react-i18next'
import { formatDistanceToNow } from 'date-fns'
import { Badge } from '@/components/ui/badge'
import type { RecentActivity } from '@/api/dashboard'

interface RecentActivityFeedProps {
  activities: RecentActivity[]
}

function initials(name: string): string {
  return name
    .split(' ')
    .map((w) => w[0] ?? '')
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

function ScoreBadge({
  score,
  passed,
}: {
  score: number | null
  passed: boolean | null
}) {
  if (passed === null || score === null) {
    return (
      <Badge variant="secondary" className="text-xs">
        —
      </Badge>
    )
  }
  return (
    <Badge
      variant={passed ? 'success' : 'destructive'}
      className="text-xs"
    >
      {score.toFixed(0)}%
    </Badge>
  )
}

export function RecentActivityFeed({ activities }: RecentActivityFeedProps) {
  const { t } = useTranslation()

  if (activities.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-muted-foreground">
        {t('dashboard.activity_empty')}
      </p>
    )
  }

  return (
    <ul className="max-h-96 divide-y overflow-y-auto">
      {activities.slice(0, 20).map((item) => (
        <li key={item.session_id} className="flex items-center gap-3 py-3 px-1">
          <div
            aria-hidden
            className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary"
          >
            {initials(item.employee_name)}
          </div>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{item.employee_name}</p>
            <p className="truncate text-xs text-muted-foreground">{item.exam_title}</p>
          </div>
          <ScoreBadge score={item.score_pct} passed={item.passed} />
          <span className="flex-shrink-0 text-xs text-muted-foreground">
            {formatDistanceToNow(new Date(item.submitted_at), { addSuffix: true })}
          </span>
        </li>
      ))}
    </ul>
  )
}
