import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from '@/components/ui/sheet'
import { DepartmentTreeSelect } from '@/components/DepartmentTreeSelect'
import { useAssignableRoles } from '@/hooks/useAssignableRoles'
import { userErrorKey } from '@/lib/assignableRoles'
import { useCreateUser, type CreateUserRequest, type CreateUserResponse } from '@/api/users'

interface UserCreateDrawerProps {
  open: boolean
  onClose: () => void
  onCreated?: (resp: CreateUserResponse) => void
}

export function UserCreateDrawer({ open, onClose, onCreated }: UserCreateDrawerProps) {
  const { t } = useTranslation()
  const createUser = useCreateUser()
  const { roles } = useAssignableRoles()

  const [form, setForm] = useState<CreateUserRequest>({
    email: '',
    full_name: '',
    department_id: null,
    role_id: '',
  })
  const [error, setError] = useState<string | null>(null)

  function handleChange(field: keyof CreateUserRequest) {
    return (e: React.ChangeEvent<HTMLInputElement>) => {
      setForm((prev) => ({ ...prev, [field]: e.target.value || null }))
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      const resp = await createUser.mutateAsync(form)
      if (onCreated) onCreated(resp)
      setForm({ email: '', full_name: '', department_id: null, role_id: '' })
    } catch (err) {
      const key = userErrorKey(err)
      setError(key ? t(key) : err instanceof Error ? err.message : t('users.messages.error_generic'))
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t('users.create_drawer.title')}</SheetTitle>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.form.email_label')}</label>
            <Input
              type="email"
              placeholder={t('users.form.email_placeholder')}
              value={form.email}
              onChange={handleChange('email')}
              required
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.form.full_name_label')}</label>
            <Input
              placeholder={t('users.form.full_name_placeholder')}
              value={form.full_name}
              onChange={handleChange('full_name')}
              required
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm font-medium">{t('users.form.department_label')}</label>
            <DepartmentTreeSelect
              value={form.department_id}
              onChange={(id) => setForm((prev) => ({ ...prev, department_id: id }))}
              placeholder={t('users.form.department_placeholder')}
              disabled={createUser.isPending}
            />
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
          {error && <p className="text-sm text-danger">{error}</p>}
          <SheetFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('users.deactivate_dialog.cancel')}
            </Button>
            <Button type="submit" disabled={createUser.isPending}>
              {t('users.create_drawer.submit')}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
