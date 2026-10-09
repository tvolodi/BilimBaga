import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Select } from '@/components/ui/select'
import { useUpdateMyLocale } from '@/api/users'
import { SUPPORTED_LOCALES, applyLocale, isSupportedLocale } from '@/lib/locale'

export function LocaleSwitcher() {
  const { i18n } = useTranslation()
  const qc = useQueryClient()
  const updateMyLocale = useUpdateMyLocale()

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const code = e.target.value
    if (!isSupportedLocale(code)) return
    applyLocale(i18n, code)
    // FR-BB116 AC-7: a signed-in user also persists the choice. Best-effort: a failure is not
    // surfaced and never reverts the switch that already happened.
    if (qc.getQueryData<string | null>(['auth', 'accessToken'])) {
      updateMyLocale.mutate(code)
    }
  }

  return (
    <Select value={i18n.language} onChange={handleChange} className="w-[104px] sm:w-[120px]">
      {SUPPORTED_LOCALES.map((locale) => (
        <option key={locale.code} value={locale.code}>
          {locale.label}
        </option>
      ))}
    </Select>
  )
}
