import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useUpdateTag, type Tag } from '@/api/tags'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const MAX_TAG_NAME_LENGTH = 64

interface Props {
  open: boolean
  tag: Tag | null
  onClose: () => void
}

export function TagRenameModal({ open, tag, onClose }: Props) {
  const { t } = useTranslation()
  const updateMutation = useUpdateTag()
  const [name, setName] = useState('')
  const [inlineError, setInlineError] = useState<string | null>(null)

  useEffect(() => {
    if (open && tag) {
      setName(tag.name)
      setInlineError(null)
    }
  }, [open, tag])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!tag) return
    setInlineError(null)
    try {
      await updateMutation.mutateAsync({ id: tag.id, name: name.trim() })
      onClose()
    } catch (err: unknown) {
      const e = err as Error & { code?: string }
      if (e.code === 'ERR_TAG_DUPLICATE') {
        setInlineError(t('tags.errors.duplicate'))
      } else if (e.code === 'ERR_INVALID_NAME') {
        setInlineError(t('tags.errors.invalidName'))
      } else if (e.code === 'ERR_NOT_FOUND') {
        setInlineError(t('tags.errors.notFound'))
      } else {
        setInlineError(e.message)
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('tags.renameModal.title')}</DialogTitle>
          <DialogDescription>{t('tags.renameModal.description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="tag-rename">{t('tags.form.name')}</Label>
            <Input
              id="tag-rename"
              value={name}
              onChange={(e) => { setName(e.target.value); setInlineError(null) }}
              maxLength={MAX_TAG_NAME_LENGTH}
              autoFocus
            />
            {inlineError && (
              <p className="text-sm text-danger">{inlineError}</p>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={!name.trim() || updateMutation.isPending}>
              {t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
