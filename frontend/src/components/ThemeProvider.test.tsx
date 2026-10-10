import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, render } from '@testing-library/react'
import {
  ThemeProvider,
  THEME_STORAGE_KEY,
  THEME_SWITCH_ENABLED,
  useTheme,
  type ThemePreference,
} from './ThemeProvider'

type ThemeValue = ReturnType<typeof useTheme>
type Listener = (event: MediaQueryListEvent) => void

const DARK_QUERY = '(prefers-color-scheme: dark)'

interface SystemThemeStub {
  emit: (prefersDark: boolean) => void
  addEventListener: ReturnType<typeof vi.fn>
  removeEventListener: ReturnType<typeof vi.fn>
  listeners: Set<Listener>
}

// jsdom has no matchMedia. Each test installs a stub for the dark query and can fire OS changes.
function stubSystemTheme(prefersDark: boolean): SystemThemeStub {
  const listeners = new Set<Listener>()
  let matches = prefersDark
  const addEventListener = vi.fn((_type: string, listener: Listener) => {
    listeners.add(listener)
  })
  const removeEventListener = vi.fn((_type: string, listener: Listener) => {
    listeners.delete(listener)
  })
  window.matchMedia = vi.fn(
    (query: string) =>
      ({
        get matches() {
          return matches
        },
        media: query,
        addEventListener,
        removeEventListener,
      }) as unknown as MediaQueryList,
  )
  return {
    listeners,
    addEventListener,
    removeEventListener,
    emit(next: boolean) {
      matches = next
      for (const listener of [...listeners]) {
        listener({ matches: next, media: DARK_QUERY } as MediaQueryListEvent)
      }
    },
  }
}

// Renders a probe under the provider. It records the context value and what <html> looked like on
// every render, so a test can check the class was already in place when React re-rendered.
function renderProvider(storedValue?: string, switchEnabled = true) {
  if (storedValue !== undefined) window.localStorage.setItem(THEME_STORAGE_KEY, storedValue)
  const latest: { current?: ThemeValue } = {}
  const renders: { resolved: string; htmlDark: boolean }[] = []
  function Probe() {
    const theme = useTheme()
    latest.current = theme
    renders.push({
      resolved: theme.resolved,
      htmlDark: document.documentElement.classList.contains('dark'),
    })
    return null
  }
  const view = render(
    <ThemeProvider switchEnabled={switchEnabled}>
      <Probe />
    </ThemeProvider>,
  )
  const current = () => {
    if (!latest.current) throw new Error('Probe has not rendered')
    return latest.current
  }
  const choose = (value: ThemePreference) => {
    act(() => {
      current().setPreference(value)
    })
  }
  return { ...view, renders, current, choose }
}

afterEach(() => {
  delete (window as { matchMedia?: unknown }).matchMedia
  window.localStorage.clear()
  document.documentElement.classList.remove('dark')
  document.documentElement.style.colorScheme = ''
  vi.restoreAllMocks()
})

describe('ThemeProvider defaults and storage', () => {
  it('defaults to system and resolves light when the OS is light and nothing is stored', () => {
    stubSystemTheme(false)
    const view = renderProvider()
    expect(view.current().preference).toBe('system')
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('resolves dark for system when the OS is dark', () => {
    stubSystemTheme(true)
    const view = renderProvider()
    expect(view.current().preference).toBe('system')
    expect(view.current().resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
  })

  it('applies a stored dark preference even when the OS is light', () => {
    stubSystemTheme(false)
    const view = renderProvider('dark')
    expect(view.current().preference).toBe('dark')
    expect(view.current().resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')
  })

  it('applies a stored light preference even when the OS is dark', () => {
    stubSystemTheme(true)
    const view = renderProvider('light')
    expect(view.current().preference).toBe('light')
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(document.documentElement.style.colorScheme).toBe('light')
  })

  it('ignores an unknown stored value and treats it as system', () => {
    stubSystemTheme(true)
    const view = renderProvider('purple')
    expect(view.current().preference).toBe('system')
    expect(view.current().resolved).toBe('dark')
  })

  it('keeps working when storage throws on read and write', () => {
    stubSystemTheme(true)
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('storage blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('storage blocked')
    })
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new Error('storage blocked')
    })
    const view = renderProvider()
    expect(view.current().preference).toBe('system')
    expect(view.current().resolved).toBe('dark')

    expect(() => view.choose('light')).not.toThrow()
    expect(view.current().preference).toBe('light')
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('stores an explicit choice and removes the key when system is chosen', () => {
    stubSystemTheme(false)
    const view = renderProvider()
    view.choose('dark')
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')
    view.choose('system')
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBeNull()
    expect(view.current().preference).toBe('system')
  })

  it('works where matchMedia is not available', () => {
    const view = renderProvider()
    expect(view.current().preference).toBe('system')
    expect(view.current().resolved).toBe('light')
  })
})

describe('ThemeProvider while the switch is off (FR-BB321 part 1)', () => {
  it('ships with the switch off', () => {
    expect(THEME_SWITCH_ENABLED).toBe(false)
  })

  it('resolves light for a stored dark value and never adds the dark class', () => {
    stubSystemTheme(true)
    const view = renderProvider('dark', false)
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(document.documentElement.style.colorScheme).toBe('light')
  })

  it('stays light when the OS prefers dark and the preference is system', () => {
    stubSystemTheme(true)
    const view = renderProvider(undefined, false)
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('stays light after a dark choice, while still storing the choice', () => {
    stubSystemTheme(false)
    const view = renderProvider(undefined, false)
    view.choose('dark')
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')
  })

  it('exposes switchEnabled as false', () => {
    stubSystemTheme(false)
    expect(renderProvider(undefined, false).current().switchEnabled).toBe(false)
  })

  it('does not subscribe to OS changes', () => {
    const system = stubSystemTheme(false)
    renderProvider(undefined, false)
    expect(system.addEventListener).not.toHaveBeenCalled()
  })
})

describe('ThemeProvider OS changes', () => {
  it('follows OS changes while the preference is system', () => {
    const os = stubSystemTheme(false)
    const view = renderProvider()
    expect(view.current().resolved).toBe('light')

    act(() => os.emit(true))
    expect(view.current().resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(document.documentElement.style.colorScheme).toBe('dark')

    act(() => os.emit(false))
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('ignores OS changes when an explicit preference is stored', () => {
    const os = stubSystemTheme(false)
    const view = renderProvider('light')
    expect(os.addEventListener).not.toHaveBeenCalled()

    act(() => os.emit(true))
    expect(view.current().preference).toBe('light')
    expect(view.current().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('ignores OS changes after an explicit choice made while on system', () => {
    const os = stubSystemTheme(false)
    const view = renderProvider()
    view.choose('dark')
    act(() => os.emit(false))
    expect(view.current().preference).toBe('dark')
    expect(view.current().resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('removes the matchMedia listener on unmount', () => {
    const os = stubSystemTheme(false)
    const view = renderProvider()
    expect(os.addEventListener).toHaveBeenCalledTimes(1)
    const [, registered] = os.addEventListener.mock.calls[0] as [string, Listener]

    view.unmount()

    expect(os.removeEventListener).toHaveBeenCalledWith('change', registered)
    expect(os.listeners.size).toBe(0)
  })

  it('applies the class before React state is updated when the user switches', () => {
    stubSystemTheme(false)
    const view = renderProvider('light')
    view.renders.length = 0

    view.choose('dark')

    expect(view.renders[view.renders.length - 1]).toEqual({ resolved: 'dark', htmlDark: true })
  })

  it('applies the class before React state is updated when the OS changes', () => {
    const os = stubSystemTheme(false)
    const view = renderProvider()
    view.renders.length = 0

    act(() => os.emit(true))

    expect(view.renders[view.renders.length - 1]).toEqual({ resolved: 'dark', htmlDark: true })
  })
})

describe('useTheme outside a provider', () => {
  it('returns the default value and a setter that does nothing', () => {
    const latest: { current?: ThemeValue } = {}
    function Probe() {
      latest.current = useTheme()
      return null
    }
    render(<Probe />)
    expect(latest.current?.preference).toBe('system')
    expect(latest.current?.resolved).toBe('light')

    expect(() => latest.current?.setPreference('dark')).not.toThrow()
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBeNull()
  })
})
