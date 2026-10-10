import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderTree, Plus, AlertCircle, X } from 'lucide-react'
import { useMe } from '@/api/users'
import { useCategories, useDeleteCategory, type CategoryNode } from '@/api/categories'
import { CategoryTree } from '@/components/admin/categories/CategoryTree'
import { CategoryEditModal } from './CategoryEditModal'
import { CategoryDeleteConfirm } from './CategoryDeleteConfirm'
import { Button } from '@/components/ui/button'
import { useUpdateCategory } from '@/api/categories'

// Roles that may manage categories (matches backend `categories:manage` permission)
const MANAGE_ROLES = new Set(['super_admin', 'department_admin', 'examiner'])

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

export function CategoriesPage() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: tree = [], isLoading, isError } = useCategories()
  const deleteMutation = useDeleteCategory()
  const updateMutation = useUpdateCategory()

  const canManage = MANAGE_ROLES.has(me?.role_name ?? '')

  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<CategoryNode | null>(null)
  const [defaultParentId, setDefaultParentId] = useState<string | null>(null)

  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [deletingNode, setDeletingNode] = useState<CategoryNode | null>(null)

  const [notification, setNotification] = useState<Notification | null>(null)

  function showNotification(type: Notification['type'], message: string) {
    setNotification({ type, message })
    setTimeout(() => setNotification(null), 5000)
  }

  function openCreate() {
    setEditingNode(null)
    setDefaultParentId(null)
    setEditModalOpen(true)
  }

  function openAddChild(node: CategoryNode) {
    setEditingNode(null)
    setDefaultParentId(node.id)
    setEditModalOpen(true)
  }

  function openEdit(node: CategoryNode) {
    setEditingNode(node)
    setDefaultParentId(null)
    setEditModalOpen(true)
  }

  function openDelete(node: CategoryNode) {
    setDeletingNode(node)
    setDeleteConfirmOpen(true)
  }

  function handleModalClose() {
    setEditModalOpen(false)
    setEditingNode(null)
    setDefaultParentId(null)
  }

  async function handleDeleteConfirm() {
    if (!deletingNode) return
    try {
      await deleteMutation.mutateAsync(deletingNode.id)
      setDeleteConfirmOpen(false)
      setDeletingNode(null)
      showNotification('success', t('categories.messages.deleteSuccess'))
    } catch (err: unknown) {
      const e = err as Error & { code?: string; details?: Record<string, unknown> }
      if (e.code === 'CATEGORY_IN_USE') {
        const count = (e.details?.in_use_count as number | undefined) ?? 0
        showNotification('error', t('categories.errors.inUse', { count }))
      } else {
        showNotification('error', e.message)
      }
      setDeleteConfirmOpen(false)
      setDeletingNode(null)
    }
  }

  async function handleReorder(_parentId: string | null, nodeId: string, newSortOrder: number) {
    try {
      await updateMutation.mutateAsync({
        id: nodeId,
        body: { sort_order: newSortOrder },
      })
    } catch (err: unknown) {
      const e = err as Error
      showNotification('error', e.message)
    }
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <FolderTree size={20} className="text-muted-foreground" />
          <h1 className="text-xl font-semibold">{t('categories.title')}</h1>
        </div>
        {canManage && (
          <Button onClick={openCreate} size="sm">
            <Plus size={16} className="mr-1" />
            {t('categories.actions.new')}
          </Button>
        )}
      </div>

      {/* Notification banner */}
      {notification && (
        <NotificationBanner notification={notification} onDismiss={() => setNotification(null)} />
      )}

      {/* Content */}
      {isLoading && (
        <div className="text-sm text-muted-foreground py-8 text-center">
          {t('common.loading')}
        </div>
      )}

      {isError && (
        <div className="text-sm text-danger py-8 text-center">
          {t('common.loadError')}
        </div>
      )}

      {!isLoading && !isError && tree.length === 0 && (
        <div className="text-sm text-muted-foreground py-12 text-center">
          {t('categories.empty')}
        </div>
      )}

      {!isLoading && !isError && tree.length > 0 && (
        <div className="rounded-md border bg-background p-2">
          <CategoryTree
            nodes={tree}
            canManage={canManage}
            onAddChild={openAddChild}
            onEdit={openEdit}
            onDelete={openDelete}
            onReorder={handleReorder}
          />
        </div>
      )}

      {/* Edit / Create modal */}
      <CategoryEditModal
        open={editModalOpen}
        onClose={handleModalClose}
        tree={tree}
        editing={editingNode}
        defaultParentId={defaultParentId}
      />

      {/* Delete confirmation */}
      <CategoryDeleteConfirm
        open={deleteConfirmOpen}
        onClose={() => { setDeleteConfirmOpen(false); setDeletingNode(null) }}
        onConfirm={handleDeleteConfirm}
        isPending={deleteMutation.isPending}
      />
    </div>
  )
}
