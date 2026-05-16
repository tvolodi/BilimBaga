import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts'
import { useTranslation } from 'react-i18next'

interface PassRateChartProps {
  passRate: number
  primaryColor: string
}

export function PassRateChart({ passRate, primaryColor }: PassRateChartProps) {
  const { t } = useTranslation()
  const data = [
    { name: t('exam_analytics.pass_rate'), value: passRate },
    { name: t('exam_analytics.failed'), value: 1 - passRate },
  ]

  const COLORS = [primaryColor || '#6366f1', '#e5e7eb']

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
      <p className="mt-1 text-xl font-bold" style={{ color: primaryColor || '#6366f1' }}>
        {(passRate * 100).toFixed(1)}% {t('exam_analytics.pass_rate')}
      </p>
    </div>
  )
}
