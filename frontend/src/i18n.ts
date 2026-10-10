import i18n from 'i18next'
import type { i18n as I18nInstance } from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'

// English is bundled eagerly: it is the fallbackLng and the first render needs it. kk and ru are
// fetched on demand so they stay out of the initial JS (FR-BB65 budget). A bundle is added once and
// the language is switched only after it is present, so a key is never looked up in a language whose
// strings are not loaded yet.
type LocaleModule = { default: Record<string, unknown> }

// Exported so tests can replace a loader (for example to make one load slow).
export const LAZY_LOCALES = new Map<string, () => Promise<LocaleModule>>([
  ['kk', () => import('./locales/kk.json')],
  ['ru', () => import('./locales/ru.json')],
])

const inFlight = new Map<string, Promise<void>>()

/**
 * True when `lng` can be switched to right away: English, a code that is not a lazy locale (the
 * i18next fallback applies), or a lazy locale whose bundle is already present.
 */
export function isLocaleReady(lng: string): boolean {
  return !LAZY_LOCALES.has(lng) || i18n.hasResourceBundle(lng, 'translation')
}

/**
 * Resolves once the bundle for `lng` is registered with i18next. Resolves at once when
 * isLocaleReady(lng). A failed load is forgotten so a later call can retry it.
 */
export function ensureLocale(lng: string): Promise<void> {
  const load = LAZY_LOCALES.get(lng)
  if (!load || isLocaleReady(lng)) return Promise.resolve()
  let pending = inFlight.get(lng)
  if (!pending) {
    pending = load()
      .then((mod) => {
        i18n.addResourceBundle(lng, 'translation', mod.default, true, true)
      })
      .finally(() => {
        inFlight.delete(lng)
      })
    inFlight.set(lng, pending)
  }
  return pending
}

// Each switch takes a number. When switches overlap, only the last one requested is applied, even
// if an earlier bundle finishes loading after it (an earlier request must never win).
let latestRequest = 0

/**
 * Switches `instance` to `lng` once its bundle is present. When the bundle is already present the
 * switch happens synchronously, as it always did.
 */
export async function switchLocale(instance: I18nInstance, lng: string): Promise<void> {
  const request = ++latestRequest
  if (!isLocaleReady(lng)) await ensureLocale(lng)
  if (request !== latestRequest) return // a later request superseded this one
  await instance.changeLanguage(lng)
}

/** Switches the module's i18next instance to `lng`, loading its bundle first when needed. */
export function changeLocale(lng: string): Promise<void> {
  return switchLocale(i18n, lng)
}

function readPersistedLocale(): string | null {
  try {
    return localStorage.getItem('i18n-lang')
  } catch {
    return null // storage unavailable: the default language applies
  }
}

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
  },
  lng: 'en',
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
})

const persisted = readPersistedLocale()

/**
 * Resolves when the persisted locale (if it is kk or ru) is active. main.tsx renders only after this,
 * so the first paint is never in English for a user who chose another language. Never rejects: a
 * failed load leaves the English fallback and the app still starts.
 */
export const i18nReady: Promise<void> =
  persisted !== null && LAZY_LOCALES.has(persisted)
    ? changeLocale(persisted).catch((err: unknown) => {
        console.error(`Could not load the ${persisted} translations`, err)
      })
    : Promise.resolve()

export default i18n
