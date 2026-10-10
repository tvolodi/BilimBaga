import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import '@/i18n'
import i18n from '@/i18n'
import { LoginPage } from './LoginPage'
import { SESSION_REVOKED_KEY } from '@/lib/sessionRevoked'

const { mutateAsync } = vi.hoisted(() => ({ mutateAsync: vi.fn() }))

vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: () => ({ data: { app_name: 'BilimBaga', available_locales: ['en'] }, isLoading: false }),
}))

vi.mock('@/api/auth', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/auth')>()),
  useLogin: () => ({ mutateAsync, isPending: false, error: null }),
}))

function renderLogin(revoked: boolean) {
  const qc = new QueryClient()
  if (revoked) qc.setQueryData(SESSION_REVOKED_KEY, true)
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return qc
}

/** Renders /login as if a guard had redirected here with `from` in location state (AC-11). */
function renderLoginFrom(from: string) {
  const qc = new QueryClient()
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[{ pathname: '/login', state: { from } }]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/admin" element={<div>Admin Home</div>} />
          <Route path="/admin/departments" element={<div>Departments Page</div>} />
          <Route path="/portal/results" element={<div>Results Page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function submitCredentials() {
  fireEvent.change(screen.getByLabelText(i18n.t('auth.login.emailLabel')), { target: { value: 'user@example.com' } })
  fireEvent.change(screen.getByLabelText(i18n.t('auth.login.passwordLabel')), { target: { value: 'Secret-pass-1' } })
  fireEvent.click(screen.getByRole('button', { name: i18n.t('auth.login.submitButton') }))
}

describe('LoginPage session-revoked notice (ISS-249)', () => {
  it('shows the localized notice after a TOKEN_REVOKED logout', async () => {
    await i18n.changeLanguage('en')
    renderLogin(true)
    expect(screen.getByRole('alert')).toHaveTextContent(i18n.t('auth.login.sessionRevoked'))
    expect(i18n.t('auth.login.sessionRevoked')).not.toBe('auth.login.sessionRevoked')
  })

  it('shows no notice on a normal visit', async () => {
    await i18n.changeLanguage('en')
    renderLogin(false)
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('can be dismissed with its button (AC-11)', async () => {
    await i18n.changeLanguage('en')
    const qc = renderLogin(true)
    const alert = screen.getByRole('alert')
    fireEvent.click(alert.querySelector('button') as HTMLButtonElement)
    expect(screen.queryByRole('alert')).toBeNull()
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBe(false)
  })
  // #437: the dismiss control is the components/ui Button in the warning colour, not a raw button.
  it('dismisses with the ui Button in the warning colour (#437)', async () => {
    await i18n.changeLanguage('en')
    renderLogin(true)
    const dismiss = screen.getByRole('alert').querySelector('button') as HTMLButtonElement
    expect(dismiss.className).toMatch(/focus-visible:ring-2/)
    expect(dismiss.className).toMatch(/text-warning/)
    expect(dismiss.className).not.toMatch(/#/)
  })
})

describe('LoginPage return path after a revoked session (AC-11)', () => {
  it('restores the requested path after signing in again', async () => {
    await i18n.changeLanguage('en')
    mutateAsync.mockResolvedValueOnce({ user: { role: 'super_admin', force_password_change: false } })
    renderLoginFrom('/admin/departments')
    submitCredentials()
    expect(await screen.findByText('Departments Page')).toBeInTheDocument()
    expect(screen.queryByText('Admin Home')).toBeNull()
  })

  it('restores a portal path for an employee', async () => {
    await i18n.changeLanguage('en')
    mutateAsync.mockResolvedValueOnce({ user: { role: 'employee', force_password_change: false } })
    renderLoginFrom('/portal/results')
    submitCredentials()
    expect(await screen.findByText('Results Page')).toBeInTheDocument()
  })

  it('ignores a return target that is not an in-app path', async () => {
    await i18n.changeLanguage('en')
    mutateAsync.mockResolvedValueOnce({ user: { role: 'super_admin', force_password_change: false } })
    renderLoginFrom('//example.com/phish')
    submitCredentials()
    expect(await screen.findByText('Admin Home')).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText('Departments Page')).toBeNull())
  })
})
