import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui/table'
import { RoleBadge } from '@/components/admin/RoleBadge'
import { StatusBadge } from '@/components/admin/StatusBadge'
import { useUsers, useRoles, useResetPassword, useUnlockUser, type User, type UsersFilters } from '@/api/users'
import { DepartmentTreeSelect } from '@/components/DepartmentTreeSelect'
import { UserCreateDrawer } from './UserCreateDrawer'
import { UserEditDrawer } from './UserEditDrawer'
import { ImportModal } from './ImportModal'
import { DeactivateConfirmDialog } from '@/pages/users/DeactivateConfirmDialog'
import { ReactivateConfirmDialog } from '@/pages/users/ReactivateConfirmDialog'
import { PasswordResetModal } from '@/pages/users/PasswordResetModal'
import type { CreateUserResponse } from '@/api/users'
import { userErrorKey } from '@/lib/assignableRoles'

type SortKey = 'full_name' | 'email' | 'department_name' | 'role_name' | 'status'
type SortDir = 'asc' | 'desc'

const VIEW_RECORD_ROLES = new Set(['super_admin', 'department_admin', 'examiner'])

export function UsersListPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const currentUser = qc.getQueryData<{ role: string }>(['auth', 'currentUser'])
  const canViewRecord = currentUser ? VIEW_RECORD_ROLES.has(currentUser.role) : false

  const [filters, setFilters] = useState<UsersFilters>({ page: 1, per_page: 20 })
  const [sortKey, setSortKey] = useState<SortKey>('full_name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | null>(null)
  const [deactivateUser, setDeactivateUser] = useState<User | null>(null)
  const [reactivateUser, setReactivateUser] = useState<User | null>(null)
  const [importOpen, setImportOpen] = useState(false)

  const [resetUserId, setResetUserId] = useState<string | null>(null)
  const [resetPassword, setResetPassword] = useState<string | null>(null)
  const [resetModalOpen, setResetModalOpen] = useState(false)
  const resetMutation = useResetPassword(resetUserId ?? '')

  const [createdPassword, setCreatedPassword] = useState<string | null>(null)
  const [createdPasswordOpen, setCreatedPasswordOpen] = useState(false)

  const { data, isLoading, isError } = useUsers(filters)
  const { data: roles = [] } = useRoles()

  // FR-BB115 AC-8: unlock action with an inline, auto-dismissing notice.
  const unlockMutation = useUnlockUser()
  const [notice, setNotice] = useState<{ text: string; type: 'success' | 'error' } | null>(null)

  async function handleUnlock(user: User) {
    try {
      await unlockMutation.mutateAsync(user.id)
      setNotice({ text: t('users.messages.unlock_success'), type: 'success' })
    } catch (err) {
      setNotice({ text: t(userErrorKey(err) ?? 'users.messages.unlock_error'), type: 'error' })
    }
    setTimeout(() => setNotice(null), 5000)
  }

  function showError(text: string) {
    setNotice({ text, type: 'error' })
    setTimeout(() => setNotice(null), 5000)
  }

  function setFilter(key: keyof UsersFilters, value: string) {
    setFilters((prev) => ({ ...prev, [key]: value || undefined, page: 1 }))
  }

  function handleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  function sortIndicator(key: SortKey) {
    if (sortKey !== key) return null
    return sortDir === 'asc' ? ' ↑' : ' ↓'
  }

  const sortedItems = [...(data?.items ?? [])].sort((a, b) => {
    const av = (a[sortKey] ?? '') as string
    const bv = (b[sortKey] ?? '') as string
    const cmp = av.localeCompare(bv)
    return sortDir === 'asc' ? cmp : -cmp
  })

  async function handleResetPassword(user: User) {
    setResetUserId(user.id)
    setNotice(null)
    try {
      const resp = await resetMutation.mutateAsync()
      setResetPassword(resp.temporary_password)
      setResetModalOpen(true)
    } catch (err) {
      showError(t(userErrorKey(err) ?? 'users.messages.error_generic'))
    }
  }

  function handleCreated(resp: CreateUserResponse) {
    setCreateOpen(false)
    setCreatedPassword(resp.temporary_password)
    setCreatedPasswordOpen(true)
  }

  const total = data?.meta.total ?? 0
  const page = data?.meta.page ?? 1
  const perPage = data?.meta.per_page ?? 20
  const totalPages = Math.max(1, Math.ceil(total / perPage))

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('users.title')}</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setImportOpen(true)}>
            {t('users.actions.import')}
          </Button>
          <Button onClick={() => setCreateOpen(true)}>
            {t('users.actions.create')}
          </Button>
        </div>
      </div>

      {notice && (
        <p
          role={notice.type === 'error' ? 'alert' : 'status'}
          className={`rounded-md border px-4 py-3 text-sm ${
            notice.type === 'error'
              ? 'border-red-200 bg-red-50 text-red-800'
              : 'border-green-200 bg-green-50 text-green-800'
          }`}
        >
          {notice.text}
        </p>
      )}

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="w-64">
          <DepartmentTreeSelect
            value={filters.department_id ?? null}
            onChange={(id) => setFilter('department_id', id ?? '')}
            placeholder={t('users.filters.department')}
            aria-label={t('users.filters.department')}
            clearable
          />
        </div>

        <Select
          className="w-48"
          aria-label={t('users.filters.role')}
          value={filters.role_id ?? ''}
          onChange={(e) => setFilter('role_id', e.target.value)}
        >
          <option value="">{t('users.filters.role')}</option>
          {roles.map((r) => (
            <option key={r.id} value={r.id}>
              {t(`users.roles.${r.name}`, { defaultValue: r.name })}
            </option>
          ))}
        </Select>

        <Select
          className="w-40"
          value={filters.status ?? ''}
          onChange={(e) => setFilter('status', e.target.value)}
        >
          <option value="">{t('users.filters.status')}</option>
          <option value="active">{t('users.status.active')}</option>
          <option value="inactive">{t('users.status.inactive')}</option>
        </Select>
      </div>

      {/* Table */}
      {isLoading && <p className="text-muted-foreground">…</p>}
      {isError && <p className="text-red-600">{t('users.messages.error_generic')}</p>}
      {data && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort('full_name')}
              >
                {t('users.columns.name')}{sortIndicator('full_name')}
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort('email')}
              >
                {t('users.columns.email')}{sortIndicator('email')}
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort('department_name')}
              >
                {t('users.columns.department')}{sortIndicator('department_name')}
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort('role_name')}
              >
                {t('users.columns.role')}{sortIndicator('role_name')}
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort('status')}
              >
                {t('users.columns.status')}{sortIndicator('status')}
              </TableHead>
              <TableHead>{t('users.columns.actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedItems.map((user) => (
              <TableRow key={user.id}>
                <TableCell className="font-medium">{user.full_name}</TableCell>
                <TableCell>{user.email}</TableCell>
                <TableCell>{user.department_name ?? '—'}</TableCell>
                <TableCell>
                  <RoleBadge role={user.role_name} />
                </TableCell>
                <TableCell>
                  <StatusBadge status={user.status} />
                  {user.is_locked && (
                    <span className="ml-2 inline-flex items-center rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-semibold text-amber-800">
                      {t('users.status.locked')}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    {canViewRecord && (
                      <Button size="sm" variant="ghost" asChild>
                        <Link to={`/admin/users/${user.id}/record`}>
                          {t('users.actions.view_record')}
                        </Link>
                      </Button>
                    )}
                    <Button size="sm" variant="ghost" onClick={() => setEditUser(user)}>
                      {t('users.actions.edit')}
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => handleResetPassword(user)}>
                      {t('users.actions.reset_password')}
                    </Button>
                    {user.is_locked && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={unlockMutation.isPending}
                        onClick={() => handleUnlock(user)}
                      >
                        {t('users.actions.unlock')}
                      </Button>
                    )}
                    {user.status === 'active' && (
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeactivateUser(user)}
                      >
                        {t('users.actions.deactivate')}
                      </Button>
                    )}
                    {user.status === 'inactive' && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setReactivateUser(user)}
                      >
                        {t('users.actions.reactivate')}
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Pagination */}
      {data && (
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>{t('users.pagination.page_info', { page, total: totalPages })}</span>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={page <= 1}
              onClick={() => setFilters((prev) => ({ ...prev, page: page - 1 }))}
            >
              {t('users.pagination.previous')}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={page >= totalPages}
              onClick={() => setFilters((prev) => ({ ...prev, page: page + 1 }))}
            >
              {t('users.pagination.next')}
            </Button>
          </div>
        </div>
      )}

      {/* Drawers / Modals */}
      <UserCreateDrawer
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={handleCreated}
      />
      <UserEditDrawer user={editUser} onClose={() => setEditUser(null)} />
      <DeactivateConfirmDialog user={deactivateUser} onClose={() => setDeactivateUser(null)} />
      <ReactivateConfirmDialog user={reactivateUser} onClose={() => setReactivateUser(null)} />
      <ImportModal open={importOpen} onClose={() => setImportOpen(false)} />

      <PasswordResetModal
        open={createdPasswordOpen}
        temporaryPassword={createdPassword}
        onClose={() => {
          setCreatedPasswordOpen(false)
          setCreatedPassword(null)
        }}
      />
      <PasswordResetModal
        open={resetModalOpen}
        temporaryPassword={resetPassword}
        onClose={() => {
          setResetModalOpen(false)
          setResetPassword(null)
          setResetUserId(null)
        }}
      />
    </div>
  )
}
