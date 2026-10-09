import { describe, it, expect } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { PasswordChangeGuard } from './PasswordChangeGuard'
import { markPasswordChangeRequired, clearPasswordChangeRequired } from '@/lib/passwordChangeRequired'

const user = (force: boolean) => ({ id: 'u', email: 'a@b.c', role: 'employee', full_name: 'A', force_password_change: force })

function setup(qc: QueryClient, path: string) {
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <PasswordChangeGuard />
        <Routes>
          <Route path="/portal" element={<div>portal page</div>} />
          <Route path="/admin" element={<div>admin page</div>} />
          <Route path="/change-password" element={<div>change password page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('PasswordChangeGuard (ISS-160)', () => {
  it('redirects a user with force_password_change to /change-password', async () => {
    const qc = new QueryClient()
    qc.setQueryData(['auth', 'currentUser'], user(true))
    setup(qc, '/admin')
    await waitFor(() => expect(screen.getByText('change password page')).toBeInTheDocument())
  })

  it('redirects when the backend 403 flag is raised, even without a currentUser flag', async () => {
    const qc = new QueryClient()
    qc.setQueryData(['auth', 'currentUser'], user(false))
    setup(qc, '/portal')
    expect(screen.getByText('portal page')).toBeInTheDocument()
    act(() => markPasswordChangeRequired(qc))
    await waitFor(() => expect(screen.getByText('change password page')).toBeInTheDocument())
  })

  it('does nothing for a normal user and stays put on /change-password', async () => {
    const qc = new QueryClient()
    qc.setQueryData(['auth', 'currentUser'], user(false))
    setup(qc, '/portal')
    expect(screen.getByText('portal page')).toBeInTheDocument()

    const qc2 = new QueryClient()
    qc2.setQueryData(['auth', 'currentUser'], user(true))
    setup(qc2, '/change-password')
    expect(screen.getAllByText('change password page').length).toBeGreaterThan(0)
  })

  it('clearing the flag stops redirecting', () => {
    const qc = new QueryClient()
    markPasswordChangeRequired(qc)
    clearPasswordChangeRequired(qc)
    setup(qc, '/portal')
    expect(screen.getByText('portal page')).toBeInTheDocument()
  })
})
