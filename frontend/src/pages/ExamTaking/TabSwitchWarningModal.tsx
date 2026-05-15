import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface TabSwitchWarningModalProps {
  open: boolean
  onClose: () => void
}

export function TabSwitchWarningModal({ open, onClose }: TabSwitchWarningModalProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>⚠️ {t('exam.taking.tabswitch.title')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">{t('exam.taking.tabswitch.warning')}</p>
        <div className="flex justify-end mt-4">
          <Button onClick={onClose}>{t('common.cancel')}</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
