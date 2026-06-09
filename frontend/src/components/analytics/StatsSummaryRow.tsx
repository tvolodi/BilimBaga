import { useTranslation } from 'react-i18next'
import { Card, CardContent } from '@/components/ui/card'

interface StatsSummaryRowProps {
  avgScore: number | null
  medianScore: number | null
  totalAttempts: number
  uniqueParticipants: number
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <CardContent className="p-6">
        <p className="text-sm font-medium text-muted-foreground">{label}</p>
        <p className="mt-2 text-3xl font-bold tracking-tight">{value}</p>
      </CardContent>
    </Card>
  )
}

export function StatsSummaryRow({
  avgScore,
  medianScore,
  totalAttempts,
  uniqueParticipants,
}: StatsSummaryRowProps) {
  const { t } = useTranslation()

  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <StatCard
        label={t('exam_analytics.avg_score')}
        value={avgScore !== null ? `${avgScore.toFixed(1)}%` : '—'}
      />
      <StatCard
        label={t('exam_analytics.median_score')}
        value={medianScore !== null ? `${medianScore.toFixed(1)}%` : '—'}
      />
      <StatCard
        label={t('exam_analytics.total_attempts')}
        value={totalAttempts}
      />
      <StatCard
        label={t('exam_analytics.unique_participants')}
        value={uniqueParticipants}
      />
    </div>
  )
}
