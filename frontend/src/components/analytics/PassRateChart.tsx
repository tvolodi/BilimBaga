import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts'
import { useTranslation } from 'react-i18next'
import { useThemeColors } from '@/hooks/useThemeColors'

interface PassRateChartProps {
  passRate: number
  /** Overrides the resolved primary when given. Charts follow the theme by default (FR-BB321 AC-10). */
  primaryColor?: string
}

export function PassRateChart({ passRate, primaryColor }: PassRateChartProps) {
  const { t } = useTranslation()
  const colours = useThemeColors()
  const primary = primaryColor || colours.primary
  const data = [
    { name: t('exam_analytics.pass_rate'), value: passRate },
    { name: t('exam_analytics.failed'), value: 1 - passRate },
  ]

  const COLORS = [primary, colours.border]

  return (
    <div
      role="img"
      aria-label={`${t('exam_analytics.pass_rate')}: ${(passRate * 100).toFixed(1)}%`}
      className="flex flex-col items-center"
    >
      <ResponsiveContainer width={200} height={200}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={55}
            outerRadius={80}
            dataKey="value"
            startAngle={90}
            endAngle={-270}
          >
            {data.map((_entry, index) => (
              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip formatter={(value) => typeof value === 'number' ? `${(value * 100).toFixed(1)}%` : value} />
        </PieChart>
      </ResponsiveContainer>
      <p className="mt-1 text-xl font-bold" style={{ color: primary }}>
        {(passRate * 100).toFixed(1)}% {t('exam_analytics.pass_rate')}
      </p>
    </div>
  )
}
