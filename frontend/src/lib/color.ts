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
