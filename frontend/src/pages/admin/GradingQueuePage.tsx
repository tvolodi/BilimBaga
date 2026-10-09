import React, { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useGradingQueue } from '@/api/grading'
import { useExams } from '@/api/exams'
import { GradingQueueTable } from '@/components/grading/GradingQueueTable'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'

export function GradingQueuePage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [examId, setExamId] = useState('')
  const location = useLocation()
  const navigate = useNavigate()
  // One-shot success notice handed over by GradingDetailPage via navigation state.
  const [showSuccess, setShowSuccess] = useState(
    () => (location.state as { gradingSuccess?: boolean } | null)?.gradingSuccess === true,
  )

  // Clear the navigation state so a refresh / back-forward does not show it again.
  useEffect(() => {
    if ((location.state as { gradingSuccess?: boolean } | null)?.gradingSuccess) {
      navigate(location.pathname + location.search, { replace: true, state: null })
    }
  }, [location, navigate])

  // Auto-dismiss after 5s.
  useEffect(() => {
    if (!showSuccess) return
    const id = setTimeout(() => setShowSuccess(false), 5000)
    return () => clearTimeout(id)
  }, [showSuccess])

  const { data, isLoading, isError } = useGradingQueue(page, examId || undefined)
  const { data: examsData } = useExams({ status: 'active', per_page: 100 })

  function handleExamChange(e: React.ChangeEvent<HTMLSelectElement>) {
    setExamId(e.target.value)
    setPage(1)
  }

  const totalPages = data ? Math.ceil(data.meta.total / data.meta.per_page) : 1
  const start = data && data.items.length > 0 ? (page - 1) * data.meta.per_page + 1 : 0
  const end = data ? Math.min(page * data.meta.per_page, data.meta.total) : 0

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold">{t('grading.queue_title')}</h1>
        <Select
          className="w-64"
          value={examId}
          onChange={handleExamChange}
          aria-label={t('grading.filter_by_exam')}
        >
          <option value="">{t('grading.filter_all_exams')}</option>
          {examsData?.items.map((exam) => (
            <option key={exam.id} value={exam.id}>
              {exam.title}
            </option>
          ))}
        </Select>
      </div>

      {showSuccess && (
        <div
          role="status"
          className="flex items-center justify-between gap-4 px-4 py-3 rounded-md text-sm bg-green-50 border border-green-200 text-green-800"
        >
          <span>{t('grading.success_toast')}</span>
          <button
            type="button"
            onClick={() => setShowSuccess(false)}
            aria-label={t('grading.dismiss_toast')}
            className="font-medium hover:underline"
          >
            &times;
          </button>
        </div>
      )}

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

          <div className="flex items-center justify-between pt-2">
            <span className="text-sm text-muted-foreground">
              {t('grading.showing_range', { start, end, total: data.meta.total })}
            </span>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
              >
                {t('grading.prev_page')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => p + 1)}
                disabled={page >= totalPages}
              >
                {t('grading.next_page')}
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
