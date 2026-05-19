import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useCreateDepartment, type Department } from '@/api/departments'
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
  defaultParentId?: string | null
  departments: Department[]
  onClose: (errorMessage?: string) => void
}

function flattenDepartments(nodes: Department[], result: Department[] = []): Department[] {
  for (const node of nodes) {
    result.push(node)
    if (node.children?.length) flattenDepartments(node.children, result)
  }
  return result
}

export function DepartmentCreateModal({ open, defaultParentId, departments, onClose }: Props) {
  const { t } = useTranslation()
  const createMutation = useCreateDepartment()
  const [name, setName] = useState('')
  const [parentId, setParentId] = useState<string>('')
  const [inlineError, setInlineError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName('')
      setParentId(defaultParentId ?? '')
      setInlineError(null)
    }
  }, [open, defaultParentId])

  const flat = flattenDepartments(departments)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setInlineError(null)
    try {
      await createMutation.mutateAsync({
        name: name.trim(),
        parent_id: parentId || null,
      })
      onClose()
    } catch (err: unknown) {
      const e = err as Error & { code?: string }
      if (e.code === 'DUPLICATE_NAME') {
        setInlineError(t('departments.errors.duplicateName'))
      } else if (e.code === 'NOT_FOUND') {
        setInlineError(t('departments.errors.parentNotFound'))
      } else {
        onClose(e.message)
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('departments.actions.new')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="dept-name">{t('departments.form.name')}</Label>
            <Input
              id="dept-name"
              value={name}
              onChange={(e) => { setName(e.target.value); setInlineError(null) }}
              maxLength={MAX_NAME_LENGTH}
              autoFocus
            />
            {inlineError && (
              <p className="text-sm text-red-600">{inlineError}</p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="dept-parent">{t('departments.form.parent')}</Label>
            <select
              id="dept-parent"
              value={parentId}
              onChange={(e) => setParentId(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm bg-background"
            >
              <option value="">{t('departments.form.parentTopLevel')}</option>
              {flat.map((d) => (
                <option key={d.id} value={d.id}>{d.name}</option>
              ))}
            </select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onClose()}>
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
