import { describe, it, expect } from 'vitest'
import { canonicalHex, normaliseHex } from './color'

// #498: the saved colour is uppercase, the case of the seeded default; validation stays case-insensitive.
describe('canonicalHex (#498)', () => {
  it('saves a six-digit hex in uppercase, with or without the leading #', () => {
    expect(canonicalHex('2E6DB4')).toBe('#2E6DB4')
    expect(canonicalHex('#2E6DB4')).toBe('#2E6DB4')
    expect(canonicalHex('#2e6db4')).toBe('#2E6DB4')
  })

  it('expands the three-digit shorthand and uppercases it', () => {
    expect(canonicalHex('#abc')).toBe('#AABBCC')
  })

  it('returns null for anything that is not a hex colour', () => {
    expect(canonicalHex('nope')).toBeNull()
    expect(canonicalHex('#12')).toBeNull()
  })

  it('leaves normaliseHex lowercase, which the native colour input needs', () => {
    expect(normaliseHex('#2E6DB4')).toBe('#2e6db4')
  })
})
