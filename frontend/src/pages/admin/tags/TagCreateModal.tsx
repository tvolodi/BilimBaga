import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useCreateTag } from '@/api/tags'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const MAX_TAG_NAME_LENGTH = 64

interface Props {
  open: boolean
  onClose: () => void
}

export function TagCreateModal({ open, onClose }: Props) {
  const { t } = useTranslation()
  const createMutation = useCreateTag()
  const [name, setName] = useState('')
  const [inlineError, setInlineError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName('')
      setInlineError(null)
    }
  }, [open])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setInlineError(null)
    try {
      await createMutation.mutateAsync(name.trim())
      onClose()
    } catch (err: unknown) {
      const e = err as Error & { code?: string }
      if (e.code === 'ERR_TAG_DUPLICATE') {
        setInlineError(t('tags.errors.duplicate'))
      } else if (e.code === 'ERR_INVALID_NAME') {
        setInlineError(t('tags.errors.invalidName'))
      } else {
        setInlineError(e.message)
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('tags.actions.new')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="tag-name">{t('tags.form.name')}</Label>
            <Input
              id="tag-name"
              value={name}
              onChange={(e) => { setName(e.target.value); setInlineError(null) }}
              maxLength={MAX_TAG_NAME_LENGTH}
              placeholder={t('tags.form.nameHelp')}
              autoFocus
            />
            {inlineError && (
              <p className="text-sm text-danger">{inlineError}</p>
            )}
            {!inlineError && (
              <p className="text-xs text-muted-foreground">{t('tags.form.nameHelp')}</p>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!name.trim() || createMutation.isPending}>
              {t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
