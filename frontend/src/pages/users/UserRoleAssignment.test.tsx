import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { UserCreateDrawer } from './UserCreateDrawer'
import { UserEditDrawer } from './UserEditDrawer'
import { ImportModal } from './ImportModal'
import { DeactivateConfirmDialog } from './DeactivateConfirmDialog'
import type { User } from '@/api/users'

const roles = [
  { id: 'r-sa', name: 'super_admin' },
  { id: 'r-da', name: 'department_admin' },
  { id: 'r-ex', name: 'examiner' },
  { id: 'r-emp', name: 'employee' },
]

const server = setupServer(
  http.get('/api/v1/departments', () => HttpResponse.json({ data: [], error: null })),
  http.get('/api/v1/users/roles', () => HttpResponse.json({ data: roles, error: null })),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function wrap(role: string, ui: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], `h.${btoa(JSON.stringify({ role }))}.s`)
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

const target = (role_id: string, role_name: string): User => ({
  id: 'u1', email: 'a@b.c', full_name: 'Target', department_id: null, department_name: null,
  role_id, role_name, status: 'active', force_password_change: false, is_locked: false, created_at: '',
})

function optionNames() {
  return screen.getAllByRole('option').map((o) => o.textContent)
}

describe('role dropdown follows the rank rule', () => {
  it('create drawer: department_admin sees only examiner and employee', async () => {
    wrap('department_admin', <UserCreateDrawer open onClose={() => {}} />)
    await waitFor(() => expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument())
    expect(screen.queryByRole('option', { name: 'department_admin' })).toBeNull()
    expect(screen.queryByRole('option', { name: 'super_admin' })).toBeNull()
    expect(screen.getByRole('option', { name: 'examiner' })).toBeInTheDocument()
  })

  it('create drawer: super_admin sees every role', async () => {
    wrap('super_admin', <UserCreateDrawer open onClose={() => {}} />)
    await waitFor(() => expect(screen.getByRole('option', { name: 'super_admin' })).toBeInTheDocument())
    expect(optionNames()).toHaveLength(5) // placeholder + 4
  })

  it('edit drawer: assignable current role keeps an enabled dropdown', async () => {
    wrap('department_admin', <UserEditDrawer user={target('r-ex', 'examiner')} onClose={() => {}} />)
    await waitFor(() => expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument())
    const sel = screen.getByRole('option', { name: 'examiner' }).closest('select') as HTMLSelectElement
    expect(sel).toBeEnabled()
    expect(screen.queryByText('This user\'s current role cannot be changed by you.')).toBeNull()
  })

  it('edit drawer: non-assignable current role is shown read-only, not replaced', async () => {
    wrap('department_admin', <UserEditDrawer user={target('r-da', 'department_admin')} onClose={() => {}} />)
    await waitFor(() => expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument())
    const current = screen.getByRole('option', { name: 'department_admin' }) as HTMLOptionElement
    const select = current.closest('select') as HTMLSelectElement
    expect(select.value).toBe('r-da')
    expect(select).toBeDisabled()
    expect(screen.getByText("This user's current role cannot be changed by you.")).toBeInTheDocument()
  })

  it('import modal lists only assignable roles in the hint', async () => {
    wrap('department_admin', <ImportModal open onClose={() => {}} />)
    expect(await screen.findByText('Roles you may assign: examiner, employee')).toBeInTheDocument()
  })
})

describe('403 FORBIDDEN is localized', () => {
  const forbidden = () =>
    HttpResponse.json({ data: null, error: { code: 'FORBIDDEN', message: 'forbidden' } }, { status: 403 })

  const MSG = 'You are not allowed to assign this role or act on this user.'

  it('create shows the forbidden message', async () => {
    server.use(http.post('/api/v1/users', forbidden))
    const user = userEvent.setup()
    wrap('department_admin', <UserCreateDrawer open onClose={() => {}} />)
    await waitFor(() => expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument())
    await user.type(screen.getByPlaceholderText('user@example.com'), 'x@y.z')
    await user.type(screen.getByPlaceholderText('Enter full name'), 'New User')
    const sel = screen.getByRole('option', { name: 'employee' }).closest('select') as HTMLSelectElement
    await user.selectOptions(sel, 'r-emp')
    await user.click(screen.getByRole('button', { name: 'Create' }))
    expect(await screen.findByText(MSG)).toBeInTheDocument()
  })

  it('edit shows the forbidden message', async () => {
    server.use(http.put('/api/v1/users/u1', forbidden))
    const user = userEvent.setup()
    wrap('department_admin', <UserEditDrawer user={target('r-ex', 'examiner')} onClose={() => {}} />)
    await waitFor(() => expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'Save Changes' }))
    expect(await screen.findByText(MSG)).toBeInTheDocument()
  })

  it('deactivate shows the forbidden message', async () => {
    server.use(http.post('/api/v1/users/u1/deactivate', forbidden))
    const user = userEvent.setup()
    wrap('department_admin', <DeactivateConfirmDialog user={target('r-da', 'department_admin')} onClose={() => {}} />)
    await user.click(await screen.findByRole('button', { name: 'Deactivate' }))
    expect(await screen.findByText(MSG)).toBeInTheDocument()
  })
})
