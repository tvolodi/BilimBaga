/* design-ok-file: the design-system defaults are compared as literals */
// FR-BB321 AC-8 / DEC-002 section 4: which tenant colours override the design-system tokens in each theme.
import { contrastRatio, deriveDarkColour, MIN_TEXT_CONTRAST, parseHex, readableForeground } from '@/lib/contrast'

/** The design-system primary and accent (DEC-002 section 1). A tenant on these keeps the dark pair. */
export const DESIGN_DEFAULT_PRIMARY = '#2e6db4'
export const DESIGN_DEFAULT_ACCENT = '#c8a84b'
const WHITE = '#ffffff'

export type ThemeName = 'light' | 'dark'

export interface TenantPalette {
  primary_color?: string | null
  accent_color?: string | null
}

/** Overrides for one theme. A null field means the design-system token stays in force. */
export interface TenantOverrides {
  primary: string | null
  primaryForeground: string | null
  accent: string | null
}

const NO_OVERRIDES: TenantOverrides = { primary: null, primaryForeground: null, accent: null }

/**
 * The overrides a tenant palette gives in one theme.
 * Light: the tenant primary as stored, when it reaches 4.5:1 against white (DEC-002 rule 2); a value
 * below that is ignored in every theme. Dark: the primary derived for the dark grounds, unless it is the
 * design default or was ignored. The accent is as stored in light and derived in dark. With no primary,
 * every override is removed.
 */
export function tenantOverrides(palette: TenantPalette | undefined, theme: ThemeName): TenantOverrides {
  const storedPrimary = palette?.primary_color
  if (!storedPrimary) return NO_OVERRIDES

  const storedAccent = palette?.accent_color
  const accent = storedAccent && parseHex(storedAccent) ? storedAccent.toLowerCase() : null

  const primary = parseHex(storedPrimary) && contrastRatio(storedPrimary, WHITE) >= MIN_TEXT_CONTRAST
    ? storedPrimary.toLowerCase()
    : null

  if (theme === 'light') {
    return {
      primary,
      primaryForeground: primary ? readableForeground(primary) : null,
      accent,
    }
  }

  const derivedPrimary =
    primary && primary !== DESIGN_DEFAULT_PRIMARY ? deriveDarkColour(primary) : null
  return {
    primary: derivedPrimary,
    primaryForeground: derivedPrimary ? readableForeground(derivedPrimary) : null,
    accent: accent && accent !== DESIGN_DEFAULT_ACCENT ? deriveDarkColour(accent) : null,
  }
}
