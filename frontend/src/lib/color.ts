// The contrast maths lives in contrast.ts (FR-BB321 AC-8). This module keeps the input normaliser.
export { contrastRatio, MIN_TEXT_CONTRAST } from '@/lib/contrast'

/** Normalises #rgb / #rrggbb (with or without the leading #) to lowercase #rrggbb, or null. */
export function normaliseHex(input: string): string | null {
  const trimmed = input.trim().replace(/^#/, '')
  if (!/^([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(trimmed)) return null
  const expanded =
    trimmed.length === 3
      ? trimmed
          .split('')
          .map((c) => c + c)
          .join('')
      : trimmed
  return `#${expanded.toLowerCase()}`
}

/**
 * The saved form of a hex colour: #RRGGBB in uppercase, the case the seeded default uses (#498). The
 * colour field works in lowercase, because the native colour input needs it; only the saved value is
 * uppercased. Validation stays case-insensitive, through normaliseHex.
 */
export function canonicalHex(input: string): string | null {
  const normalised = normaliseHex(input)
  return normalised === null ? null : normalised.toUpperCase()
}
