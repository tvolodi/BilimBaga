import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import type { PortalExam } from '@/api/portal'

interface StartExamModalProps {
  exam: PortalExam
  open: boolean
  onClose: () => void
  onConfirm: () => void
  isLoading: boolean
}

export function StartExamModal({
  exam,
  open,
  onClose,
  onConfirm,
  isLoading,
}: StartExamModalProps) {
  const { t } = useTranslation()
  const remaining = exam.max_attempts - exam.attempts_used

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('portal.modal.title')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 text-sm py-2">
          {exam.time_limit_minutes > 0 && (
            <p>{t('portal.modal.timeLimit', { minutes: exam.time_limit_minutes })}</p>
          )}
          <p>{t('portal.modal.maxAttempts', { remaining })}</p>
          <p className="text-amber-600 font-medium">{t('portal.modal.warning')}</p>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            {t('portal.modal.cancel')}
          </Button>
          <Button onClick={onConfirm} disabled={isLoading}>
            {isLoading ? t('common.loading') : t('portal.modal.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
