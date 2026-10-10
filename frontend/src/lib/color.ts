/** WCAG 2.x minimum contrast for normal text (FR-BB320 AC-1). */
export const MIN_TEXT_CONTRAST = 4.5

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

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace(/^#/, '')
  const full =
    h.length === 3
      ? h
          .split('')
          .map((c) => c + c)
          .join('')
      : h
  const num = parseInt(full, 16)
  return [(num >> 16) & 0xff, (num >> 8) & 0xff, num & 0xff]
}

function relativeLuminance(hex: string): number {
  const rgb = hexToRgb(hex)
  return rgb
    .map((c) => {
      const s = c / 255
      return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
    })
    .reduce((acc, c, i) => acc + c * [0.2126, 0.7152, 0.0722][i], 0)
}

/** WCAG 2.x contrast ratio between two hex colours, in [1, 21]. */
export function contrastRatio(hex1: string, hex2: string): number {
  const l1 = relativeLuminance(hex1)
  const l2 = relativeLuminance(hex2)
  const [lighter, darker] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (lighter + 0.05) / (darker + 0.05)
}
