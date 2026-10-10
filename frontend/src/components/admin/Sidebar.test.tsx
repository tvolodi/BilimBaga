import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/i18n'
import { Sidebar } from './Sidebar'

vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: vi.fn(),
}))

import { useTenantConfig } from '@/api/useTenantConfig'

function tokenFor(role: string): string {
  const b64 = (o: object) => btoa(JSON.stringify(o)).replace(/=/g, '')
  return `${b64({ alg: 'HS256', typ: 'JWT' })}.${b64({ sub: 'u', role, exp: 9999999999 })}.sig`
}

function renderSidebar(collapsed = false, role = 'super_admin') {
  const onToggle = vi.fn()
  const qc = new QueryClient()
  qc.setQueryData(['auth', 'accessToken'], tokenFor(role))
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <Sidebar collapsed={collapsed} onToggle={onToggle} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return { onToggle }
}

describe('Sidebar', () => {
  beforeEach(() => {
    vi.mocked(useTenantConfig).mockReturnValue({
      data: { app_name: 'BilimBaga', primary_color: '#000', accent_color: '#fff', default_locale: 'en', available_locales: ['en'] },
    } as ReturnType<typeof useTenantConfig>)
  })

  it('renders all navigation links when expanded', () => {
    renderSidebar(false)
    // Should have nav links for key sections
    expect(screen.getByRole('link', { name: /users/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /departments/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /questions/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /categories/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /tags/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /settings/i })).toBeInTheDocument()
  })

  it('calls onToggle when the collapse button is clicked', async () => {
    const { onToggle } = renderSidebar(false)
    const toggleButton = screen.getByRole('button')
    await userEvent.click(toggleButton)
    expect(onToggle).toHaveBeenCalledOnce()
  })

  it('uses the navy tokens for the surface and no default-palette grey (FR-BB320 AC-2)', () => {
    renderSidebar(false)
    const aside = document.querySelector('aside')
    expect(aside).toHaveClass('bg-bg-navy', 'text-text-on-navy')
    expect(aside?.innerHTML).not.toMatch(/gray-\d/)
  })

  it('renders in collapsed state without crashing', () => {
    renderSidebar(true)
    // Sidebar still renders
    expect(document.body).toBeInTheDocument()
  })

  it('nav links point to correct paths', () => {
    renderSidebar(false)
    const usersLink = screen.getByRole('link', { name: /users/i })
    expect(usersLink).toHaveAttribute('href', '/admin/users')

    const settingsLink = screen.getByRole('link', { name: /settings/i })
    expect(settingsLink).toHaveAttribute('href', '/admin/settings/branding')
  })

  it('displays app_name from tenant config instead of hardcoded literal', () => {
    vi.mocked(useTenantConfig).mockReturnValue({
      data: { app_name: 'AcmeCorp', primary_color: '#000', accent_color: '#fff', default_locale: 'en', available_locales: ['en'] },
    } as ReturnType<typeof useTenantConfig>)

    renderSidebar(false)
    expect(screen.getByText('AcmeCorp')).toBeInTheDocument()
    expect(screen.queryByText('BilimBaga')).not.toBeInTheDocument()
  })

  it('falls back to "BilimBaga" when tenant config is not yet loaded', () => {
    vi.mocked(useTenantConfig).mockReturnValue({
      data: undefined,
    } as ReturnType<typeof useTenantConfig>)

    renderSidebar(false)
    expect(screen.getByText('BilimBaga')).toBeInTheDocument()
  })

  it.each([
    ['super_admin', true, true],
    ['department_admin', false, true],
    ['examiner', false, true],
    ['employee', false, false],
  ])('role %s: audit link=%s, reports link=%s', (role, audit, reports) => {
    renderSidebar(false, role)
    const hrefs = screen.getAllByRole('link').map((a) => a.getAttribute('href'))
    expect(hrefs.includes('/admin/audit')).toBe(audit)
    expect(hrefs.includes('/admin/reports')).toBe(reports)
    expect(hrefs).toContain('/admin/users')
  })
})

describe('Sidebar overlay and navigation (FR-BB321 AC-6, #472)', () => {
  beforeEach(() => {
    vi.mocked(useTenantConfig).mockReturnValue({
      data: { app_name: 'BilimBaga', primary_color: '#000', accent_color: '#fff', default_locale: 'en', available_locales: ['en'] },
    } as ReturnType<typeof useTenantConfig>)
  })

  function renderWith(props: { overlay?: boolean; onNavigate?: () => void }) {
    const qc = new QueryClient()
    qc.setQueryData(['auth', 'accessToken'], tokenFor('super_admin'))
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <Sidebar collapsed={false} onToggle={vi.fn()} {...props} />
        </MemoryRouter>
      </QueryClientProvider>,
    )
  }

  it('floats over the page when overlay is set', () => {
    renderWith({ overlay: true })
    expect(document.querySelector('aside')).toHaveClass('fixed', 'z-40')
  })

  it('takes its width in the layout when overlay is not set', () => {
    renderWith({})
    expect(document.querySelector('aside')).not.toHaveClass('fixed')
  })

  it('calls onNavigate when a link is followed, so an overlay can close', async () => {
    const onNavigate = vi.fn()
    renderWith({ overlay: true, onNavigate })
    await userEvent.click(screen.getByRole('link', { name: /users/i }))
    expect(onNavigate).toHaveBeenCalledTimes(1)
  })
})
