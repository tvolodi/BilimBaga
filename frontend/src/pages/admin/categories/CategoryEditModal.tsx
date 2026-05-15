import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
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
import { Select } from '@/components/ui/select'
import {
  useCreateCategory,
  useUpdateCategory,
  flattenCategories,
  getDescendantIds,
  type CategoryNode,
  type CategoryCreate,
  type CategoryUpdate,
} from '@/api/categories'

interface CategoryEditModalProps {
  open: boolean
  onClose: () => void
  tree: CategoryNode[]
  /** When set, the modal is in edit mode; when null, it's in create mode */
  editing: CategoryNode | null
  /** Pre-fills parent_id when opening via "Add child" */
  defaultParentId?: string | null
}

interface FormState {
  name: string
  parent_id: string
  track: string
  sort_order: string
}

const EMPTY_FORM: FormState = { name: '', parent_id: '', track: '', sort_order: '0' }

export function CategoryEditModal({
  open,
  onClose,
  tree,
  editing,
  defaultParentId,
}: CategoryEditModalProps) {
  const { t } = useTranslation()
  const createMutation = useCreateCategory()
  const updateMutation = useUpdateCategory()

  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({})
  const [apiError, setApiError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    if (editing) {
      setForm({
        name: editing.name,
        parent_id: editing.parent_id ?? '',
        track: editing.track ?? '',
        sort_order: String(editing.sort_order),
      })
    } else {
      setForm({ ...EMPTY_FORM, parent_id: defaultParentId ?? '' })
    }
    setErrors({})
    setApiError(null)
  }, [open, editing, defaultParentId])

  const flat = useMemo(() => flattenCategories(tree), [tree])

  const parentOptions = useMemo(() => {
    if (!editing) return flat
    const excluded = new Set([editing.id, ...getDescendantIds(editing)])
    return flat.filter((n) => !excluded.has(n.id))
  }, [flat, editing])

  function getIndentedName(node: CategoryNode): string {
    const ancestors: string[] = []
    let cur: CategoryNode | undefined = node
    while (cur?.parent_id) {
      const parent = flat.find((n) => n.id === cur!.parent_id)
      if (!parent) break
      ancestors.unshift(parent.name)
      cur = parent
    }
    return ancestors.length ? `${ancestors.join(' › ')} › ${node.name}` : node.name
  }

  function validate(): boolean {
    const next: typeof errors = {}
    if (!form.name.trim()) {
      next.name = t('categories.errors.invalidName')
    } else if (form.name.trim().length > 100) {
      next.name = t('categories.errors.invalidName')
    }
    const sortNum = parseInt(form.sort_order, 10)
    if (isNaN(sortNum)) {
      next.sort_order = t('categories.errors.invalidName')
    }
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    setApiError(null)

    try {
      if (editing) {
        const body: CategoryUpdate = {}
        if (form.name.trim() !== editing.name) body.name = form.name.trim()
        if (form.track.trim() !== (editing.track ?? '')) body.track = form.track.trim() || null
        const sortNum = parseInt(form.sort_order, 10)
        if (sortNum !== editing.sort_order) body.sort_order = sortNum
        const newParentId = form.parent_id || null
        if (newParentId !== editing.parent_id) {
          if (newParentId === null) {
            body.clear_parent = true
          } else {
            body.parent_id = newParentId
          }
        }
        await updateMutation.mutateAsync({ id: editing.id, body })
      } else {
        const body: CategoryCreate = {
          name: form.name.trim(),
          sort_order: parseInt(form.sort_order, 10),
        }
        if (form.parent_id) body.parent_id = form.parent_id
        if (form.track.trim()) body.track = form.track.trim()
        await createMutation.mutateAsync(body)
      }
      onClose()
    } catch (err: unknown) {
      const e = err as Error & { code?: string }
      if (e.code === 'CATEGORY_CYCLE') {
        setErrors((prev) => ({ ...prev, parent_id: t('categories.errors.cycle') }))
      } else if (e.code === 'PARENT_NOT_FOUND') {
        setErrors((prev) => ({ ...prev, parent_id: t('categories.errors.parentNotFound') }))
      } else if (e.code === 'INVALID_NAME') {
        setErrors((prev) => ({ ...prev, name: t('categories.errors.invalidName') }))
      } else {
        setApiError(e.message)
      }
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {editing ? t('categories.actions.edit') : t('categories.actions.new')}
          </DialogTitle>
        </DialogHeader>

        <form id="category-edit-form" onSubmit={handleSubmit} className="space-y-4 py-2">
          {/* Name */}
          <div className="space-y-1">
            <Label htmlFor="cat-name">{t('categories.form.name')}</Label>
            <Input
              id="cat-name"
              value={form.name}
              onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
              maxLength={100}
              disabled={isPending}
            />
            {errors.name && <p className="text-xs text-red-600">{errors.name}</p>}
          </div>

          {/* Parent */}
          <div className="space-y-1">
            <Label htmlFor="cat-parent">{t('categories.form.parent')}</Label>
            <Select
              id="cat-parent"
              value={form.parent_id}
              onChange={(e) => setForm((p) => ({ ...p, parent_id: e.target.value }))}
              disabled={isPending}
            >
              <option value="">{t('categories.form.parentTopLevel')}</option>
              {parentOptions.map((n) => (
                <option key={n.id} value={n.id}>
                  {getIndentedName(n)}
                </option>
              ))}
            </Select>
            {errors.parent_id && <p className="text-xs text-red-600">{errors.parent_id}</p>}
          </div>

          {/* Track */}
          <div className="space-y-1">
            <Label htmlFor="cat-track">{t('categories.form.track')}</Label>
            <Input
              id="cat-track"
              value={form.track}
              onChange={(e) => setForm((p) => ({ ...p, track: e.target.value }))}
              maxLength={50}
              placeholder={t('categories.form.trackHelp')}
              disabled={isPending}
            />
          </div>

          {/* Sort order */}
          <div className="space-y-1">
            <Label htmlFor="cat-sort">{t('categories.form.sortOrder')}</Label>
            <Input
              id="cat-sort"
              type="number"
              value={form.sort_order}
              onChange={(e) => setForm((p) => ({ ...p, sort_order: e.target.value }))}
              disabled={isPending}
            />
            {errors.sort_order && <p className="text-xs text-red-600">{errors.sort_order}</p>}
          </div>

          {apiError && <p className="text-sm text-red-600">{apiError}</p>}
        </form>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            {t('common.cancel')}
          </Button>
          <Button
            type="submit"
            form="category-edit-form"
            disabled={isPending}
          >
            {isPending ? '…' : t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
