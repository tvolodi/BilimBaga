import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus, BarChart2, Pencil } from 'lucide-react'
import { useExams } from '@/api/exams'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

function StatusBadge({ status }: { status: 'draft' | 'active' | 'archived' }) {
  const colors = {
    draft: 'bg-gray-100 text-gray-700',
    active: 'bg-green-100 text-green-700',
    archived: 'bg-red-100 text-red-700',
  }
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${colors[status]}`}
    >
      {status}
    </span>
  )
}

export function ExamsListPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useExams({ page, per_page: 20 })

  const totalPages = data ? Math.ceil(data.meta.total / data.meta.per_page) : 1

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{t('nav.exams')}</h1>
        <Link to="/admin/exams/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            {t('exam.wizard.createTitle')}
          </Button>
        </Link>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-14 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && (
        <p className="text-sm text-destructive">{t('common.loadError')}</p>
      )}

      {/* Table */}
      {data && (
        <div className="rounded-lg border overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  {t('exam.field.title')}
                </th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  {t('exam_list.col_status')}
                </th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  {t('exam.field.timeLimitMinutes')}
                </th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  {t('exam.field.passingScorePct')}
                </th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  {t('exam_list.col_actions')}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.items.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                    {t('common.coming_soon')}
                  </td>
                </tr>
              )}
              {data.items.map((exam) => (
                <tr key={exam.id} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3 font-medium">{exam.title}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={exam.status} />
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {t('exam_list.minutes_format', { minutes: exam.time_limit_minutes })}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {exam.passing_score_pct}%
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Link to={`/admin/exams/${exam.id}/edit`}>
                        <Button variant="ghost" size="sm" aria-label={t('questionBank.actions.edit')}>
                          <Pencil size={14} />
                        </Button>
                      </Link>
                      <Link to={`/admin/exams/${exam.id}/analytics`}>
                        <Button variant="outline" size="sm">
                          <BarChart2 size={14} className="mr-1" />
                          {t('exam_analytics.title')}
                        </Button>
                      </Link>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {data && totalPages > 1 && (
        <div className="flex items-center justify-between">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
          >
            {t('users.pagination.previous')}
          </Button>
          <span className="text-sm text-muted-foreground">
            {t('users.pagination.page_info', { page, total: totalPages })}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            {t('users.pagination.next')}
          </Button>
        </div>
      )}
    </div>
  )
}
