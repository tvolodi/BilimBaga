import { useTranslation } from 'react-i18next'
import { Select } from '@/components/ui/select'

const LOCALES = [
  { code: 'kk', label: 'Қазақша' },
  { code: 'ru', label: 'Русский' },
  { code: 'en', label: 'English' },
]

export function LocaleSwitcher() {
  const { i18n } = useTranslation()

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const code = e.target.value
    void i18n.changeLanguage(code)
    localStorage.setItem('i18n-lang', code)
    document.documentElement.lang = code
    document.documentElement.dir = ['ar'].includes(code) ? 'rtl' : 'ltr'
  }

  return (
    <Select value={i18n.language} onChange={handleChange} className="w-[120px]">
      {LOCALES.map((locale) => (
        <option key={locale.code} value={locale.code}>
          {locale.label}
        </option>
      ))}
    </Select>
  )
}
