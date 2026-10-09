import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui/table'
import { DepartmentTreeSelect } from '@/components/DepartmentTreeSelect'
import { useUsers, useResetPassword, type User, type UsersFilters } from '@/api/users'
import { UserCreateDrawer } from './UserCreateDrawer'
import { UserEditDrawer } from './UserEditDrawer'
import { userErrorKey } from '@/lib/assignableRoles'
import { DeactivateConfirmDialog } from './DeactivateConfirmDialog'
import { PasswordResetModal } from './PasswordResetModal'
import { ImportModal } from './ImportModal'
import type { CreateUserResponse } from '@/api/users'

export function UsersListPage() {
  const { t } = useTranslation()

  // Filters and pagination
  const [filters, setFilters] = useState<UsersFilters>({ page: 1, per_page: 20 })

  // Drawer/modal state
  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | null>(null)
  const [deactivateUser, setDeactivateUser] = useState<User | null>(null)
  const [importOpen, setImportOpen] = useState(false)

  // Password reset state
  const [resetUserId, setResetUserId] = useState<string | null>(null)
  const [resetPassword, setResetPassword] = useState<string | null>(null)
  const [resetModalOpen, setResetModalOpen] = useState(false)
  const [resetError, setResetError] = useState<string | null>(null)
  const resetMutation = useResetPassword(resetUserId ?? '')

  // Created user temp password
  const [createdPassword, setCreatedPassword] = useState<string | null>(null)
  const [createdPasswordOpen, setCreatedPasswordOpen] = useState(false)

  const { data, isLoading, isError } = useUsers(filters)

  function setFilter(key: keyof UsersFilters, value: string) {
    setFilters((prev) => ({ ...prev, [key]: value || undefined, page: 1 }))
  }

  async function handleResetPassword(user: User) {
    setResetUserId(user.id)
    setResetError(null)
    try {
      const resp = await resetMutation.mutateAsync()
      setResetPassword(resp.temporary_password)
      setResetModalOpen(true)
    } catch (err) {
      setResetError(t(userErrorKey(err) ?? 'users.messages.error_generic'))
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
    <div className="p-6 space-y-4">
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

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <Select
          className="w-48"
          value={filters.status ?? ''}
          onChange={(e) => setFilter('status', e.target.value)}
        >
          <option value="">{t('users.filters.all')}</option>
          <option value="active">{t('users.status.active')}</option>
          <option value="inactive">{t('users.status.inactive')}</option>
        </Select>
        <div className="w-64">
          <DepartmentTreeSelect
            value={filters.department_id ?? null}
            onChange={(id) => setFilter('department_id', id ?? '')}
            placeholder={t('users.filters.department')}
            clearable
          />
        </div>
      </div>

      {/* Table */}
      {isLoading && <p className="text-muted-foreground">Loading...</p>}
      {resetError && <p role="alert" className="text-red-600">{resetError}</p>}
      {isError && <p className="text-red-600">{t('users.messages.error_generic')}</p>}
      {data && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('users.columns.name')}</TableHead>
              <TableHead>{t('users.columns.email')}</TableHead>
              <TableHead>{t('users.columns.department')}</TableHead>
              <TableHead>{t('users.columns.role')}</TableHead>
              <TableHead>{t('users.columns.status')}</TableHead>
              <TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.items.map((user) => (
              <TableRow key={user.id}>
                <TableCell className="font-medium">{user.full_name}</TableCell>
                <TableCell>{user.email}</TableCell>
                <TableCell>{user.department_name ?? '—'}</TableCell>
                <TableCell>{user.role_name}</TableCell>
                <TableCell>
                  <Badge variant={user.status === 'active' ? 'success' : 'secondary'}>
                    {t(`users.status.${user.status}`)}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setEditUser(user)}
                    >
                      {t('users.actions.edit')}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => handleResetPassword(user)}
                    >
                      {t('users.actions.reset_password')}
                    </Button>
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
      <ImportModal open={importOpen} onClose={() => setImportOpen(false)} />

      {/* Password shown after create */}
      <PasswordResetModal
        open={createdPasswordOpen}
        temporaryPassword={createdPassword}
        onClose={() => {
          setCreatedPasswordOpen(false)
          setCreatedPassword(null)
        }}
      />

      {/* Password shown after reset */}
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
