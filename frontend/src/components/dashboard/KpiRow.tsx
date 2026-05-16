import { useTranslation } from 'react-i18next'
import { Users, BookOpen, CheckCircle2, TrendingUp } from 'lucide-react'
import { KpiCard } from './KpiCard'
import { computeKpis } from '@/api/dashboard'
import type { DashboardMetrics } from '@/api/dashboard'

interface KpiRowProps {
  data: DashboardMetrics
  employeeCount: number
  activeExamCount: number
}

export function KpiRow({ data, employeeCount, activeExamCount }: KpiRowProps) {
  const { t } = useTranslation()
  const { completionRate, passRate } = computeKpis(data)
  const last30Days = t('dashboard.kpi_last_30_days')

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <KpiCard
        icon={<Users size={20} />}
        label={t('dashboard.kpi_employees')}
        value={employeeCount}
      />
      <KpiCard
        icon={<BookOpen size={20} />}
        label={t('dashboard.kpi_active_exams')}
        value={activeExamCount}
      />
      <KpiCard
        icon={<CheckCircle2 size={20} />}
        label={t('dashboard.kpi_completion_rate')}
        value={completionRate}
        description={last30Days}
      />
      <KpiCard
        icon={<TrendingUp size={20} />}
        label={t('dashboard.kpi_pass_rate')}
        value={passRate}
        description={last30Days}
      />
    </div>
  )
}
