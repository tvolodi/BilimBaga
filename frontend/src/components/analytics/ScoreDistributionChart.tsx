import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from 'recharts'
import { useTranslation } from 'react-i18next'
import { useThemeColors } from '@/hooks/useThemeColors'
import type { BucketCount } from '@/api/analytics'

interface ScoreDistributionChartProps {
  distribution: BucketCount[]
  /** Overrides the resolved primary when given (FR-BB321 AC-10). */
  primaryColor?: string
}

export function ScoreDistributionChart({
  distribution,
  primaryColor,
}: ScoreDistributionChartProps) {
  const colours = useThemeColors()
  const { t } = useTranslation()
  return (
    <div role="img" aria-label={t('exam_analytics.score_distribution')}>
      <ResponsiveContainer width="100%" height={240}>
        <BarChart data={distribution} margin={{ top: 8, right: 8, left: 0, bottom: 8 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={colours.border} vertical={false} />
          <XAxis dataKey="bucket" tick={{ fontSize: 11, fill: colours.textSecondary }} tickLine={false} axisLine={false} />
          <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: colours.textSecondary }} tickLine={false} axisLine={false} />
          <Tooltip />
          <Bar dataKey="count" fill={primaryColor || colours.primary} radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
