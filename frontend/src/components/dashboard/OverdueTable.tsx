import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDistanceToNow } from 'date-fns'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { useRemindEmployee } from '@/api/dashboard'
import type { OverdueEmployee } from '@/api/dashboard'

interface ToastState {
  message: string
  type: 'success' | 'error'
}

interface OverdueTableProps {
  employees: OverdueEmployee[]
}

export function OverdueTable({ employees }: OverdueTableProps) {
  const { t } = useTranslation()
  const [toast, setToast] = useState<ToastState | null>(null)
  const [sentIds, setSentIds] = useState<Set<string>>(new Set())

  function showToast(message: string, type: 'success' | 'error') {
    setToast({ message, type })
    setTimeout(() => setToast(null), 5000)
  }

  const remind = useRemindEmployee(
    () => showToast(t('dashboard.reminder_sent'), 'success'),
    () => showToast(t('dashboard.reminder_error'), 'error'),
  )

  function handleRemind(employee: OverdueEmployee) {
    const examId = employee.exam_id ?? ''
    remind.mutate(
      { userId: employee.user_id, examId },
      {
        onSuccess: () => {
          setSentIds((prev) => new Set(prev).add(employee.user_id + examId))
        },
      },
    )
  }

  if (employees.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-muted-foreground">
        {t('dashboard.overdue_empty')}
      </p>
    )
  }

  return (
    <div className="space-y-2">
      {toast && (
        <div
          className={`rounded-md px-4 py-2 text-sm ${
            toast.type === 'success'
              ? 'bg-green-50 text-green-800'
              : 'bg-red-50 text-red-800'
          }`}
        >
          {toast.message}
        </div>
      )}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('dashboard.col_employee')}</TableHead>
            <TableHead>{t('dashboard.col_exam')}</TableHead>
            <TableHead>{t('dashboard.col_deadline')}</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {employees.map((emp) => {
            const key = emp.user_id + (emp.exam_id ?? '')
            const alreadySent = sentIds.has(key)
            return (
              <TableRow key={key}>
                <TableCell className="font-medium">{emp.name}</TableCell>
                <TableCell>{emp.exam_title}</TableCell>
                <TableCell className="text-muted-foreground">
                  {formatDistanceToNow(new Date(emp.deadline), { addSuffix: true })}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={remind.isPending || alreadySent}
                    onClick={() => handleRemind(emp)}
                  >
                    {t('dashboard.send_reminder')}
                  </Button>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
