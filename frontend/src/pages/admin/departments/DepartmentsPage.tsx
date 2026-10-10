import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Building2, Plus, AlertCircle, X } from 'lucide-react'
import { useMe } from '@/api/users'
import { useDepartments, useDeleteDepartment, type Department } from '@/api/departments'
import { DepartmentTree } from '@/components/admin/DepartmentTree'
import { DepartmentCreateModal } from './DepartmentCreateModal'
import { DepartmentRenameModal } from './DepartmentRenameModal'
import { DepartmentDeleteConfirm } from './DepartmentDeleteConfirm'
import { Button } from '@/components/ui/button'
import type { ApiError } from '@/api/apiFetch'

interface Notification {
  type: 'success' | 'error'
  message: string
}

function NotificationBanner({ notification, onDismiss }: { notification: Notification; onDismiss: () => void }) {
  return (
    <div
      className={`flex items-center gap-3 px-4 py-3 rounded-md text-sm mb-4 ${
        notification.type === 'error'
          ? 'bg-bg-danger border border-danger text-danger'
          : 'bg-bg-success border border-success text-success'
      }`}
    >
      <AlertCircle size={16} className="flex-shrink-0" />
      <span className="flex-1">{notification.message}</span>
      <button onClick={onDismiss} className="p-0.5 rounded hover:bg-foreground/10">
        <X size={14} />
      </button>
    </div>
  )
}

export function DepartmentsPage() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: departments = [], isLoading, isError } = useDepartments()
  const deleteMutation = useDeleteDepartment()

  const canManage = me?.role_name === 'super_admin'

  const [createOpen, setCreateOpen] = useState(false)
  const [createParentId, setCreateParentId] = useState<string | null>(null)
  const [renameTarget, setRenameTarget] = useState<Department | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Department | null>(null)
  const [notification, setNotification] = useState<Notification | null>(null)

  function showNotification(type: Notification['type'], message: string) {
    setNotification({ type, message })
    setTimeout(() => setNotification(null), 5000)
  }

  function handleAddChild(node: Department) {
    setCreateParentId(node.id)
    setCreateOpen(true)
  }

  function handleNewDepartment() {
    setCreateParentId(null)
    setCreateOpen(true)
  }

  function handleCreateClose(errorMessage?: string) {
    setCreateOpen(false)
    setCreateParentId(null)
    if (errorMessage) {
      showNotification('error', errorMessage)
    }
  }

  async function handleDeleteConfirm() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      setDeleteTarget(null)
      showNotification('success', t('departments.success.deleted'))
    } catch (err: unknown) {
      const e = err as ApiError
      setDeleteTarget(null)
      if (e.code === 'DEPARTMENT_NOT_EMPTY') {
        showNotification('error', t('departments.errors.notEmpty'))
      } else if (e.code === 'DEPARTMENT_HAS_CHILDREN') {
        showNotification('error', t('departments.errors.hasChildren'))
      } else if (e.code === 'NOT_FOUND') {
        showNotification('error', t('departments.errors.parentNotFound'))
      } else {
        showNotification('error', e.message)
      }
    }
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Building2 size={20} className="text-muted-foreground" />
          <h1 className="text-xl font-semibold">{t('departments.title')}</h1>
        </div>
        {canManage && (
          <Button onClick={handleNewDepartment} size="sm">
            <Plus size={16} className="mr-1" />
            {t('departments.actions.new')}
          </Button>
        )}
      </div>

      {/* Notification */}
      {notification && (
        <NotificationBanner notification={notification} onDismiss={() => setNotification(null)} />
      )}

      {/* Loading / error states */}
      {isLoading && (
        <div className="text-sm text-muted-foreground py-8 text-center">{t('common.loading')}</div>
      )}
      {isError && (
        <div className="text-sm text-danger py-8 text-center">{t('common.loadError')}</div>
      )}

      {/* Tree */}
      {!isLoading && !isError && (
        departments.length === 0 ? (
          <div className="text-sm text-muted-foreground py-12 text-center">{t('departments.empty')}</div>
        ) : (
          <div className="rounded-md border bg-background py-2">
            <DepartmentTree
              nodes={departments}
              canManage={canManage}
              onAddChild={handleAddChild}
              onRename={(node) => setRenameTarget(node)}
              onDelete={(node) => setDeleteTarget(node)}
            />
          </div>
        )
      )}

      {/* Modals */}
      <DepartmentCreateModal
        open={createOpen}
        defaultParentId={createParentId}
        departments={departments}
        onClose={handleCreateClose}
      />
      <DepartmentRenameModal
        open={!!renameTarget}
        department={renameTarget}
        onClose={() => setRenameTarget(null)}
      />
      <DepartmentDeleteConfirm
        open={!!deleteTarget}
        department={deleteTarget}
        isPending={deleteMutation.isPending}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
      />
    </div>
  )
}
