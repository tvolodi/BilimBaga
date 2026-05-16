import { useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useUser } from '@/api/users'
import { useEmployeeRecord, useEmployeeProgress } from '@/api/employees'
import { EmployeeInfoHeader } from '@/components/employees/EmployeeInfoHeader'
import { SessionHistoryTable } from '@/components/employees/SessionHistoryTable'
import { TrackProgressCards } from '@/components/employees/TrackProgressCards'
import { EmployeeRecordSkeleton } from '@/components/employees/EmployeeRecordSkeleton'

export function EmployeeRecordPage() {
  const { userId = '' } = useParams<{ userId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation()

  const page = Math.max(1, parseInt(searchParams.get('page') ?? '1', 10))

  const userQuery = useUser(userId)
  const recordQuery = useEmployeeRecord(userId, page)
  const progressQuery = useEmployeeProgress(userId)

  const isLoading =
    (!userQuery.data && userQuery.isLoading) ||
    (!recordQuery.data && recordQuery.isLoading) ||
    (!progressQuery.data && progressQuery.isLoading)

  const isError = userQuery.isError || recordQuery.isError || progressQuery.isError

  function handleRetry() {
    if (userQuery.isError) void userQuery.refetch()
    if (recordQuery.isError) void recordQuery.refetch()
    if (progressQuery.isError) void progressQuery.refetch()
  }

  if (isLoading) {
    return (
      <div className="max-w-6xl">
        <EmployeeRecordSkeleton />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex flex-col items-start gap-4 p-6">
        <p className="text-destructive">{t('employee_record.load_error')}</p>
        <Button variant="outline" onClick={handleRetry}>
          {t('employee_record.retry')}
        </Button>
      </div>
    )
  }

  const user = userQuery.data!
  const record = recordQuery.data!
  const progress = progressQuery.data!

  const total = record.meta.total
  const perPage = record.meta.per_page
  const from = total === 0 ? 0 : (page - 1) * perPage + 1
  const to = Math.min(page * perPage, total)
  const totalPages = Math.max(1, Math.ceil(total / perPage))

  function setPage(p: number) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('page', String(p))
      return next
    })
  }

  return (
    <div className="space-y-8 max-w-6xl">
      <h1 className="text-2xl font-bold">{t('employee_record.title')}</h1>

      {/* Employee information header */}
      <section aria-label={t('employee_record.info_section')}>
        <EmployeeInfoHeader user={user} />
      </section>

      {/* Session history */}
      <section className="space-y-3" aria-label={t('employee_record.history_section')}>
        <h2 className="text-lg font-semibold">{t('employee_record.history_section')}</h2>
        <SessionHistoryTable sessions={record.data.sessions} />

        {total > 0 && (
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>{t('employee_record.showing', { from, to, total })}</span>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                {t('users.pagination.previous')}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={page >= totalPages}
                onClick={() => setPage(page + 1)}
              >
                {t('users.pagination.next')}
              </Button>
            </div>
          </div>
        )}
      </section>

      {/* Track progress */}
      <section className="space-y-3" aria-label={t('employee_record.progress_section')}>
        <h2 className="text-lg font-semibold">{t('employee_record.progress_section')}</h2>
        <TrackProgressCards tracks={progress.tracks} />
      </section>
    </div>
  )
}
