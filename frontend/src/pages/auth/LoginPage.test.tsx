import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import i18n from '@/i18n'
import { LoginPage } from './LoginPage'
import { SESSION_REVOKED_KEY } from '@/lib/sessionRevoked'

vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: () => ({ data: { app_name: 'BilimBaga', available_locales: ['en'] }, isLoading: false }),
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
})
