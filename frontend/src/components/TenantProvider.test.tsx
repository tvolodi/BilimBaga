import { afterEach, describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { useTenantConfig } from '@/api/useTenantConfig'
import { ThemeProvider, THEME_STORAGE_KEY } from '@/components/ThemeProvider'
import { TenantProvider, useTenantVersion } from './TenantProvider'

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
  it('writes the tenant primary for the light theme under :root:not(.dark)', () => {
    mockConfig('#123456')
    render(<TenantProvider><span /></TenantProvider>)

    expect(styleText()).toContain(':root:not(.dark) { --color-primary: #123456; --color-primary-foreground: #ffffff; }')
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

describe('TenantProvider in the dark theme (FR-BB321 AC-8)', () => {
  afterEach(() => {
    window.localStorage.removeItem(THEME_STORAGE_KEY)
    document.documentElement.classList.remove('dark')
  })

  it('writes the derived dark primary for the dark theme only', () => {
    mockConfig('#1b3a6b')
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    render(
      <ThemeProvider switchEnabled>
        <TenantProvider><span /></TenantProvider>
      </ThemeProvider>,
    )

    expect(styleText()).toContain(':root.dark { --color-primary: #6c97da; --color-primary-foreground: #0f1623; }')
    expect(styleText()).toContain(':root:not(.dark) { --color-primary: #1b3a6b; --color-primary-foreground: #ffffff; }')
  })

  it('leaves the dark pair in force for the design default primary', () => {
    mockConfig('#2e6db4')
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    render(
      <ThemeProvider switchEnabled>
        <TenantProvider><span /></TenantProvider>
      </ThemeProvider>,
    )

    expect(styleText()).not.toContain(':root.dark')
  })

  it('applies the derived accent inline while dark', () => {
    mockConfig('#1b3a6b', '#b91c1c')
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    render(
      <ThemeProvider switchEnabled>
        <TenantProvider><span /></TenantProvider>
      </ThemeProvider>,
    )

    expect(document.documentElement.style.getPropertyValue('--color-accent')).toBe('#e96d6d')
  })

  it('bumps the version that consumers watch, so they re-read the overrides', () => {
    mockConfig('#123456')
    let seen = -1
    function Probe() {
      seen = useTenantVersion()
      return null
    }
    render(<TenantProvider><Probe /></TenantProvider>)

    expect(seen).toBeGreaterThan(0)
  })
})

describe('TenantProvider accent foreground (FR-BB321 AC-8)', () => {
  afterEach(() => {
    document.documentElement.removeAttribute('style')
  })

  it('sets the foreground of an overridden accent in the light theme', () => {
    mockConfig('#123456', '#c8a84b')
    render(<TenantProvider><span /></TenantProvider>)

    expect(document.documentElement.style.getPropertyValue('--color-accent-foreground')).toBe('#0f1623')
  })

  it('removes the accent and its foreground when the accent is not set', () => {
    mockConfig('#123456', '#c8a84b')
    const view = render(<TenantProvider><span /></TenantProvider>)
    expect(document.documentElement.style.getPropertyValue('--color-accent-foreground')).toBe('#0f1623')

    vi.mocked(useTenantConfig).mockReturnValue({
      data: { app_name: 'Acme', primary_color: '#123456', accent_color: '', default_locale: 'en', available_locales: ['en'] },
    } as unknown as ReturnType<typeof useTenantConfig>)
    view.rerender(<TenantProvider><span /></TenantProvider>)

    expect(document.documentElement.style.getPropertyValue('--color-accent')).toBe('')
    expect(document.documentElement.style.getPropertyValue('--color-accent-foreground')).toBe('')
  })
})
