import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { requestDocumentFullscreen } from '@/lib/fullscreen'

const WARNING_ICON = '⚠️'

interface TabSwitchWarningModalProps {
  open: boolean
  onClose: () => void
  /** Running violation count from the server event_count (FR-BB319 AC-9). Hidden below 1. */
  count: number
  /** True when the candidate was in fullscreen and is not now (FR-BB319 AC-9). */
  canReturnFullscreen: boolean
  /** Called after the fullscreen request is made from the return button; the parent closes the modal. */
  onReturnFullscreen: () => void
}

export function TabSwitchWarningModal({
  open,
  onClose,
  count,
  canReturnFullscreen,
  onReturnFullscreen,
}: TabSwitchWarningModalProps) {
  const { t } = useTranslation()

  function handleReturnFullscreen() {
    requestDocumentFullscreen()
    onReturnFullscreen()
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{WARNING_ICON} {t('exam.taking.tabswitch.title')}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">{t('exam.taking.tabswitch.warning')}</p>
        {count >= 1 && (
          <p className="text-sm font-medium mt-2">{t('exam.taking.tabswitch.count', { count })}</p>
        )}
        <div className="flex flex-wrap justify-end gap-2 mt-4">
          {canReturnFullscreen && (
            <Button variant="outline" className="min-h-11" onClick={handleReturnFullscreen}>
              {t('exam.taking.tabswitch.returnFullscreen')}
            </Button>
          )}
          <Button className="min-h-11" onClick={onClose}>{t('common.cancel')}</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
