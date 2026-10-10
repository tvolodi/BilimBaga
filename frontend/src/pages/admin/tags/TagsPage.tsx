import { useState, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Hash, Plus, ChevronUp, ChevronDown, AlertCircle, X, MoreHorizontal } from 'lucide-react'
import { useMe } from '@/api/users'
import { useTags, useDeleteTag, type Tag } from '@/api/tags'
import { TagCreateModal } from './TagCreateModal'
import { TagRenameModal } from './TagRenameModal'
import { TagDeleteConfirm } from './TagDeleteConfirm'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const MANAGE_ROLES = new Set(['super_admin', 'department_admin', 'examiner'])
const PAGE_SIZE_OPTIONS = [25, 50, 100]
const DEFAULT_PAGE_SIZE = 50
const SEARCH_DEBOUNCE_MS = 200

type SortKey = 'name' | 'usage_count' | 'created_at'
type SortDir = 'asc' | 'desc'

interface Notification {
  type: 'success' | 'error'
  message: string
}

function NotificationBanner({ notification, onDismiss }: { notification: Notification; onDismiss: () => void }) {
  return (
    <div
      className={`flex items-center gap-3 px-4 py-3 rounded-md text-sm mb-4 ${
        notification.type === 'error'
          ? 'bg-bg-danger border border-transparent text-danger'
          : 'bg-bg-success border border-transparent text-success'
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

function SortIcon({ active, dir }: { active: boolean; dir: SortDir }) {
  if (!active) return <ChevronUp size={14} className="opacity-20" />
  return dir === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
}

interface ActionsMenuProps {
  tag: Tag
  onRename: (tag: Tag) => void
  onDelete: (tag: Tag) => void
}

function ActionsMenu({ tag, onRename, onDelete }: ActionsMenuProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  // Close on outside click
  useMemo(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative inline-block">
      <Button
        variant="ghost"
        size="sm"
        className="h-7 w-7 p-0"
        onClick={() => setOpen((v) => !v)}
        aria-label={t('common.actions')}
      >
        <MoreHorizontal size={15} aria-hidden="true" />
      </Button>
      {open && (
        <div className="absolute right-0 z-50 mt-1 w-36 rounded-md border bg-popover text-popover-foreground shadow-md text-sm">
          <button
            className="w-full px-3 py-2 text-left hover:bg-muted"
            onClick={() => { setOpen(false); onRename(tag) }}
          >
            {t('tags.actions.rename')}
          </button>
          <button
            className="w-full px-3 py-2 text-left text-danger hover:bg-bg-danger"
            onClick={() => { setOpen(false); onDelete(tag) }}
          >
            {t('tags.actions.delete')}
          </button>
        </div>
      )}
    </div>
  )
}

export function TagsPage() {
  const { t } = useTranslation()
  const { data: me } = useMe()
  const { data: tags = [] as Tag[], isLoading, isError } = useTags()
  const deleteMutation = useDeleteTag()

  const canManage = MANAGE_ROLES.has(me?.role_name ?? '')

  const [createOpen, setCreateOpen] = useState(false)
  const [renameTag, setRenameTag] = useState<Tag | null>(null)
  const [deleteTag, setDeleteTag] = useState<Tag | null>(null)
  const [notification, setNotification] = useState<Notification | null>(null)

  // Search state with debounce
  const [searchInput, setSearchInput] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  function handleSearchChange(value: string) {
    setSearchInput(value)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => setSearchQuery(value), SEARCH_DEBOUNCE_MS)
  }

  // Sort state
  const [sortKey, setSortKey] = useState<SortKey>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  function handleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
    setPage(1)
  }

  // Pagination state
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)

  // Derived: filter + sort + paginate
  const filtered = useMemo(() => {
    const q = searchQuery.toLowerCase()
    return tags.filter((t) => t.name.toLowerCase().includes(q))
  }, [tags, searchQuery])

  const sorted = useMemo(() => {
    return [...filtered].sort((a, b) => {
      let cmp = 0
      if (sortKey === 'name') cmp = a.name.localeCompare(b.name)
      else if (sortKey === 'usage_count') cmp = a.usage_count - b.usage_count
      else if (sortKey === 'created_at') cmp = a.created_at.localeCompare(b.created_at)
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [filtered, sortKey, sortDir])

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize))
  const currentPage = Math.min(page, totalPages)
  const paginated = sorted.slice((currentPage - 1) * pageSize, currentPage * pageSize)

  function showNotification(type: Notification['type'], message: string) {
    setNotification({ type, message })
    setTimeout(() => setNotification(null), 5000)
  }

  async function handleDeleteConfirm() {
    if (!deleteTag) return
    const usageCount = deleteTag.usage_count
    try {
      await deleteMutation.mutateAsync(deleteTag.id)
      setDeleteTag(null)
    } catch (err: unknown) {
      const e = err as Error & { code?: string }
      if (e.code === 'ERR_TAG_IN_USE') {
        showNotification('error', t('tags.errors.inUse', { count: usageCount }))
      } else if (e.code === 'ERR_NOT_FOUND') {
        showNotification('error', t('tags.errors.notFound'))
      } else {
        showNotification('error', e.message)
      }
      setDeleteTag(null)
    }
  }

  function ThCol({ label, sortable, colKey }: { label: string; sortable?: boolean; colKey?: SortKey }) {
    if (!sortable || !colKey) {
      return <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</th>
    }
    return (
      <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground uppercase tracking-wide">
        <button
          onClick={() => handleSort(colKey)}
          className="flex items-center gap-1 hover:text-foreground transition-colors"
        >
          {label}
          <SortIcon active={sortKey === colKey} dir={sortDir} />
        </button>
      </th>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Hash size={20} className="text-muted-foreground" />
          <h1 className="text-xl font-semibold">{t('tags.title')}</h1>
        </div>
        {canManage && (
          <Button onClick={() => setCreateOpen(true)} size="sm">
            <Plus size={16} className="mr-1" />
            {t('tags.actions.new')}
          </Button>
        )}
      </div>

      {/* Notification */}
      {notification && (
        <NotificationBanner notification={notification} onDismiss={() => setNotification(null)} />
      )}

      {/* Search */}
      <Input
        placeholder={t('tags.search')}
        value={searchInput}
        onChange={(e) => handleSearchChange(e.target.value)}
        className="max-w-sm"
      />

      {/* States */}
      {isLoading && (
        <div className="text-sm text-muted-foreground py-8 text-center">{t('common.loading')}</div>
      )}
      {isError && (
        <div className="text-sm text-danger py-8 text-center">{t('common.loadError')}</div>
      )}

      {/* Table */}
      {!isLoading && !isError && (
        <>
          {sorted.length === 0 ? (
            <div className="text-sm text-muted-foreground py-12 text-center">{t('tags.empty')}</div>
          ) : (
            <div className="rounded-md border bg-background overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="border-b bg-muted/50">
                  <tr>
                    <ThCol label={t('tags.columns.name')} sortable colKey="name" />
                    <ThCol label={t('tags.columns.usage')} sortable colKey="usage_count" />
                    <ThCol label={t('tags.columns.createdAt')} sortable colKey="created_at" />
                    {canManage && <ThCol label={t('tags.columns.actions')} />}
                  </tr>
                </thead>
                <tbody>
                  {paginated.map((tag) => (
                    <tr key={tag.id} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-2.5 font-medium">{tag.name}</td>
                      <td className="px-4 py-2.5 text-muted-foreground">{tag.usage_count}</td>
                      <td className="px-4 py-2.5 text-muted-foreground">
                        {new Date(tag.created_at).toLocaleDateString()}
                      </td>
                      {canManage && (
                        <td className="px-4 py-2.5">
                          <ActionsMenu
                            tag={tag}
                            onRename={(t) => setRenameTag(t)}
                            onDelete={(t) => setDeleteTag(t)}
                          />
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Pagination */}
          {sorted.length > 0 && (
            <div className="flex items-center justify-between text-sm text-muted-foreground">
              <div className="flex items-center gap-2">
                <span>{t('questionBank.pagination.pageSize')}:</span>
                <select
                  value={pageSize}
                  onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1) }}
                  className="border rounded px-2 py-1 text-sm bg-background"
                >
                  {PAGE_SIZE_OPTIONS.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
              <div className="flex items-center gap-3">
                <span>{t('questionBank.pagination.pageOf', { page: currentPage, total: totalPages })}</span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={currentPage <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t('questionBank.pagination.previous')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={currentPage >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t('questionBank.pagination.next')}
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Modals */}
      <TagCreateModal open={createOpen} onClose={() => setCreateOpen(false)} />
      <TagRenameModal open={!!renameTag} tag={renameTag} onClose={() => setRenameTag(null)} />
      <TagDeleteConfirm
        open={!!deleteTag}
        isPending={deleteMutation.isPending}
        onClose={() => setDeleteTag(null)}
        onConfirm={handleDeleteConfirm}
      />
    </div>
  )
}
