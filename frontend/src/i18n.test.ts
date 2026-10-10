import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n, { changeLocale, ensureLocale } from '@/i18n'

afterEach(async () => {
  localStorage.clear()
  await i18n.changeLanguage('en')
})

describe('i18n lazy locales', () => {
  it('has English available synchronously', () => {
    expect(i18n.language).toBe('en')
    expect(i18n.hasResourceBundle('en', 'translation')).toBe(true)
    expect(i18n.t('common.loading')).toBe('Loading…')
  })

  it('loads kk on demand and makes its strings available', async () => {
    i18n.removeResourceBundle('kk', 'translation')
    expect(i18n.hasResourceBundle('kk', 'translation')).toBe(false)

    await ensureLocale('kk')

    expect(i18n.hasResourceBundle('kk', 'translation')).toBe(true)
    expect(i18n.t('common.loading', { lng: 'kk' })).toBe('Жүктелуде…')
  })

  it('switches the language and i18n.language follows', async () => {
    await changeLocale('ru')
    expect(i18n.language).toBe('ru')
    expect(i18n.t('common.loading')).toBe('Загрузка…')

    await changeLocale('en')
    expect(i18n.language).toBe('en')
    expect(i18n.t('common.loading')).toBe('Loading…')
  })

  it('does not throw for unknown codes', async () => {
    await expect(ensureLocale('de')).resolves.toBeUndefined()
    await expect(ensureLocale('constructor')).resolves.toBeUndefined()
    await expect(changeLocale('de')).resolves.toBeUndefined()
    // Missing strings fall back to English, as before lazy loading.
    expect(i18n.t('common.loading')).toBe('Loading…')
  })

  it('applies the persisted locale before i18nReady resolves', async () => {
    localStorage.setItem('i18n-lang', 'ru')
    vi.resetModules()

    const fresh = await import('@/i18n')
    await fresh.i18nReady

    expect(fresh.default.language).toBe('ru')
    expect(fresh.default.t('common.loading')).toBe('Загрузка…')
  })
})
