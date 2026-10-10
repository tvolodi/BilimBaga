import { useCallback, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Loader2, Download } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { downloadErrorKey } from '@/api/download'
import { useAuditLog, exportAuditLog, type AuditFilters } from '@/api/audit'
import { AuditFilterBar } from '@/components/audit/AuditFilterBar'
import { AuditLogTable } from '@/components/audit/AuditLogTable'
import { Button } from '@/components/ui/button'

// Default "last 7 days" to avoid loading entire audit history on first load.
function defaultFrom(): string {
  const d = new Date()
  d.setDate(d.getDate() - 7)
  return d.toISOString()
}

function filtersFromParams(params: URLSearchParams): AuditFilters {
  const actions = params.getAll('action')
  return {
    from: params.get('from') ?? undefined,
    to: params.get('to') ?? undefined,
    actor: params.get('actor') ?? undefined,
    actions: actions.length ? actions : undefined,
    entityType: params.get('entity_type') ?? undefined,
  }
}

function filtersToParams(filters: AuditFilters): URLSearchParams {
  const p = new URLSearchParams()
  if (filters.from) p.set('from', filters.from)
  if (filters.to) p.set('to', filters.to)
  if (filters.actor) p.set('actor', filters.actor)
  if (filters.actions?.length) filters.actions.forEach((a) => p.append('action', a))
  if (filters.entityType) p.set('entity_type', filters.entityType)
  return p
}

export function AuditLogPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const [isExporting, setIsExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)
  const fallbackFrom = useMemo(() => defaultFrom(), [])

  // Initialise filters from URL; default to "last 7 days" when no date range set.
  const rawFilters = filtersFromParams(searchParams)
  const filters: AuditFilters = {
    ...rawFilters,
    from: rawFilters.from ?? fallbackFrom,
  }

  const page = parseInt(searchParams.get('page') ?? '1', 10) || 1

  const { data, isLoading, isError } = useAuditLog(filters, page)

  const handleFiltersChange = useCallback(
    (next: AuditFilters) => {
      // Changing any filter resets to page 1 (AC-9).
      const p = filtersToParams(next)
      p.delete('page')
      setSearchParams(p, { replace: true })
    },
    [setSearchParams],
  )

  function setPage(p: number) {
    const params = filtersToParams(filters)
    params.set('page', String(p))
    setSearchParams(params, { replace: true })
  }

  async function handleExport() {
    setExportError(null)
    setIsExporting(true)
    try {
      await exportAuditLog(qc, filters)
    } catch (err) {
      setExportError(downloadErrorKey(err))
    } finally {
      setIsExporting(false)
    }
  }

  const total = data?.meta.total ?? 0
  const perPage = data?.meta.per_page ?? 50
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  const fromEntry = total === 0 ? 0 : (page - 1) * perPage + 1
  const toEntry = Math.min(page * perPage, total)

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{t('audit.title')}</h1>
        <Button
          variant="outline"
          size="sm"
          onClick={handleExport}
          disabled={isExporting}
        >
          {isExporting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              {t('audit.exporting')}
            </>
          ) : (
            <>
              <Download className="mr-2 h-4 w-4" />
              {t('audit.export_csv')}
            </>
          )}
        </Button>
      </div>

      {exportError && (
        <div role="alert" className="px-4 py-3 rounded-md text-sm bg-bg-danger border border-danger text-danger">
          {t(exportError)}
        </div>
      )}

      {/* Filter bar */}
      <AuditFilterBar filters={filters} onChange={handleFiltersChange} />

      {/* Loading / error states */}
      {isLoading && (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      )}
      {isError && (
        <p className="text-sm text-destructive">{t('common.loadError')}</p>
      )}

      {/* Table */}
      {data && (
        <>
          <AuditLogTable entries={data.items} />

          {/* Pagination */}
          <div className="flex items-center justify-between pt-2">
            <span className="text-sm text-muted-foreground">
              {t('audit.pagination_info', { from: fromEntry, to: toEntry, total })}
            </span>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(page - 1)}
                disabled={page <= 1}
              >
                &larr;
              </Button>
              <span className="text-sm text-muted-foreground">
                {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(page + 1)}
                disabled={page >= totalPages}
              >
                &rarr;
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
