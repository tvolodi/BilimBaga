import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from 'recharts'
import type { BucketCount } from '@/api/analytics'

interface ScoreDistributionChartProps {
  distribution: BucketCount[]
  primaryColor: string
}

export function ScoreDistributionChart({
  distribution,
  primaryColor,
}: ScoreDistributionChartProps) {
  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={distribution} margin={{ top: 8, right: 8, left: 0, bottom: 8 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis dataKey="bucket" tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
        <YAxis allowDecimals={false} tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
        <Tooltip />
        <Bar dataKey="count" fill={primaryColor || '#6366f1'} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}
