import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { UsersListPage } from './UsersListPage'

// ISS-250 (FR-BB117 D-4): the routed users page, opened as a department_admin, offers only
// the roles the caller may assign in both the create and the edit drawer.

const roles = [
  { id: 'r-sa', name: 'super_admin' },
  { id: 'r-da', name: 'department_admin' },
  { id: 'r-ex', name: 'examiner' },
  { id: 'r-emp', name: 'employee' },
]

const listed = [
  { id: 'u-emp', email: 'emp@example.com', full_name: 'Emma Employee', department_id: 'd-1', department_name: 'Eng',
    role_id: 'r-emp', role_name: 'employee', status: 'active', force_password_change: false, is_locked: false, created_at: '' },
  { id: 'u-da', email: 'da@example.com', full_name: 'Dana Peer', department_id: 'd-1', department_name: 'Eng',
    role_id: 'r-da', role_name: 'department_admin', status: 'active', force_password_change: false, is_locked: false, created_at: '' },
]

const server = setupServer(
  http.get('/api/v1/departments', () => HttpResponse.json({ data: [], error: null })),
  http.get('/api/v1/users/roles', () => HttpResponse.json({ data: roles, error: null })),
  http.get('/api/v1/users', () =>
    HttpResponse.json({ data: { items: listed, meta: { page: 1, per_page: 20, total: 2 } }, error: null })),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function renderPage(role: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], `h.${btoa(JSON.stringify({ role, sub: 'me-1' }))}.s`)
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <UsersListPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('UsersListPage role dropdown (department_admin)', () => {
  it('create drawer offers only examiner and employee', async () => {
    renderPage('department_admin')
    await userEvent.click(await screen.findByRole('button', { name: 'New User' }))
    await waitFor(() => expect(screen.getByRole('option', { name: 'employee' })).toBeInTheDocument())
    expect(screen.getByRole('option', { name: 'examiner' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'department_admin' })).toBeNull()
    expect(screen.queryByRole('option', { name: 'super_admin' })).toBeNull()
  })

  it('edit drawer for an employee offers only examiner and employee', async () => {
    renderPage('department_admin')
    await screen.findByText('Emma Employee')
    await userEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0])
    await waitFor(() => expect(screen.getByRole('option', { name: 'examiner' })).toBeInTheDocument())
    expect(screen.queryByRole('option', { name: 'super_admin' })).toBeNull()
    expect(screen.queryByRole('option', { name: 'department_admin' })).toBeNull()
  })

  it('edit drawer for a peer department_admin shows the role read-only', async () => {
    renderPage('department_admin')
    await screen.findByText('Dana Peer')
    await userEvent.click(screen.getAllByRole('button', { name: 'Edit' })[1])
    const current = (await screen.findByRole('option', { name: 'department_admin' })) as HTMLOptionElement
    const select = current.closest('select') as HTMLSelectElement
    expect(select).toBeDisabled()
    expect(select.value).toBe('r-da')
    expect(screen.queryByRole('option', { name: 'super_admin' })).toBeNull()
  })
})
