# FR-BB62 — Full i18n Coverage

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB62 |
| Phase | 6 — Polish & Hardening |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB110, FR-BB111, FR-BB112, FR-BB26, FR-BB27, FR-BB312, FR-BB313, FR-BB314, FR-BB45, FR-BB46, FR-BB47, FR-BB56, FR-BB57, FR-BB58 |

## Description
Ensures complete internationalization coverage across all frontend React components. All user-visible strings are moved to locale JSON files for Kazakh (`kk`), Russian (`ru`), and English (`en`). Number and date formatting respects the active locale. Backend error responses use error codes rather than English prose, enabling the frontend to translate all error messages. A CI check enforces key-set parity across all locale files.

## Acceptance Criteria
- [ ] AC-1: Every user-visible string in every React component and page is externalized to `src/locales/{locale}.json`; no hardcoded natural-language string exists in any `.tsx` / `.ts` file (verified by ESLint rule `i18next/no-literal-string` or equivalent manual audit sign-off).
- [ ] AC-2: All three locale files (`kk.json`, `ru.json`, `en.json`) have identical key sets; a CI script (or `npm test` sub-task) fails the build if any key is present in one file but absent in another.
- [ ] AC-3: A locale switcher UI element is available on the login screen and the user profile settings page; switching locale immediately re-renders all text without a full page reload.
- [ ] AC-4: Number values (scores, percentages, counts) are formatted via `Intl.NumberFormat` using the active locale; date and time values are formatted via `Intl.DateTimeFormat` or locale-aware `date-fns` functions throughout all pages.
- [ ] AC-5: When a question has no translation for the user's active locale, the question renders in its `default_locale` with a visible "(original language)" badge rendered from an i18n key (not hardcoded text).
- [ ] AC-6: All API error responses from the backend return an error code string (e.g. `EXAM_NOT_FOUND`) rather than an English message; the frontend maps each code to an i18n key and displays a translated message.
- [ ] AC-7: CSS throughout the frontend uses logical properties (`padding-inline-start`, `margin-inline-end`, `border-inline-start`, etc.) instead of physical directional properties (`padding-left`, `margin-right`, etc.) so RTL layouts render correctly without overrides.
- [ ] AC-8: The `<html>` element's `dir` attribute is set to `"rtl"` or `"ltr"` dynamically based on the active locale; the language is set via the `lang` attribute to the BCP-47 locale code.

## Technical Specification

### Configuration / Infrastructure

**Locale file location**: `frontend/src/locales/{locale}.json`

**i18next initialisation** (`src/i18n.ts`):
```ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import kk from './locales/kk.json';
import ru from './locales/ru.json';
import en from './locales/en.json';

i18n.use(initReactI18next).init({
  resources: { kk: { translation: kk }, ru: { translation: ru }, en: { translation: en } },
  lng: localStorage.getItem('locale') ?? 'ru',
  fallbackLng: 'ru',
  interpolation: { escapeValue: false },
});
```

**Locale persistence**: stored in `localStorage` key `'locale'`; also sent in `Accept-Language` header for API calls so the backend can localise email sends.

**CI key-parity check** (`scripts/check-i18n.ts` or inline in `package.json` test script):
```ts
// Flatten all keys from each JSON, compare sets; exit 1 if any diff
const locales = ['kk', 'ru', 'en'];
const keysets = locales.map(l => new Set(Object.keys(flatten(require(`./src/locales/${l}.json`)))));
// assert all sets are equal
```

### Frontend Components

**Locale switcher component** (`src/components/LocaleSwitcher.tsx`):
- Renders a `<Select>` (shadcn/ui) with options: Қазақша, Русский, English.
- On change: calls `i18n.changeLanguage(code)`, updates `localStorage`, sets `document.documentElement.lang` and `document.documentElement.dir`.
- Placed in: `LoginPage` header area and `UserProfilePage` settings card.

**RTL direction hook** (`src/hooks/useLocaleDirection.ts`):
```ts
const RTL_LOCALES = new Set(['ar']); // ready for future Arabic locale
export function useLocaleDirection() {
  const { i18n } = useTranslation();
  useEffect(() => {
    const dir = RTL_LOCALES.has(i18n.language) ? 'rtl' : 'ltr';
    document.documentElement.dir = dir;
    document.documentElement.lang = i18n.language;
  }, [i18n.language]);
}
```
Called once in `App.tsx`.

**Number formatter utility** (`src/utils/format.ts`):
```ts
export const formatNumber = (value: number, locale: string, opts?: Intl.NumberFormatOptions) =>
  new Intl.NumberFormat(locale, opts).format(value);

export const formatPercent = (value: number, locale: string) =>
  new Intl.NumberFormat(locale, { style: 'percent', maximumFractionDigits: 1 }).format(value / 100);

export const formatDate = (iso: string, locale: string) =>
  new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
```

**Content language fallback badge**:
- When question locale ≠ active locale, render:
  ```tsx
  <Badge variant="outline">{t('question.originalLanguageBadge')}</Badge>
  ```
  Translation key `question.originalLanguageBadge` must exist in all three locale files.

**Backend error code mapping**:
- Maintain `src/utils/errorMessages.ts` that maps backend error codes to i18n translation keys:
  ```ts
  export const ERROR_CODE_MAP: Record<string, string> = {
    EXAM_NOT_FOUND: 'errors.examNotFound',
    UNAUTHORIZED: 'errors.unauthorized',
    // ...
  };
  ```
- API query hooks use this map in their `onError` handlers before displaying toasts.

### API Endpoints
No new endpoints. The backend change is to ensure all error response bodies use `error.code` values from a defined enum (documented in `docs/architecture-guide.md`) rather than human-readable English prose.

### Implementation Details

**Locale JSON key namespace conventions**:
```
auth.*          — login, logout, register screens
nav.*           — sidebar, breadcrumbs, page titles
exam.*          — exam configuration, assignment labels
question.*      — question editor, bank list
session.*       — exam-taking screen labels
result.*        — result and certificate screens
admin.*         — admin dashboard, analytics
errors.*        — error code translations
email.*         — email template strings (used by FR-BB61)
common.*        — shared labels: save, cancel, loading, etc.
```

**ESLint rule** (optional but recommended):
```jsonc
// .eslintrc
"plugins": ["i18next"],
"rules": { "i18next/no-literal-string": ["warn", { "markupOnly": true }] }
```

## Notes
- Kazakh (`kk`) locale uses Cyrillic script for now; if Latin-script Kazakh (`kk-Latn`) is required later, it can be added as an additional locale without architectural changes.
- `date-fns` locale packages should be imported dynamically to avoid bundling all locales.
- `react-i18next` `Trans` component must be used for strings containing React elements (e.g., bold text, links within a sentence) to avoid splitting translatable strings.
