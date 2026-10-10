import { afterEach, describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { useTenantConfig } from '@/api/useTenantConfig'
import { TenantProvider } from './TenantProvider'

vi.mock('@/api/useTenantConfig', () => ({ useTenantConfig: vi.fn() }))

function mockConfig(primary: string, accent = '#c8a84b') {
  vi.mocked(useTenantConfig).mockReturnValue({
    data: {
      app_name: 'Acme',
      primary_color: primary,
      accent_color: accent,
      default_locale: 'en',
      available_locales: ['en'],
    },
  } as unknown as ReturnType<typeof useTenantConfig>)
}

function styleText(): string {
  return document.getElementById('tenant-branding')?.textContent ?? ''
}

afterEach(() => {
  document.getElementById('tenant-branding')?.remove()
  document.documentElement.removeAttribute('style')
})

describe('TenantProvider', () => {
  it('scopes the tenant primary to the light theme only', () => {
    mockConfig('#123456')
    render(<TenantProvider><span /></TenantProvider>)

    expect(styleText()).toBe(':root:not(.dark) { --color-primary: #123456; }')
  })

  it('does not write the primary as an inline root style, so the dark pair is not overridden', () => {
    mockConfig('#123456')
    render(<TenantProvider><span /></TenantProvider>)

    expect(document.documentElement.style.getPropertyValue('--color-primary')).toBe('')
  })

  it('keeps applying the tenant accent to the root element', () => {
    mockConfig('#123456', '#c8a84b')
    render(<TenantProvider><span /></TenantProvider>)

    expect(document.documentElement.style.getPropertyValue('--color-accent')).toBe('#c8a84b')
  })

  it('writes no primary rule when the stored value is not a hex colour', () => {
    mockConfig('red} body { display: none')
    render(<TenantProvider><span /></TenantProvider>)

    expect(styleText()).toBe('')
  })
})
