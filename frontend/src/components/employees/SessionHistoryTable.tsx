import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatDuration } from '@/utils/format'
import type { SessionRecord } from '@/api/employees'
import { useQueryClient } from '@tanstack/react-query'
import { downloadAdminCertificate } from '@/api/employees'
import { downloadErrorKey } from '@/api/download'

interface SessionHistoryTableProps {
  sessions: SessionRecord[]
}

export function SessionHistoryTable({ sessions }: SessionHistoryTableProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [downloadError, setDownloadError] = useState<string | null>(null)

  async function handleDownload(session: SessionRecord) {
    if (!session.certificate_id) return
    setDownloadError(null)
    try {
      await downloadAdminCertificate(qc, session.session_id, session.certificate_id)
    } catch (err) {
      setDownloadError(downloadErrorKey(err))
      setTimeout(() => setDownloadError(null), 5000)
    }
  }

  if (sessions.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4">{t('employee_record.history_empty')}</p>
    )
  }

  return (
    <div className="space-y-2">
      {downloadError && (
        <div role="alert" className="px-4 py-3 rounded-md text-sm bg-red-50 border border-red-200 text-red-800">
          {t(downloadError)}
        </div>
      )}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('employee_record.col_exam')}</TableHead>
              <TableHead>{t('employee_record.col_date')}</TableHead>
              <TableHead>{t('employee_record.col_score')}</TableHead>
              <TableHead>{t('employee_record.col_status')}</TableHead>
              <TableHead>{t('employee_record.col_time')}</TableHead>
              <TableHead>{t('employee_record.col_certificate')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sessions.map((s) => (
              <TableRow key={s.session_id}>
                <TableCell className="font-medium">{s.exam_title}</TableCell>
                <TableCell>
                  {s.submitted_at
                    ? new Date(s.submitted_at).toLocaleDateString()
                    : s.started_at
                      ? new Date(s.started_at).toLocaleDateString()
                      : '—'}
                </TableCell>
                <TableCell>
                  {s.score_pct != null ? `${s.score_pct.toFixed(1)}%` : '—'}
                </TableCell>
                <TableCell>
                  {s.passed === true ? (
                    <Badge variant="success">{t('employee_record.status_passed')}</Badge>
                  ) : s.passed === false ? (
                    <Badge variant="destructive">{t('employee_record.status_failed')}</Badge>
                  ) : (
                    <Badge variant="outline">{t('employee_record.status_grading_pending')}</Badge>
                  )}
                </TableCell>
                <TableCell>
                  {s.time_taken_seconds != null ? formatDuration(s.time_taken_seconds) : '—'}
                </TableCell>
                <TableCell>
                  {s.certificate_id != null ? (
                    <Button size="sm" variant="outline" onClick={() => handleDownload(s)}>
                      {t('employee_record.download_certificate')}
                    </Button>
                  ) : (
                    '—'
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
