import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'

interface PasswordResetModalProps {
  open: boolean
  temporaryPassword: string | null
  onClose: () => void
}

export function PasswordResetModal({ open, temporaryPassword, onClose }: PasswordResetModalProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  async function handleCopy() {
    if (!temporaryPassword) return
    await navigator.clipboard.writeText(temporaryPassword)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('users.reset_password_modal.title')}</DialogTitle>
          <DialogDescription>{t('users.reset_password_modal.message')}</DialogDescription>
        </DialogHeader>
        <div className="space-y-3 my-4">
          <p className="text-sm font-medium">{t('users.reset_password_modal.temp_password_label')}</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-muted rounded px-3 py-2 text-sm font-mono">
              {temporaryPassword}
            </code>
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? '✓' : t('users.actions.edit')}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">{t('users.reset_password_modal.note')}</p>
        </div>
        <DialogFooter>
          <Button onClick={onClose}>{t('users.reset_password_modal.close')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
