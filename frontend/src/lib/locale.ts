import type { i18n as I18n } from 'i18next'

/**
 * Languages the UI ships translations for (FR-BB116). The labels are the languages' own names,
 * so they are intentionally not translated.
 */
export const SUPPORTED_LOCALES = [
  { code: 'kk', label: 'Қазақша' },
  { code: 'ru', label: 'Русский' },
  { code: 'en', label: 'English' },
] as const

export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]['code']

const RTL_LOCALES: readonly string[] = ['ar']

export function isSupportedLocale(code: unknown): code is SupportedLocale {
  return typeof code === 'string' && SUPPORTED_LOCALES.some((locale) => locale.code === code)
}

/**
 * Switch the UI language everywhere it is remembered: i18next, localStorage ('i18n-lang') and the
 * document language and direction. Storage can throw (private mode, blocked site data), so the
 * write is guarded; the switch itself still applies for the current page.
 */
export function applyLocale(i18n: I18n, code: SupportedLocale): void {
  void i18n.changeLanguage(code)
  try {
    localStorage.setItem('i18n-lang', code)
  } catch {
    /* storage unavailable: the choice lasts for this page only */
  }
  document.documentElement.lang = code
  document.documentElement.dir = RTL_LOCALES.includes(code) ? 'rtl' : 'ltr'
}
