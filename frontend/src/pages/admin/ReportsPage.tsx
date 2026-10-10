import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { downloadErrorKey } from '@/api/download'
import { Loader2, Download, BarChart2 } from 'lucide-react'
import { useExamsListForReports, downloadDashboardPdf, downloadExamCsv } from '@/api/reports'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { ExamListItem } from '@/api/exams'

// ---- Status badge -----------------------------------------------------------

function statusVariant(status: ExamListItem['status']): 'default' | 'secondary' | 'outline' {
  if (status === 'active') return 'default'
  if (status === 'archived') return 'secondary'
  return 'outline'
}

// ---- PDF export card --------------------------------------------------------

function DashboardPdfCard() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [exportError, setExportError] = useState<string | null>(null)

  // Default range: last 30 days
  const today = new Date()
  const thirtyDaysAgo = new Date(today)
  thirtyDaysAgo.setDate(today.getDate() - 30)

  const [fromDate, setFromDate] = useState<string>(thirtyDaysAgo.toISOString().slice(0, 10))
  const [toDate, setToDate] = useState<string>(today.toISOString().slice(0, 10))
  const [isExporting, setIsExporting] = useState(false)

  async function handleExport() {
    if (!fromDate || !toDate) return
    setIsExporting(true)
    setExportError(null)
    try {
      await downloadDashboardPdf(qc, fromDate, toDate)
    } catch (err) {
      setExportError(downloadErrorKey(err))
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('reports.pdf_export_title')}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex flex-col gap-1">
            <label htmlFor="pdf-from" className="text-sm font-medium text-muted-foreground">
              {t('reports.pdf_date_from')}
            </label>
            <input
              id="pdf-from"
              type="date"
              value={fromDate}
              max={toDate}
              onChange={(e) => setFromDate(e.target.value)}
              className="rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="pdf-to" className="text-sm font-medium text-muted-foreground">
              {t('reports.pdf_date_to')}
            </label>
            <input
              id="pdf-to"
              type="date"
              value={toDate}
              min={fromDate}
              onChange={(e) => setToDate(e.target.value)}
              className="rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <Button
            onClick={handleExport}
            disabled={isExporting || !fromDate || !toDate}
            className="sm:self-end"
          >
            {isExporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t('reports.pdf_exporting')}
              </>
            ) : (
              <>
                <Download className="mr-2 h-4 w-4" />
                {t('reports.pdf_export_btn')}
              </>
            )}
          </Button>
        </div>
        {exportError && (
          <p role="alert" className="text-sm text-red-700">
            {t(exportError)}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

// ---- Exam reports table card ------------------------------------------------

function ExamReportsCard() {
  const { t } = useTranslation()
  const { data: exams, isLoading } = useExamsListForReports()
  const qc = useQueryClient()
  const [exportingId, setExportingId] = useState<string | null>(null)
  const [exportError, setExportError] = useState<string | null>(null)

  async function handleExportCsv(exam: ExamListItem) {
    if (exportingId) return
    setExportingId(exam.id)
    setExportError(null)
    try {
      await downloadExamCsv(qc, exam.id, exam.title)
    } catch (err) {
      setExportError(downloadErrorKey(err))
    } finally {
      setExportingId(null)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('reports.exam_reports_title')}</CardTitle>
      </CardHeader>
      <CardContent>
        {exportError && (
          <p role="alert" className="text-sm text-red-700">
            {t(exportError)}
          </p>
        )}
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-10 w-full rounded-md" />
            ))}
          </div>
        ) : !exams || exams.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('reports.no_exams')}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-muted-foreground">
                  <th className="pb-2 pr-4 font-medium">{t('reports.col_exam')}</th>
                  <th className="pb-2 pr-4 font-medium">{t('reports.col_status')}</th>
                  <th className="pb-2 font-medium">{t('reports.col_actions')}</th>
                </tr>
              </thead>
              <tbody>
                {exams.map((exam) => (
                  <tr key={exam.id} className="border-b last:border-0">
                    <td className="py-2 pr-4 font-medium">{exam.title}</td>
                    <td className="py-2 pr-4">
                      <Badge variant={statusVariant(exam.status)}>{t(`exam_list.status.${exam.status}`)}</Badge>
                    </td>
                    <td className="py-2">
                      <div className="flex items-center gap-2">
                        <Button variant="ghost" size="sm" asChild>
                          <Link to={`/admin/exams/${exam.id}/analytics`}>
                            <BarChart2 className="mr-1 h-4 w-4" />
                            {t('reports.view_analytics')}
                          </Link>
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={exportingId === exam.id}
                          onClick={() => handleExportCsv(exam)}
                        >
                          {exportingId === exam.id ? (
                            <>
                              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                              {t('reports.exporting_csv')}
                            </>
                          ) : (
                            <>
                              <Download className="mr-1 h-4 w-4" />
                              {t('reports.export_csv')}
                            </>
                          )}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---- Page -------------------------------------------------------------------

export function ReportsPage() {
  const { t } = useTranslation()

  return (
    <div className="space-y-6 p-6">
      <h1 className="text-2xl font-bold">{t('reports.title')}</h1>
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <DashboardPdfCard />
        <ExamReportsCard />
      </div>
    </div>
  )
}
