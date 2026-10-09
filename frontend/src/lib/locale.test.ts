import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { applyLocale, isSupportedLocale } from './locale'

beforeEach(async () => {
  localStorage.clear()
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  document.documentElement.dir = 'ltr'
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('isSupportedLocale', () => {
  it('accepts exactly the shipped languages', () => {
    expect(isSupportedLocale('kk')).toBe(true)
    expect(isSupportedLocale('ru')).toBe(true)
    expect(isSupportedLocale('en')).toBe(true)
    expect(isSupportedLocale('de')).toBe(false)
    expect(isSupportedLocale('RU')).toBe(false)
    expect(isSupportedLocale('')).toBe(false)
    expect(isSupportedLocale(null)).toBe(false)
    expect(isSupportedLocale(undefined)).toBe(false)
    expect(isSupportedLocale(5)).toBe(false)
  })
})

describe('applyLocale', () => {
  it('switches i18next, localStorage and the document language', () => {
    applyLocale(i18n, 'ru')

    expect(i18n.language).toBe('ru')
    expect(localStorage.getItem('i18n-lang')).toBe('ru')
    expect(document.documentElement.lang).toBe('ru')
    expect(document.documentElement.dir).toBe('ltr')
  })

  it('still switches the UI when storage is unavailable', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })

    expect(() => applyLocale(i18n, 'kk')).not.toThrow()
    expect(i18n.language).toBe('kk')
    expect(document.documentElement.lang).toBe('kk')
  })
})
