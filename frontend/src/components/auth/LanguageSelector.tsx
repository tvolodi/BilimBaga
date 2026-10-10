import { useTranslation } from 'react-i18next'
import { changeLocale } from '@/i18n'

interface LanguageSelectorProps {
  availableLocales: string[]
}

const LOCALE_LABELS: Record<string, string> = {
  kk: 'Қазақша',
  ru: 'Русский',
  en: 'English',
}

export function LanguageSelector({ availableLocales }: LanguageSelectorProps) {
  const { i18n, t } = useTranslation()

  function handleChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const lang = e.target.value
    void changeLocale(lang).catch((err: unknown) => {
      console.error(`Could not load the ${lang} translations`, err)
    })
    localStorage.setItem('i18n-lang', lang)
  }

  return (
    <div className="mt-4 flex justify-center">
      <select
        value={i18n.language}
        onChange={handleChange}
        className="rounded-md border border-input bg-background px-3 py-1 text-sm"
        aria-label={t('common.language')}
      >
        {availableLocales.map((locale) => (
          <option key={locale} value={locale}>
            {LOCALE_LABELS[locale] ?? locale}
          </option>
        ))}
      </select>
    </div>
  )
}
