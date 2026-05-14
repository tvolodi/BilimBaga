import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { useDeactivateUser, type User } from '@/api/users'

interface DeactivateConfirmDialogProps {
  user: User | null
  onClose: () => void
}

export function DeactivateConfirmDialog({ user, onClose }: DeactivateConfirmDialogProps) {
  const { t } = useTranslation()
  const deactivate = useDeactivateUser(user?.id ?? '')
  const [error, setError] = useState<string | null>(null)

  const open = user !== null

  async function handleConfirm() {
    if (!user) return
    setError(null)
    try {
      await deactivate.mutateAsync()
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('users.messages.error_generic'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('users.deactivate_dialog.title')}</DialogTitle>
          <DialogDescription>
            {t('users.deactivate_dialog.message', { name: user?.full_name ?? '' })}
          </DialogDescription>
        </DialogHeader>
        {error && <p className="text-sm text-red-600 mb-2">{error}</p>}
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={deactivate.isPending}>
            {t('users.deactivate_dialog.cancel')}
          </Button>
          <Button variant="destructive" onClick={handleConfirm} disabled={deactivate.isPending}>
            {t('users.deactivate_dialog.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
