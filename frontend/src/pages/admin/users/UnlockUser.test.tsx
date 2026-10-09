import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { UsersListPage } from './UsersListPage'

function user(id: string, name: string, isLocked: boolean) {
  return {
    id,
    email: `${id}@example.com`,
    full_name: name,
    department_id: null,
    department_name: null,
    role_id: 'role-emp',
    role_name: 'employee',
    status: 'active',
    force_password_change: false,
    is_locked: isLocked,
    created_at: '2024-01-01T00:00:00Z',
  }
}

let lockedState = true
let unlockCalls: string[] = []
let unlockStatus = 200
let unlockCode = 'INTERNAL_ERROR'
let resetStatus = 200

const server = setupServer(
  http.get('/api/v1/departments', () => HttpResponse.json({ data: [], error: null })),
  http.get('/api/v1/users/roles', () => HttpResponse.json({ data: [], error: null })),
  http.get('/api/v1/users', () =>
    HttpResponse.json({
      data: {
        items: [user('u-locked', 'Locked Larry', lockedState), user('u-free', 'Free Fiona', false)],
        meta: { page: 1, per_page: 20, total: 2 },
      },
      error: null,
    }),
  ),
  http.post('/api/v1/users/:id/reset-password', () => {
    if (resetStatus !== 200) {
      return HttpResponse.json(
        { data: null, error: { code: resetStatus === 403 ? 'FORBIDDEN' : 'INTERNAL_ERROR', message: 'x' } },
        { status: resetStatus },
      )
    }
    return HttpResponse.json({ data: { temporary_password: 'Tmp-Pass-123' }, error: null })
  }),
  http.post('/api/v1/users/:id/unlock', ({ params }) => {
    unlockCalls.push(String(params.id))
    if (unlockStatus !== 200) {
      return HttpResponse.json(
        { data: null, error: { code: unlockCode, message: 'boom' } },
        { status: unlockStatus },
      )
    }
    lockedState = false
    return HttpResponse.json({ data: user(String(params.id), 'Locked Larry', false), error: null })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  lockedState = true
  unlockCalls = []
  unlockStatus = 200
  unlockCode = 'INTERNAL_ERROR'
  resetStatus = 200
})
afterAll(() => server.close())

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <UsersListPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('UsersListPage unlock action (FR-BB115 AC-8)', () => {
  it('shows "Unlock account" only for locked users', async () => {
    renderPage()
    await screen.findByText('Locked Larry')
    expect(screen.getAllByRole('button', { name: /unlock/i })).toHaveLength(1)
    expect(screen.getByText('Locked')).toBeInTheDocument()
  })

  it('calls the unlock endpoint, shows a success notice and refreshes the row', async () => {
    const u = userEvent.setup()
    renderPage()
    await screen.findByText('Locked Larry')

    await u.click(screen.getByRole('button', { name: /unlock/i }))

    expect(await screen.findByRole('status')).toHaveTextContent(/account unlocked/i)
    expect(unlockCalls).toEqual(['u-locked'])
    await waitFor(() => expect(screen.queryByRole('button', { name: /unlock/i })).not.toBeInTheDocument())
  })

  it('shows an error notice when unlock fails and keeps the action available', async () => {
    unlockStatus = 500
    const u = userEvent.setup()
    renderPage()
    await screen.findByText('Locked Larry')

    await u.click(screen.getByRole('button', { name: /unlock/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/could not unlock/i)
    expect(screen.getByRole('button', { name: /unlock/i })).toBeInTheDocument()
  })

  it('shows the localized not-allowed message when unlock is forbidden', async () => {
    unlockStatus = 403
    unlockCode = 'FORBIDDEN'
    const u = userEvent.setup()
    renderPage()
    await screen.findByText('Locked Larry')
    await u.click(screen.getByRole('button', { name: /unlock/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/not allowed/i)
  })
})

describe('UsersListPage reset password errors (ISS-226)', () => {
  it('shows the not-allowed message on 403', async () => {
    resetStatus = 403
    const u = userEvent.setup()
    renderPage()
    await screen.findByText('Free Fiona')
    await u.click(screen.getAllByRole('button', { name: /reset/i })[0])
    expect(await screen.findByRole('alert')).toHaveTextContent(/not allowed/i)
  })

  it('shows a generic message on 500', async () => {
    resetStatus = 500
    const u = userEvent.setup()
    renderPage()
    await screen.findByText('Free Fiona')
    await u.click(screen.getAllByRole('button', { name: /reset/i })[0])
    expect(await screen.findByRole('alert')).toHaveTextContent(/an error occurred/i)
  })

  it('shows the temporary password on success', async () => {
    const u = userEvent.setup()
    renderPage()
    await screen.findByText('Free Fiona')
    await u.click(screen.getAllByRole('button', { name: /reset/i })[0])
    expect(await screen.findByText('Tmp-Pass-123')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
