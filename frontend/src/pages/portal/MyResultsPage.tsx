import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMyResults } from '@/api/sessions'
import { ResultsTable } from '@/components/results/ResultsTable'
import { ResultsTableSkeleton } from '@/components/results/ResultsTableSkeleton'
import { EmptyResults } from '@/components/results/EmptyResults'
import { Button } from '@/components/ui/button'

const PER_PAGE = 20

export function MyResultsPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()

  const page = Math.max(1, parseInt(searchParams.get('page') ?? '1', 10))
  const rawSort = searchParams.get('sort')
  const sort: 'date' | 'score' = rawSort === 'score' ? 'score' : 'date'
  const rawDir = searchParams.get('dir')
  const dir: 'asc' | 'desc' = rawDir === 'asc' ? 'asc' : 'desc'

  const { data, isLoading, isError } = useMyResults(page, sort, dir)

  function handleSort(col: 'date' | 'score') {
    const newDir: 'asc' | 'desc' =
      col === sort ? (dir === 'asc' ? 'desc' : 'asc') : 'desc'
    setSearchParams({ page: '1', sort: col, dir: newDir })
  }

  function goToPage(newPage: number) {
    setSearchParams({ page: String(newPage), sort, dir })
  }

  const total = data?.meta.total ?? 0
  const from = total === 0 ? 0 : (page - 1) * PER_PAGE + 1
  const to = Math.min(page * PER_PAGE, total)
  const totalPages = Math.ceil(total / PER_PAGE)

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">{t('history.title')}</h1>

      {isLoading && <ResultsTableSkeleton />}

      {isError && (
        <p className="text-destructive">{t('common.loadError', 'Failed to load results.')}</p>
      )}

      {!isLoading && !isError && data && data.sessions.length === 0 && <EmptyResults />}

      {!isLoading && !isError && data && data.sessions.length > 0 && (
        <>
          <ResultsTable
            sessions={data.sessions}
            sortCol={sort}
            sortDir={dir}
            onSort={handleSort}
          />

          <div className="flex items-center justify-between pt-2">
            <p className="text-sm text-muted-foreground">
              {t('history.pagination_info', { from, to, total })}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => goToPage(page - 1)}
              >
                {t('common.previous', 'Previous')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => goToPage(page + 1)}
              >
                {t('common.next', 'Next')}
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
