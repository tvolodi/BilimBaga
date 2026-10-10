import { createContext, useContext, useLayoutEffect, useMemo, useState, type ReactNode } from 'react'
import { useTenantConfig } from '@/api/useTenantConfig'
import { useTheme } from '@/components/ThemeProvider'
import { tenantOverrides, type TenantOverrides, type TenantPalette, type ThemeName } from '@/lib/tenantColours'

interface TenantProviderProps {
  children: ReactNode
}

const TENANT_STYLE_ID = 'tenant-branding'

interface TenantContextValue {
  /** Bumped after every change of the tenant overrides (FR-BB321 AC-10). */
  version: number
}

const TenantContext = createContext<TenantContextValue>({ version: 0 })

/** The version of the tenant overrides. Chart colours re-read the computed style when it changes. */
export function useTenantVersion(): number {
  return useContext(TenantContext).version
}

function declarations({ primary, primaryForeground }: TenantOverrides): string {
  const parts: string[] = []
  if (primary) parts.push(`--color-primary: ${primary};`)
  if (primaryForeground) parts.push(`--color-primary-foreground: ${primaryForeground};`)
  return parts.join(' ')
}

/**
 * The primary overrides as stylesheet rules, one per theme.
 *
 * The `.dark` rule and the light rule are both written, so the class on `<html>` picks the theme.
 * An inline style on the root element would outrank the `.dark` token block and override it in
 * both themes (FR-BB321, DEC-002 section 4). Only values that parsed as hex reach the CSS.
 */
function primaryStylesheet(palette: TenantPalette | undefined): string {
  const light = declarations(tenantOverrides(palette, 'light'))
  const dark = declarations(tenantOverrides(palette, 'dark'))
  const rules: string[] = []
  if (light) rules.push(`:root:not(.dark) { ${light} }`)
  if (dark) rules.push(`:root.dark { ${dark} }`)
  return rules.join('\n')
}

/** Sets or removes one inline root override. The accent pair stays inline, so each theme writes its own. */
function setRootOverride(name: string, value: string | null) {
  if (value) document.documentElement.style.setProperty(name, value)
  else document.documentElement.style.removeProperty(name)
}

/** Writes the tenant overrides for the current theme. */
function applyTenantStyles(palette: TenantPalette | undefined, theme: ThemeName) {
  let style = document.getElementById(TENANT_STYLE_ID) as HTMLStyleElement | null
  if (!style) {
    style = document.createElement('style')
    style.id = TENANT_STYLE_ID
    document.head.appendChild(style)
  }
  style.textContent = primaryStylesheet(palette)

  const { accent, accentForeground } = tenantOverrides(palette, theme)
  setRootOverride('--color-accent', accent)
  setRootOverride('--color-accent-foreground', accentForeground)
}

export function TenantProvider({ children }: TenantProviderProps) {
  const { data } = useTenantConfig()
  const { resolved } = useTheme()
  const [version, setVersion] = useState(0)

  // Layout effect: the overrides are on <html> before any consumer effect reads them.
  useLayoutEffect(() => {
    applyTenantStyles(data, resolved)
    setVersion((current) => current + 1)
  }, [data, resolved])

  const value = useMemo(() => ({ version }), [version])
  return <TenantContext.Provider value={value}>{children}</TenantContext.Provider>
}
