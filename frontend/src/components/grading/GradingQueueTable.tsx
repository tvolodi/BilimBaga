import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import type { GradingQueueItem } from '@/api/grading'

interface GradingQueueTableProps {
  sessions: GradingQueueItem[]
}

export function GradingQueueTable({ sessions }: GradingQueueTableProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('grading.col_session')}</TableHead>
          <TableHead>{t('grading.col_employee')}</TableHead>
          <TableHead>{t('grading.col_exam')}</TableHead>
          <TableHead>{t('grading.col_submitted')}</TableHead>
          <TableHead>{t('grading.col_pending')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sessions.map((item) => (
          <TableRow
            key={item.session_id}
            className="cursor-pointer"
            onClick={() => navigate(`/admin/grading/${item.session_id}`)}
          >
            <TableCell className="font-mono text-xs">
              {item.session_id.slice(0, 8)}
            </TableCell>
            <TableCell>{item.employee_name}</TableCell>
            <TableCell>{item.exam_title}</TableCell>
            <TableCell>{new Date(item.submitted_at).toLocaleString()}</TableCell>
            <TableCell>
              <Badge variant="outline">{item.pending_question_count}</Badge>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
