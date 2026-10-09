import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  ROLE_NAME_PATTERN,
  roleErrorKey,
  useCreateRole,
  usePermissionsCatalogue,
  useUpdateRole,
  type AdminRole,
} from '@/api/roles'
import { PermissionMatrix } from './PermissionMatrix'

interface RoleFormDialogProps {
  open: boolean
  /** null = create. A system role opens read-only (View). */
  role: AdminRole | null
  onClose: () => void
  onSaved: (message: string) => void
}

export function RoleFormDialog({ open, role, onClose, onSaved }: RoleFormDialogProps) {
  const { t } = useTranslation()
  const { data: catalogue = [], isLoading: catalogueLoading, isError: catalogueError } = usePermissionsCatalogue()
  const createRole = useCreateRole()
  const updateRole = useUpdateRole()

  const isEdit = role !== null
  const readOnly = !!role?.is_system
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [nameError, setNameError] = useState<string | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  // Reset whenever the dialog is (re)opened for a role.
  useEffect(() => {
    if (!open) return
    setName(role?.name ?? '')
    setDescription(role?.description ?? '')
    setNameError(null)
    setFormError(null)
    createRole.reset()
    updateRole.reset()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, role])

  // The role carries "resource:action" strings; the matrix and the API work in permission ids.
  useEffect(() => {
    if (!open) return
    const held = new Set(role?.permissions ?? [])
    setSelected(new Set(catalogue.filter((p) => held.has(`${p.resource}:${p.action}`)).map((p) => p.id)))
  }, [open, role, catalogue])

  const pending = createRole.isPending || updateRole.isPending

  function showError(err: unknown) {
    const { key, values } = roleErrorKey(err)
    const code = (err as { code?: string } | null)?.code
    const text = t(key, values)
    if (code === 'ROLE_NAME_TAKEN') setNameError(text)
    else setFormError(text)
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (readOnly) return
    setNameError(null)
    setFormError(null)
    const ids = [...selected]
    try {
      if (isEdit && role) {
        await updateRole.mutateAsync({ id: role.id, description: description.trim(), permissions: ids })
        onSaved(t('roles.toast.updated', { name: role.name }))
      } else {
        const trimmed = name.trim()
        if (!ROLE_NAME_PATTERN.test(trimmed)) {
          setNameError(t('roles.errors.nameInvalid'))
          return
        }
        await createRole.mutateAsync({ name: trimmed, description: description.trim(), permissions: ids })
        onSaved(t('roles.toast.created', { name: trimmed }))
      }
      onClose()
    } catch (err) {
      showError(err)
    }
  }

  const title = readOnly ? t('roles.form.viewTitle') : isEdit ? t('roles.form.editTitle') : t('roles.form.createTitle')

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v && !pending) onClose()
      }}
    >
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto" aria-labelledby="role-form-title">
        <DialogHeader>
          <DialogTitle id="role-form-title">{title}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div className="space-y-1">
            <Label htmlFor="role-name">{t('roles.form.name')}</Label>
            <Input
              id="role-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isEdit || pending}
              autoFocus={!isEdit}
              autoComplete="off"
              aria-invalid={!!nameError}
              aria-describedby={nameError ? 'role-name-error' : isEdit ? undefined : 'role-name-hint'}
            />
            {!isEdit && (
              <p id="role-name-hint" className="text-xs text-muted-foreground">
                {t('roles.form.nameHint')}
              </p>
            )}
            {nameError && (
              <p id="role-name-error" role="alert" className="text-sm text-red-600">
                {nameError}
              </p>
            )}
          </div>
          <div className="space-y-1">
            <Label htmlFor="role-description">{t('roles.form.description')}</Label>
            <Input
              id="role-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={readOnly || pending}
            />
          </div>
          <div className="space-y-2">
            <h3 className="text-sm font-medium">{t('roles.form.permissions')}</h3>
            {readOnly && <p className="text-xs text-muted-foreground">{t('roles.form.systemReadOnly')}</p>}
            {catalogueLoading && <p className="text-sm text-muted-foreground">{t('common.loading')}</p>}
            {catalogueError && (
              <p role="alert" className="text-sm text-red-600">
                {t('common.loadError')}
              </p>
            )}
            {!catalogueLoading && !catalogueError && (
              <PermissionMatrix
                catalogue={catalogue}
                selected={selected}
                onChange={setSelected}
                readOnly={readOnly || pending}
              />
            )}
          </div>
          {formError && (
            <p role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
              {formError}
            </p>
          )}
          <DialogFooter className="flex flex-col-reverse gap-2 sm:flex-row">
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              {readOnly ? t('roles.form.close') : t('common.cancel')}
            </Button>
            {!readOnly && (
              <Button type="submit" disabled={pending || catalogueLoading || catalogueError}>
                {pending ? t('roles.form.saving') : t('common.save')}
              </Button>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
