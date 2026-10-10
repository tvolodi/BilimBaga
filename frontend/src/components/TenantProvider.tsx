import { useEffect, type ReactNode } from 'react'
import { useTenantConfig } from '@/api/useTenantConfig'

interface TenantProviderProps {
  children: ReactNode
}

const TENANT_STYLE_ID = 'tenant-branding'
const HEX_COLOUR = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/

/**
 * Writes the tenant primary as a stylesheet rule scoped to the light theme.
 *
 * `:root:not(.dark)` leaves the design-system dark primary in force under `.dark`.
 * An inline style on the root element would outrank the `.dark` token block and override
 * it (FR-BB321, DEC-002 section 4). The value is checked as hex before it goes into CSS.
 */
function applyLightPrimary(colour: string) {
  let style = document.getElementById(TENANT_STYLE_ID) as HTMLStyleElement | null
  if (!style) {
    style = document.createElement('style')
    style.id = TENANT_STYLE_ID
    document.head.appendChild(style)
  }
  style.textContent = HEX_COLOUR.test(colour)
    ? `:root:not(.dark) { --color-primary: ${colour}; }`
    : ''
}

export function TenantProvider({ children }: TenantProviderProps) {
  const { data } = useTenantConfig()

  useEffect(() => {
    if (!data) return
    applyLightPrimary(data.primary_color)
    document.documentElement.style.setProperty('--color-accent', data.accent_color)
  }, [data])

  return <>{children}</>
}
