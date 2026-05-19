import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from '@/components/ui/sheet'
import { useUpdateUser, useRoles, type User, type UpdateUserRequest } from '@/api/users'
import { useDepartments } from '@/api/departments'

interface UserEditDrawerProps {
  user: User | null
  onClose: () => void
}

export function UserEditDrawer({ user, onClose }: UserEditDrawerProps) {
  const { t } = useTranslation()
  const updateUser = useUpdateUser(user?.id ?? '')
  const { data: departments = [] } = useDepartments()
  const { data: roles = [] } = useRoles()

  const [form, setForm] = useState<UpdateUserRequest>({
    full_name: '',
    department_id: null,
    role_id: '',
  })
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (user) {
      setForm({
        full_name: user.full_name,
        department_id: user.department_id,
        role_id: user.role_id,
      })
      setError(null)
    }
  }, [user])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateUser.mutateAsync(form)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('users.messages.error_generic'))
    }
  }

  return (
    <Sheet open={user !== null} onOpenChange={(v) => { if (!v) onClose() }}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t('users.edit_drawer.title')}</SheetTitle>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.form.full_name_label')}</label>
            <Input
              placeholder={t('users.form.full_name_placeholder')}
              value={form.full_name}
              onChange={(e) => setForm((prev) => ({ ...prev, full_name: e.target.value }))}
              required
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.form.department_label')}</label>
            <Select
              value={form.department_id ?? ''}
              onChange={(e) =>
                setForm((prev) => ({ ...prev, department_id: e.target.value || null }))
              }
            >
              <option value="">{t('users.form.department_placeholder')}</option>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>{d.name}</option>
              ))}
            </Select>
          </div>
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.form.role_label')}</label>
            <Select
              value={form.role_id}
              onChange={(e) => setForm((prev) => ({ ...prev, role_id: e.target.value }))}
              required
            >
              <option value="">{t('users.form.role_placeholder')}</option>
              {roles.map((r) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </Select>
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <SheetFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('users.deactivate_dialog.cancel')}
            </Button>
            <Button type="submit" disabled={updateUser.isPending}>
              {t('users.edit_drawer.submit')}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
