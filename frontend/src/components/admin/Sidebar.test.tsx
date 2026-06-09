import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { Sidebar } from './Sidebar'

vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: vi.fn(),
}))

import { useTenantConfig } from '@/api/useTenantConfig'

function renderSidebar(collapsed = false) {
  const onToggle = vi.fn()
  render(
    <MemoryRouter>
      <Sidebar collapsed={collapsed} onToggle={onToggle} />
    </MemoryRouter>,
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
})
