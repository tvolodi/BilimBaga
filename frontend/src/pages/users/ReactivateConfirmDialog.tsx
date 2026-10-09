import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { userErrorKey } from '@/lib/assignableRoles'
import { useReactivateUser, type User } from '@/api/users'

interface ReactivateConfirmDialogProps {
  user: User | null
  onClose: () => void
}

/** FR-BB18 AC-13: confirm before a deactivated user is reactivated. */
export function ReactivateConfirmDialog({ user, onClose }: ReactivateConfirmDialogProps) {
  const { t } = useTranslation()
  const reactivate = useReactivateUser(user?.id ?? '')
  const [error, setError] = useState<string | null>(null)

  const open = user !== null

  async function handleConfirm() {
    if (!user) return
    setError(null)
    try {
      await reactivate.mutateAsync()
      onClose()
    } catch (err) {
      const key = userErrorKey(err)
      setError(key ? t(key) : err instanceof Error ? err.message : t('users.messages.error_generic'))
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('users.reactivate_dialog.title')}</DialogTitle>
          <DialogDescription>
            {t('users.reactivate_dialog.message', { name: user?.full_name ?? '' })}
          </DialogDescription>
        </DialogHeader>
        {error && <p className="text-sm text-red-600 mb-2">{error}</p>}
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={reactivate.isPending}>
            {t('users.reactivate_dialog.cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={reactivate.isPending}>
            {t('users.reactivate_dialog.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
