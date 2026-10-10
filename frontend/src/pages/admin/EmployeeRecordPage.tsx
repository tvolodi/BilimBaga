import { useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Loader2, Download } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useUser, useMe } from '@/api/users'
import { downloadErrorKey } from '@/api/download'
import { useEmployeeRecord, useEmployeeProgress, exportEmployeeRecord } from '@/api/employees'
import { fetchLoyaltyNarrative } from '@/api/ai'
import { EmployeeInfoHeader } from '@/components/employees/EmployeeInfoHeader'
import { SessionHistoryTable } from '@/components/employees/SessionHistoryTable'
import { TrackProgressCards } from '@/components/employees/TrackProgressCards'
import { EmployeeRecordSkeleton } from '@/components/employees/EmployeeRecordSkeleton'
import { formatDate } from '@/utils/format'

// ── FR-BB75: Values Profile Section ─────────────────────────────────────────

function ValuesProfileSection({
  sessionId,
  isLoyaltySession,
}: {
  sessionId: string
  isLoyaltySession: boolean
}) {
  const { t, i18n } = useTranslation()
  const [narrative, setNarrative] = useState<string | null>(null)
  const [generatedAt, setGeneratedAt] = useState<string | null>(null)

  const qc = useQueryClient()
  const { mutate, isPending, isError } = useMutation({
    mutationFn: () => fetchLoyaltyNarrative(qc, sessionId),
    onSuccess: (data) => {
      setNarrative(data.narrative)
      setGeneratedAt(data.generated_at)
    },
  })

  if (!isLoyaltySession) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('employee_record.valuesProfileTitle')}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {narrative ? (
          <>
            <p className="text-sm">{narrative}</p>
            <p className="text-xs text-muted-foreground">
              {t('common.aiGenerated')} · {formatDate(generatedAt!, i18n.language)}
            </p>
            <Button variant="outline" size="sm" onClick={() => mutate()} disabled={isPending}>
              {t('employee_record.regenerateNarrative')}
            </Button>
          </>
        ) : (
          <>
            {isError && (
              <p className="text-destructive text-sm">{t('errors.aiUnavailable')}</p>
            )}
            <Button onClick={() => mutate()} disabled={isPending}>
              {isPending
                ? t('employee_record.generating')
                : t('employee_record.generateNarrative')}
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  )
}

// ── Main Page ──────────────────────────────────────────────────────────────────

export function EmployeeRecordPage() {
  const { userId = '' } = useParams<{ userId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [isExporting, setIsExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)

  async function handleExport() {
    setExportError(null)
    setIsExporting(true)
    try {
      await exportEmployeeRecord(qc, userId)
    } catch (err) {
      setExportError(downloadErrorKey(err))
    } finally {
      setIsExporting(false)
    }
  }

  const page = Math.max(1, parseInt(searchParams.get('page') ?? '1', 10))

  const meQuery = useMe()
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
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t('employee_record.history_section')}</h2>
          <Button
            variant="outline"
            size="sm"
            onClick={handleExport}
            disabled={isExporting}
          >
            {isExporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t('common.loading')}
              </>
            ) : (
              <>
                <Download className="mr-2 h-4 w-4" />
                {t('employee_record.export_csv')}
              </>
            )}
          </Button>
        </div>
        {exportError && (
          <div role="alert" className="px-4 py-3 rounded-md text-sm bg-red-50 border border-red-200 text-red-800">
            {t(exportError)}
          </div>
        )}
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

      {/* Values Profile (FR-BB75) — loyalty sessions, admin+ only */}
      {['super_admin', 'hr_admin', 'department_admin'].includes(
        meQuery.data?.role_name ?? '',
      ) &&
        record.data.sessions
          .filter((s) => s.exam_category_track === 'loyalty')
          .map((s) => (
            <section key={s.session_id} className="space-y-3">
              <h2 className="text-lg font-semibold">{t('employee_record.history_section')}</h2>
              <ValuesProfileSection sessionId={s.session_id} isLoyaltySession={true} />
            </section>
          ))}
    </div>
  )
}
