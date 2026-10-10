import '@testing-library/jest-dom/vitest'
import { ensureLocale } from '@/i18n'

// The app loads kk and ru on demand. The suite switches to them synchronously, so both are loaded
// before any test file runs.
await Promise.all([ensureLocale('kk'), ensureLocale('ru')])
