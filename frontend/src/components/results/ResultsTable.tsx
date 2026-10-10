import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { downloadFile, downloadErrorKey } from '@/api/download'
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
import type { SessionHistoryItem } from '@/api/sessions'
import { formatDuration } from '@/utils/format'

interface ResultsTableProps {
  sessions: SessionHistoryItem[]
  sortCol: 'date' | 'score'
  sortDir: 'asc' | 'desc'
  onSort: (col: 'date' | 'score') => void
}

function SortIndicator({ active, dir }: { active: boolean; dir: 'asc' | 'desc' }) {
  if (!active) return <span className="ml-1 text-muted-foreground">↕</span>
  return <span className="ml-1">{dir === 'asc' ? '↑' : '↓'}</span>
}

export function ResultsTable({ sessions, sortCol, sortDir, onSort }: ResultsTableProps) {
  const { t } = useTranslation()

  const qc = useQueryClient()
  const [downloadError, setDownloadError] = useState<string | null>(null)

  async function handleCertificate(sessionId: string) {
    setDownloadError(null)
    try {
      await downloadFile(
        qc,
        `/api/v1/portal/sessions/${sessionId}/certificate`,
        `certificate-${sessionId}.pdf`,
      )
    } catch (err) {
      setDownloadError(downloadErrorKey(err))
    }
  }

  return (
    <div className="space-y-2">
      {downloadError && (
        <p role="alert" className="text-sm text-danger">
          {t(downloadError)}
        </p>
      )}
      <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('history.col_exam')}</TableHead>
            <TableHead>
              <button
                type="button"
                className="flex items-center font-semibold hover:text-foreground"
                onClick={() => onSort('date')}
              >
                {t('history.col_date')}
                <SortIndicator active={sortCol === 'date'} dir={sortDir} />
              </button>
            </TableHead>
            <TableHead>
              <button
                type="button"
                className="flex items-center font-semibold hover:text-foreground"
                onClick={() => onSort('score')}
              >
                {t('history.col_score')}
                <SortIndicator active={sortCol === 'score'} dir={sortDir} />
              </button>
            </TableHead>
            <TableHead>{t('history.col_status')}</TableHead>
            <TableHead>{t('history.col_time')}</TableHead>
            <TableHead>{t('history.col_certificate')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sessions.map((s) => (
            <TableRow key={s.session_id}>
              <TableCell>
                <Link
                  to={`/portal/sessions/${s.session_id}/result`}
                  className="font-medium hover:underline"
                >
                  {s.exam_title}
                </Link>
              </TableCell>
              <TableCell>
                {new Date(s.submitted_at).toLocaleDateString()}
              </TableCell>
              <TableCell>{s.score_pct.toFixed(1)}%</TableCell>
              <TableCell>
                {s.passed ? (
                  <Badge variant="success">{t('history.status.passed')}</Badge>
                ) : (
                  <Badge variant="destructive">{t('history.status.failed')}</Badge>
                )}
              </TableCell>
              <TableCell>
                {s.time_taken_seconds != null
                  ? formatDuration(s.time_taken_seconds)
                  : '—'}
              </TableCell>
              <TableCell>
                {s.certificate_available ? (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleCertificate(s.session_id)}
                  >
                    {t('history.download_certificate')}
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
