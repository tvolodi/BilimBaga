import { useTranslation } from 'react-i18next'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import type { ExamCompletionRate } from '@/api/dashboard'

interface CompletionBarChartProps {
  data: ExamCompletionRate[]
  primaryColor: string
}

interface TooltipPayloadEntry {
  name: string
  value: number
  color: string
}

interface CustomTooltipProps {
  active?: boolean
  payload?: TooltipPayloadEntry[]
  label?: string
  fullTitles: Record<string, string>
}

function CustomTooltip({ active, payload, label, fullTitles }: CustomTooltipProps) {
  if (!active || !payload?.length) return null
  const fullTitle = label ? (fullTitles[label] ?? label) : label
  return (
    <div className="rounded-md border bg-background p-3 text-sm shadow-md">
      <p className="mb-1.5 font-semibold">{fullTitle}</p>
      {payload.map((entry) => (
        <p key={entry.name} style={{ color: entry.color }}>
          {entry.name}: {entry.value}%
        </p>
      ))}
    </div>
  )
}

export function CompletionBarChart({ data, primaryColor }: CompletionBarChartProps) {
  const { t } = useTranslation()

  const chartData = data.map((e) => {
    const completionPct =
      e.assigned_count > 0
        ? parseFloat(((e.completed_count / e.assigned_count) * 100).toFixed(1))
        : 0
    const passPct =
      e.assigned_count > 0
        ? parseFloat(((e.passed_count / e.assigned_count) * 100).toFixed(1))
        : 0
    return {
      name: e.title.length > 20 ? e.title.slice(0, 20) + '…' : e.title,
      fullTitle: e.title,
      assignedCount: e.assigned_count,
      completedCount: e.completed_count,
      [t('dashboard.chart_completed')]: completionPct,
      [t('dashboard.chart_assigned')]: passPct,
    }
  })

  const fullTitles = Object.fromEntries(
    chartData.map((d) => [d.name, d.fullTitle]),
  )

  const completedKey = t('dashboard.chart_completed')
  const assignedKey = t('dashboard.chart_assigned')

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={chartData} margin={{ top: 8, right: 8, left: 0, bottom: 8 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis
          dataKey="name"
          tick={{ fontSize: 12 }}
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          domain={[0, 100]}
          tickFormatter={(v: number) => `${v}%`}
          tick={{ fontSize: 12 }}
          tickLine={false}
          axisLine={false}
        />
        <Tooltip
          content={
            <CustomTooltip fullTitles={fullTitles} />
          }
        />
        <Legend wrapperStyle={{ fontSize: 12 }} />
        <Bar dataKey={assignedKey} fill="#e5e7eb" radius={[4, 4, 0, 0]} />
        <Bar dataKey={completedKey} fill={primaryColor || '#6366f1'} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}
