import { describe, expect, it } from 'vitest'
import {
  contrastRatio,
  deriveDarkColour,
  DARK_GROUNDS,
  MIN_TEXT_CONTRAST,
  parseHex,
  readableForeground,
} from './contrast'

function hueOf(hex: string): number {
  const [r, g, b] = (parseHex(hex) ?? [0, 0, 0]).map((c) => c / 255)
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  if (max === min) return 0
  const d = max - min
  let h: number
  if (max === r) h = (g - b) / d + (g < b ? 6 : 0)
  else if (max === g) h = (b - r) / d + 2
  else h = (r - g) / d + 4
  return h * 60
}

describe('parseHex', () => {
  it('parses six and three digit hex, case-insensitively', () => {
    expect(parseHex('#2E6DB4')).toEqual([46, 109, 180])
    expect(parseHex('#abc')).toEqual([0xaa, 0xbb, 0xcc])
  })

  it('returns null for anything that is not #rgb or #rrggbb', () => {
    for (const bad of ['2e6db4', '#2e6db', '#2e6db4ff', 'red', '', '#gggggg', 'red} body {']) {
      expect(parseHex(bad)).toBeNull()
    }
  })
})

describe('contrastRatio', () => {
  it('is 21 for black on white and 1 for a colour on itself', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 5)
    expect(contrastRatio('#2e6db4', '#2e6db4')).toBeCloseTo(1, 5)
  })

  it('returns 1 when either argument does not parse', () => {
    expect(contrastRatio('not-a-colour', '#ffffff')).toBe(1)
    expect(contrastRatio('#000000', '#12')).toBe(1)
  })
})

describe('readableForeground', () => {
  it('picks white on a dark colour and the dark text on a light one', () => {
    expect(readableForeground('#1b3a6b')).toBe('#ffffff')
    expect(readableForeground('#e5e7eb')).toBe('#0f1623')
  })

  it('accepts three digit input', () => {
    expect(readableForeground('#000')).toBe('#ffffff')
    expect(readableForeground('#fff')).toBe('#0f1623')
  })

  it('returns white for an invalid input', () => {
    expect(readableForeground('nope')).toBe('#ffffff')
  })
})

describe('deriveDarkColour', () => {
  it.each([
    ['#2E6DB4', '#6198d7'],
    ['#1B3A6B', '#6c97da'],
    ['#0F766E', '#15a79c'],
    ['#7C3AED', '#ab81f3'],
    ['#B91C1C', '#e96d6d'],
    ['#0EA5E9', '#0ea5e9'],
  ])('derives %s as %s', (input, expected) => {
    expect(deriveDarkColour(input)).toBe(expected)
  })

  it('returns null for an invalid input', () => {
    expect(deriveDarkColour('blue')).toBeNull()
  })

  it.each(['#808080', '#000000', '#ffffff', '#2e6db4', '#c8a84b'])(
    'gives a colour that clears every dark ground and its own foreground for %s',
    (input) => {
      const derived = deriveDarkColour(input)
      expect(derived).not.toBeNull()
      const colour = derived as string
      for (const ground of DARK_GROUNDS) {
        expect(contrastRatio(colour, ground)).toBeGreaterThanOrEqual(MIN_TEXT_CONTRAST)
      }
      expect(contrastRatio(readableForeground(colour), colour)).toBeGreaterThanOrEqual(MIN_TEXT_CONTRAST)
    },
  )

  it('keeps the hue within 2 degrees where a hue is defined', () => {
    for (const input of ['#1b3a6b', '#b91c1c', '#7c3aed', '#0f766e']) {
      const derived = deriveDarkColour(input) as string
      const difference = Math.abs(hueOf(derived) - hueOf(input))
      expect(Math.min(difference, 360 - difference)).toBeLessThanOrEqual(2)
    }
  })

  it('returns a colour that is lowercase', () => {
    expect(deriveDarkColour('#0EA5E9')).toBe('#0ea5e9')
  })
})
