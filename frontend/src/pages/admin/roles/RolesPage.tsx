import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { roleErrorKey, useDeleteRole, useRolesAdmin, type AdminRole } from '@/api/roles'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { RoleFormDialog } from './RoleFormDialog'

/** FR-BB117: admin page to list, create, edit and delete custom roles. */
export function RolesPage() {
  const { t } = useTranslation()
  const { data: roles = [], isLoading, isError } = useRolesAdmin()
  const deleteRole = useDeleteRole()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<AdminRole | null>(null)
  const [deleting, setDeleting] = useState<AdminRole | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  function openCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(role: AdminRole) {
    setEditing(role)
    setFormOpen(true)
  }

  function openDelete(role: AdminRole) {
    setDeleteError(null)
    setDeleting(role)
  }

  function closeDelete() {
    if (deleteRole.isPending) return
    setDeleting(null)
    setDeleteError(null)
  }

  async function confirmDelete() {
    if (!deleting) return
    setDeleteError(null)
    try {
      await deleteRole.mutateAsync(deleting.id)
      setNotice(t('roles.toast.deleted', { name: deleting.name }))
      setDeleting(null)
    } catch (err) {
      const { key, values } = roleErrorKey(err)
      setDeleteError(t(key, values))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-xl font-semibold">{t('roles.title')}</h1>
        <Button onClick={openCreate}>
          <Plus size={16} className="mr-2" aria-hidden="true" />
          {t('roles.create')}
        </Button>
      </div>

      {notice && (
        <div
          role="status"
          className="flex items-center justify-between gap-3 rounded-md border border-success bg-bg-success px-4 py-3 text-sm text-success"
        >
          <span>{notice}</span>
          <button
            type="button"
            className="rounded px-2 py-0.5 hover:bg-foreground/10"
            onClick={() => setNotice(null)}
            aria-label={t('roles.dismiss')}
          >
            ×
          </button>
        </div>
      )}

      {isLoading && <p className="text-sm text-muted-foreground">{t('common.loading')}</p>}
      {isError && (
        <p role="alert" className="text-sm text-danger">
          {t('common.loadError')}
        </p>
      )}

      {!isLoading && !isError && (
        <Table aria-label={t('roles.title')}>
          <TableHeader>
            <TableRow>
              <TableHead>{t('roles.columns.name')}</TableHead>
              <TableHead className="hidden sm:table-cell">{t('roles.columns.description')}</TableHead>
              <TableHead>{t('roles.columns.type')}</TableHead>
              <TableHead className="text-right">{t('roles.columns.users')}</TableHead>
              <TableHead className="text-right">{t('roles.columns.permissions')}</TableHead>
              <TableHead className="text-right">{t('roles.columns.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {roles.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground">
                  {t('roles.empty')}
                </TableCell>
              </TableRow>
            )}
            {roles.map((role) => (
              <TableRow key={role.id}>
                <TableCell className="break-all font-medium">{role.name}</TableCell>
                <TableCell className="hidden sm:table-cell">{role.description}</TableCell>
                <TableCell>
                  <Badge variant={role.is_system ? 'secondary' : 'outline'}>
                    {role.is_system ? t('roles.type.system') : t('roles.type.custom')}
                  </Badge>
                </TableCell>
                <TableCell className="text-right">{role.user_count}</TableCell>
                <TableCell className="text-right">{role.permissions.length}</TableCell>
                <TableCell className="text-right">
                  <div className="flex flex-wrap justify-end gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openEdit(role)}
                      aria-label={t(role.is_system ? 'roles.actions.viewRole' : 'roles.actions.editRole', { name: role.name })}
                    >
                      {role.is_system ? t('roles.actions.view') : t('roles.actions.edit')}
                    </Button>
                    {!role.is_system && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-danger"
                        onClick={() => openDelete(role)}
                        aria-label={t('roles.actions.deleteRole', { name: role.name })}
                      >
                        {t('roles.actions.delete')}
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <RoleFormDialog
        open={formOpen}
        role={editing}
        onClose={() => setFormOpen(false)}
        onSaved={(message) => setNotice(message)}
      />

      <Dialog open={deleting !== null} onOpenChange={(v) => { if (!v) closeDelete() }}>
        <DialogContent aria-labelledby="role-delete-title" aria-describedby="role-delete-desc">
          <DialogHeader>
            <DialogTitle id="role-delete-title">{t('roles.delete.title')}</DialogTitle>
            <DialogDescription id="role-delete-desc">
              {t('roles.delete.description', { name: deleting?.name ?? '' })}
            </DialogDescription>
          </DialogHeader>
          {deleteError && (
            <p role="alert" className="mb-2 rounded-md border border-danger bg-bg-danger px-3 py-2 text-sm text-danger">
              {deleteError}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={closeDelete} disabled={deleteRole.isPending}>
              {t('common.cancel')}
            </Button>
            <Button variant="destructive" onClick={confirmDelete} disabled={deleteRole.isPending}>
              {deleteRole.isPending ? t('roles.delete.deleting') : t('roles.actions.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
