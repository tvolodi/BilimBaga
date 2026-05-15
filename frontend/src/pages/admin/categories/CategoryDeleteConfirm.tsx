import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface CategoryDeleteConfirmProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  isPending: boolean
}

export function CategoryDeleteConfirm({
  open,
  onClose,
  onConfirm,
  isPending,
}: CategoryDeleteConfirmProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('categories.deleteConfirm.title')}</DialogTitle>
          <DialogDescription>{t('categories.deleteConfirm.description')}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            {t('common.cancel')}
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? '…' : t('categories.actions.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
