import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { TopBar } from './TopBar'
import type { User } from '@/api/users'

vi.mock('@/api/auth', () => ({
  useLogout: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

const user: User = {
  id: 'u1',
  email: 'dept@example.com',
  full_name: 'Dana Admin',
  department_id: 'd1',
  department_name: 'Engineering',
  role_id: 'r1',
  role_name: 'department_admin',
  status: 'active',
  force_password_change: false,
  is_locked: false,
  created_at: '2026-01-01T00:00:00Z',
}

function renderTopBar() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <TopBar user={user} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

// FR-BB116 AC-5: the admin top bar exposes a link to the profile page from the user name.
describe('TopBar profile link', () => {
  it('links the user name to the admin profile page', () => {
    renderTopBar()
    expect(screen.getByRole('link', { name: 'Dana Admin' })).toHaveAttribute('href', '/admin/profile')
  })
})
