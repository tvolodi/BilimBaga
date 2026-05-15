import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useGradingQueue } from '@/api/grading'
import { GradingQueueTable } from '@/components/grading/GradingQueueTable'
import { Button } from '@/components/ui/button'

export function GradingQueuePage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useGradingQueue(page)

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">{t('grading.queue_title')}</h1>

      {isLoading && (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      )}

      {isError && (
        <p className="text-sm text-destructive">{t('common.loadError')}</p>
      )}

      {data && data.items.length === 0 && (
        <p className="text-sm text-muted-foreground">{t('grading.queue_empty')}</p>
      )}

      {data && data.items.length > 0 && (
        <>
          <GradingQueueTable sessions={data.items} />

          {data.meta.total > data.meta.per_page && (
            <div className="flex items-center justify-between pt-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
              >
                &larr;
              </Button>
              <span className="text-sm text-muted-foreground">
                {page} / {Math.ceil(data.meta.total / data.meta.per_page)}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => p + 1)}
                disabled={page >= Math.ceil(data.meta.total / data.meta.per_page)}
              >
                &rarr;
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
