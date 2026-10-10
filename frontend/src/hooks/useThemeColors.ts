import { useEffect, useState } from 'react'
import { useTheme } from '@/components/ThemeProvider'
import { useTenantVersion } from '@/components/TenantProvider'

/** The colours a chart reads from the active theme (FR-BB321 AC-10). */
export interface ThemeColours {
  /** The resolved primary: the tenant colour in light, its derived dark colour in dark. */
  primary: string
  /** Axis and grid lines, tracks and the unassigned part of bars. */
  border: string
  /** Tick and axis text. */
  textSecondary: string
  /** Muted labels. */
  textMuted: string
}

function readThemeColours(): ThemeColours {
  const style = getComputedStyle(document.documentElement)
  const read = (token: string) => style.getPropertyValue(token).trim()
  return {
    primary: read('--color-primary'),
    border: read('--color-border-default'),
    textSecondary: read('--color-text-secondary'),
    textMuted: read('--color-text-muted'),
  }
}

/**
 * Chart colours from the computed tokens of `<html>`. The read happens in an effect keyed on the
 * resolved theme and the tenant version, so a consumer re-renders after the class and the overrides
 * have changed. jsdom does not load index.css, so tests stub getComputedStyle.
 */
export function useThemeColors(): ThemeColours {
  const { resolved } = useTheme()
  const version = useTenantVersion()
  const [colours, setColours] = useState<ThemeColours>(readThemeColours)

  useEffect(() => {
    setColours(readThemeColours())
  }, [resolved, version])

  return colours
}
