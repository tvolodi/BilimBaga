import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

export type ThemePreference = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

/** FR-BB321 D-2: the choice is stored on the device only (no account sync). */
export const THEME_STORAGE_KEY = 'bb-theme'
const DARK_QUERY = '(prefers-color-scheme: dark)'

/**
 * FR-BB321 switch. On: the three-way theme choice is live (part 3, after the nginx change of AC-4 and
 * the class migration of AC-13). While false the resolved theme is always light and the toggle is hidden.
 * Keep `public/theme-init.js` in step with this value.
 */
export const THEME_SWITCH_ENABLED = true

interface ThemeContextValue {
  preference: ThemePreference
  resolved: ResolvedTheme
  setPreference: (preference: ThemePreference) => void
  /** False while the switch is off: the toggle is hidden and the page stays light. */
  switchEnabled: boolean
}

// Outside a provider the app still renders: light theme, and the setter does nothing.
const ThemeContext = createContext<ThemeContextValue>({
  preference: 'system',
  resolved: 'light',
  setPreference: () => {},
  switchEnabled: THEME_SWITCH_ENABLED,
})

function isPreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

function readStoredPreference(): ThemePreference {
  try {
    const value = window.localStorage.getItem(THEME_STORAGE_KEY)
    return isPreference(value) ? value : 'system'
  } catch {
    return 'system' // storage blocked: follow the OS
  }
}

function writeStoredPreference(preference: ThemePreference) {
  try {
    if (preference === 'system') window.localStorage.removeItem(THEME_STORAGE_KEY)
    else window.localStorage.setItem(THEME_STORAGE_KEY, preference)
  } catch {
    // Storage blocked or full: the choice still applies for this page view.
  }
}

function systemPrefersDark(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(DARK_QUERY).matches
}

function resolveTheme(
  preference: ThemePreference,
  systemDark: boolean,
  switchEnabled: boolean,
): ResolvedTheme {
  if (!switchEnabled) return 'light'
  if (preference === 'system') return systemDark ? 'dark' : 'light'
  return preference
}

/** Writes the theme to <html>: the `dark` class and the native colour scheme (FR-BB321 D-3). */
function applyTheme(resolved: ResolvedTheme) {
  const root = document.documentElement
  root.classList.toggle('dark', resolved === 'dark')
  root.style.colorScheme = resolved
}

export function ThemeProvider({
  children,
  switchEnabled = THEME_SWITCH_ENABLED,
}: {
  children: ReactNode
  /** Defaults to THEME_SWITCH_ENABLED; tests pass it explicitly to cover both states. */
  switchEnabled?: boolean
}) {
  const [preference, setPreferenceState] = useState<ThemePreference>(readStoredPreference)
  const [systemDark, setSystemDark] = useState<boolean>(systemPrefersDark)
  const resolved = resolveTheme(preference, systemDark, switchEnabled)

  // Keeps <html> in step with the resolved theme on mount and after any change. The handlers
  // below also apply the theme before they set state (AC-2); this effect only covers the mount.
  useLayoutEffect(() => {
    applyTheme(resolved)
  }, [resolved])

  // Only while `system` and the switch is on: an explicit choice ignores later OS changes.
  useEffect(() => {
    if (!switchEnabled || preference !== 'system' || typeof window.matchMedia !== 'function') return
    const query = window.matchMedia(DARK_QUERY)
    const onChange = (event: MediaQueryListEvent) => {
      // The class changes before React state, so a consumer that re-renders sees the new theme.
      applyTheme(resolveTheme('system', event.matches, switchEnabled))
      setSystemDark(event.matches)
    }
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  }, [preference, switchEnabled])

  const setPreference = useCallback(
    (next: ThemePreference) => {
      const systemIsDark = systemPrefersDark()
      writeStoredPreference(next)
      applyTheme(resolveTheme(next, systemIsDark, switchEnabled))
      setSystemDark(systemIsDark)
      setPreferenceState(next)
    },
    [switchEnabled],
  )

  const value = useMemo(
    () => ({ preference, resolved, setPreference, switchEnabled }),
    [preference, resolved, setPreference, switchEnabled],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  return useContext(ThemeContext)
}
