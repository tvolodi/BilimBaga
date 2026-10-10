/* design-ok-file: colour maths needs literals */
// FR-BB321 AC-8 / DEC-002 section 4: WCAG contrast maths and the tenant colour the dark theme derives.

/** WCAG 2.x minimum contrast for normal text (FR-BB320 AC-1, DEC-002 rule 2). */
export const MIN_TEXT_CONTRAST = 4.5

/** The dark grounds a text colour or a derived fill has to clear (design-system dark tokens). */
export const DARK_GROUNDS = ['#0f1623', '#1a2133', '#222d42', '#1e2a3d'] as const

const WHITE = '#ffffff'
const DARK_TEXT = '#0f1623'
const HEX = /^#(?:[0-9a-f]{3}|[0-9a-f]{6})$/i

type Rgb = [number, number, number]

/** Parses `#rgb` or `#rrggbb` (case-insensitive) into channels, or returns null. */
export function parseHex(hex: string): Rgb | null {
  if (!HEX.test(hex)) return null
  const digits = hex.slice(1)
  const full = digits.length === 3 ? digits.split('').map((c) => c + c).join('') : digits
  const num = parseInt(full, 16)
  return [(num >> 16) & 0xff, (num >> 8) & 0xff, num & 0xff]
}

function relativeLuminance([r, g, b]: Rgb): number {
  const [lr, lg, lb] = [r, g, b].map((c) => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
  })
  return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb
}

/** WCAG contrast ratio in [1, 21]. Returns 1 when either argument does not parse. */
export function contrastRatio(a: string, b: string): number {
  const rgbA = parseHex(a)
  const rgbB = parseHex(b)
  if (!rgbA || !rgbB) return 1
  const lumA = relativeLuminance(rgbA)
  const lumB = relativeLuminance(rgbB)
  const [lighter, darker] = lumA > lumB ? [lumA, lumB] : [lumB, lumA]
  return (lighter + 0.05) / (darker + 0.05)
}

/** The text colour, white or the dark text, with the higher contrast on `hex`. White on a tie or an invalid input. */
export function readableForeground(hex: string): string {
  return contrastRatio(hex, DARK_TEXT) > contrastRatio(hex, WHITE) ? DARK_TEXT : WHITE
}

function passesDarkGrounds(hex: string): boolean {
  const weakest = Math.min(...DARK_GROUNDS.map((ground) => contrastRatio(hex, ground)))
  return weakest >= MIN_TEXT_CONTRAST && contrastRatio(readableForeground(hex), hex) >= MIN_TEXT_CONTRAST
}

function toHsl([r, g, b]: Rgb): [number, number, number] {
  const rn = r / 255
  const gn = g / 255
  const bn = b / 255
  const max = Math.max(rn, gn, bn)
  const min = Math.min(rn, gn, bn)
  const lightness = (max + min) / 2
  if (max === min) return [0, 0, lightness]
  const delta = max - min
  const saturation = lightness > 0.5 ? delta / (2 - max - min) : delta / (max + min)
  let hue: number
  if (max === rn) hue = (gn - bn) / delta + (gn < bn ? 6 : 0)
  else if (max === gn) hue = (bn - rn) / delta + 2
  else hue = (rn - gn) / delta + 4
  return [hue * 60, saturation, lightness]
}

function fromHsl(hue: number, saturation: number, lightness: number): string {
  const chroma = (1 - Math.abs(2 * lightness - 1)) * saturation
  const second = chroma * (1 - Math.abs(((hue / 60) % 2) - 1))
  const offset = lightness - chroma / 2
  const [r, g, b] =
    hue < 60 ? [chroma, second, 0]
    : hue < 120 ? [second, chroma, 0]
    : hue < 180 ? [0, chroma, second]
    : hue < 240 ? [0, second, chroma]
    : hue < 300 ? [second, 0, chroma]
    : [chroma, 0, second]
  return (
    '#' +
    [r, g, b].map((value) => Math.round((value + offset) * 255).toString(16).padStart(2, '0')).join('')
  )
}

/**
 * The colour the dark theme uses for a tenant colour (DEC-002 section 4, FR-BB321 AC-8).
 * A colour that already clears the dark grounds is returned as it is. Otherwise its lightness is
 * raised, one point at a time, keeping hue and saturation, until it clears them. Null when the input
 * is not a hex colour or no lightness up to 90% clears them.
 */
export function deriveDarkColour(hex: string): string | null {
  const rgb = parseHex(hex)
  if (!rgb) return null
  const original = hex.toLowerCase()
  if (passesDarkGrounds(original)) return original
  const [hue, saturation, lightness] = toHsl(rgb)
  const start = Math.round(lightness * 100)
  for (let step = start + 1; step <= 90; step++) {
    const candidate = fromHsl(hue, saturation, step / 100)
    if (passesDarkGrounds(candidate)) return candidate
  }
  return null
}
