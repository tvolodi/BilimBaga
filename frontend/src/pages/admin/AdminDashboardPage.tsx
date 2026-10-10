import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { useDashboard } from '@/api/dashboard'
import { KpiRow } from '@/components/dashboard/KpiRow'
import { CompletionBarChart } from '@/components/dashboard/CompletionBarChart'
import { OverdueTable } from '@/components/dashboard/OverdueTable'
import { RecentActivityFeed } from '@/components/dashboard/RecentActivityFeed'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

const ADMIN_ROLES = ['examiner', 'hr_admin', 'super_admin']
export { ADMIN_ROLES }

export function AdminDashboardPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data, isLoading, isError } = useDashboard()

  function handleRefresh() {
    qc.invalidateQueries({ queryKey: ['dashboard'] })
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{t('dashboard.title')}</h1>
        <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading}>
          <RefreshCw size={16} className={isLoading ? 'animate-spin mr-2' : 'mr-2'} />
          {t('dashboard.refresh')}
        </Button>
      </div>

      {/* Loading / error */}
      {isLoading && (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      )}
      {isError && (
        <p className="text-sm text-destructive">{t('common.loadError')}</p>
      )}

      {data && (
        <>
          {/* KPI row — employee/active-exam counts derived from data where possible */}
          <KpiRow
            data={data}
            employeeCount={
              data.completion_rate_by_exam.reduce((s, e) => s + e.assigned_count, 0)
            }
            activeExamCount={data.completion_rate_by_exam.length}
          />

          {/* Completion bar chart */}
          {data.completion_rate_by_exam.length > 0 && (
            <Card>
              <CardHeader>
                <h2 className="text-base font-semibold">{t('dashboard.chart_title')}</h2>
              </CardHeader>
              <CardContent>
                <CompletionBarChart
                  data={data.completion_rate_by_exam}
                />
              </CardContent>
            </Card>
          )}

          {/* Overdue employees & recent activity — side by side on large screens */}
          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <h2 className="text-base font-semibold">{t('dashboard.overdue_title')}</h2>
              </CardHeader>
              <CardContent className="pt-0">
                <OverdueTable employees={data.overdue_employees} />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <h2 className="text-base font-semibold">{t('dashboard.activity_title')}</h2>
              </CardHeader>
              <CardContent className="pt-0">
                <RecentActivityFeed activities={data.recent_activity} />
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  )
}
