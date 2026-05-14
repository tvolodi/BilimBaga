import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'

export const SUPPORTED_LOCALES = ['kk', 'ru', 'en'] as const
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]

const LOCALE_LABELS: Record<string, string> = {
  kk: 'Қазақша',
  ru: 'Русский',
  en: 'English',
}

interface LocaleSelectorProps {
  availableLocales: string[]
  defaultLocale: string
  onAvailableChange: (locales: string[]) => void
  onDefaultChange: (locale: string) => void
}

export function LocaleSelector({
  availableLocales,
  defaultLocale,
  onAvailableChange,
  onDefaultChange,
}: LocaleSelectorProps) {
  const { t } = useTranslation()

  function toggleLocale(locale: string) {
    if (availableLocales.includes(locale)) {
      onAvailableChange(availableLocales.filter((l) => l !== locale))
    } else {
      onAvailableChange([...availableLocales, locale])
    }
  }

  const defaultIncluded = availableLocales.includes(defaultLocale)

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>{t('settings.branding.locales')}</Label>
        <div className="flex flex-wrap gap-3">
          {SUPPORTED_LOCALES.map((locale) => {
            const checked = availableLocales.includes(locale)
            return (
              <label
                key={locale}
                className="flex items-center gap-2 rounded-md border border-input bg-background px-3 py-1.5 text-sm cursor-pointer hover:bg-muted"
              >
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => toggleLocale(locale)}
                  aria-label={LOCALE_LABELS[locale]}
                />
                <span>{LOCALE_LABELS[locale]}</span>
              </label>
            )
          })}
        </div>
        {!defaultIncluded && (
          <p role="alert" className="text-sm text-destructive">
            {t('settings.branding.localeError')}
          </p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="default-locale-select">{t('settings.branding.defaultLocale')}</Label>
        <Select
          id="default-locale-select"
          value={defaultLocale}
          onChange={(e) => onDefaultChange(e.target.value)}
          aria-label={t('settings.branding.defaultLocale')}
        >
          {SUPPORTED_LOCALES.map((locale) => (
            <option key={locale} value={locale}>
              {LOCALE_LABELS[locale]}
            </option>
          ))}
        </Select>
      </div>
    </div>
  )
}
