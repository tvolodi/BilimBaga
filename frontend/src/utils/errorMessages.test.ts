import { describe, it, expect } from 'vitest'
import en from '@/locales/en.json'
import { ERROR_CODE_MAP, resolveErrorKey } from './errorMessages'

function lookup(key: string): unknown {
  return key
    .split('.')
    .reduce<unknown>((acc, part) => (acc as Record<string, unknown> | undefined)?.[part], en)
}

describe('resolveErrorKey', () => {
  it('maps known backend codes to their i18n key', () => {
    expect(resolveErrorKey('TOKEN_EXPIRED')).toBe('errors.tokenExpired')
    expect(resolveErrorKey('DUPLICATE_EMAIL')).toBe('errors.duplicateEmail')
  })

  it('maps legacy and new aliases to the same key', () => {
    expect(resolveErrorKey('ERR_NOT_FOUND')).toBe(resolveErrorKey('NOT_FOUND'))
    expect(resolveErrorKey('ERR_INTERNAL')).toBe(resolveErrorKey('INTERNAL_ERROR'))
  })

  it('falls back to the generic internal error for unknown or missing codes', () => {
    expect(resolveErrorKey('SOMETHING_NEW')).toBe('errors.internal')
    expect(resolveErrorKey(undefined)).toBe('errors.internal')
    expect(resolveErrorKey('')).toBe('errors.internal')
  })

  it('only references keys that exist in the English locale', () => {
    const missing = Object.entries(ERROR_CODE_MAP)
      .filter(([, key]) => typeof lookup(key) !== 'string')
      .map(([code, key]) => `${code} -> ${key}`)
    expect(missing).toEqual([])
  })
})
