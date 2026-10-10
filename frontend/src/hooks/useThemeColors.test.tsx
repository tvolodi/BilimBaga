import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render } from '@testing-library/react'
import { ThemeProvider, THEME_STORAGE_KEY, useTheme } from '@/components/ThemeProvider'
import { TenantProvider } from '@/components/TenantProvider'
import { useThemeColors, type ThemeColours } from './useThemeColors'

// jsdom does not load index.css, so the computed tokens are stubbed per theme.
const state = vi.hoisted(() => ({
  config: undefined as { primary_color: string; accent_color: string } | undefined,
  primary: '#2e6db4',
}))

vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: () => ({ data: state.config }),
}))

const TOKENS: Record<'light' | 'dark', Record<string, string>> = {
  light: {
    '--color-border-default': '#dde2ed',
    '--color-text-secondary': '#4a5568',
    '--color-text-muted': '#5f6b7d',
  },
  dark: {
    '--color-border-default': '#2a3550',
    '--color-text-secondary': '#9aaabb',
    '--color-text-muted': '#8fa0b4',
  },
}

beforeEach(() => {
  state.config = undefined
  state.primary = '#2e6db4'
  vi.spyOn(window, 'getComputedStyle').mockImplementation(
    () =>
      ({
        getPropertyValue: (name: string) => {
          if (name === '--color-primary') return state.primary
          const theme = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
          return TOKENS[theme][name] ?? ''
        },
      }) as unknown as CSSStyleDeclaration,
  )
})

afterEach(() => {
  vi.restoreAllMocks()
  window.localStorage.removeItem(THEME_STORAGE_KEY)
  document.documentElement.classList.remove('dark')
  document.getElementById('tenant-branding')?.remove()
  document.documentElement.removeAttribute('style')
})

const latest: { colours?: ThemeColours; theme?: ReturnType<typeof useTheme> } = {}

function Probe() {
  latest.colours = useThemeColors()
  latest.theme = useTheme()
  return null
}

function tree() {
  return (
    <ThemeProvider switchEnabled>
      <TenantProvider>
        <Probe />
      </TenantProvider>
    </ThemeProvider>
  )
}

describe('useThemeColors (FR-BB321 AC-10)', () => {
  it('reads the light tokens by default', () => {
    render(tree())
    expect(latest.colours).toEqual({
      primary: '#2e6db4',
      border: '#dde2ed',
      textSecondary: '#4a5568',
      textMuted: '#5f6b7d',
    })
  })

  it('reads the dark tokens after the theme switches', () => {
    render(tree())
    state.primary = '#5aa0e0'
    act(() => latest.theme?.setPreference('dark'))
    expect(latest.colours).toEqual({
      primary: '#5aa0e0',
      border: '#2a3550',
      textSecondary: '#9aaabb',
      textMuted: '#8fa0b4',
    })
  })

  it('re-reads the primary when the tenant overrides change, even without a theme change', () => {
    state.config = { primary_color: '#1b3a6b', accent_color: '#c8a84b' }
    const view = render(tree())
    expect(latest.colours?.primary).toBe('#2e6db4')

    state.primary = '#1b3a6b'
    state.config = { primary_color: '#1b3a6b', accent_color: '#c8a84b' }
    view.rerender(tree())
    expect(latest.colours?.primary).toBe('#1b3a6b')
  })
})
