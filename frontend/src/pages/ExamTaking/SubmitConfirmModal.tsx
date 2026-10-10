import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface SubmitConfirmModalProps {
  open: boolean
  isSubmitting: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function SubmitConfirmModal({
  open,
  isSubmitting,
  onConfirm,
  onCancel,
}: SubmitConfirmModalProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onCancel()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('exam.taking.submit.confirm.title')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">{t('exam.taking.submit.confirm.message')}</p>
        <div className="flex justify-end gap-3 mt-4">
          <Button variant="outline" className="min-h-11" onClick={onCancel} disabled={isSubmitting}>
            {t('common.cancel')}
          </Button>
          <Button className="min-h-11" onClick={onConfirm} disabled={isSubmitting}>
            {isSubmitting ? t('common.loading') : t('exam.taking.submit.confirm.submit')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
