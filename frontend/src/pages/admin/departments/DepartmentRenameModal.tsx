import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useUpdateDepartment, type Department } from '@/api/departments'
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

const MAX_NAME_LENGTH = 100

interface Props {
  open: boolean
  department: Department | null
  onClose: () => void
}

export function DepartmentRenameModal({ open, department, onClose }: Props) {
  const { t } = useTranslation()
  const updateMutation = useUpdateDepartment()
  const [name, setName] = useState('')
  const [inlineError, setInlineError] = useState<string | null>(null)

  useEffect(() => {
    if (open && department) {
      setName(department.name)
      setInlineError(null)
    }
  }, [open, department])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!department) return
    setInlineError(null)
    try {
      await updateMutation.mutateAsync({ id: department.id, name: name.trim() })
      onClose()
    } catch (err: unknown) {
      const e = err as Error & { code?: string }
      if (e.code === 'DUPLICATE_NAME') {
        setInlineError(t('departments.errors.duplicateName'))
      } else if (e.code === 'NOT_FOUND') {
        setInlineError(t('departments.errors.parentNotFound'))
      } else {
        setInlineError(e.message)
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('departments.actions.rename')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="dept-rename">{t('departments.form.name')}</Label>
            <Input
              id="dept-rename"
              value={name}
              onChange={(e) => { setName(e.target.value); setInlineError(null) }}
              maxLength={MAX_NAME_LENGTH}
              autoFocus
            />
            {inlineError && (
              <p className="text-sm text-red-600">{inlineError}</p>
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
