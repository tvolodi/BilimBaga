import { useTranslation } from 'react-i18next'
import { type Department } from '@/api/departments'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface Props {
  open: boolean
  department: Department | null
  isPending: boolean
  onClose: () => void
  onConfirm: () => void
}

export function DepartmentDeleteConfirm({ open, department, isPending, onClose, onConfirm }: Props) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('departments.deleteConfirm.title')}</DialogTitle>
          <DialogDescription>
            {t('departments.deleteConfirm.description')}
            {department && (
              <span className="block mt-1 font-medium text-foreground">{department.name}</span>
            )}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            {t('common.cancel')}
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? '…' : t('departments.actions.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
