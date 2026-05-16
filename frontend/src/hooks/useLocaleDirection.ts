import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

const RTL_LOCALES = new Set(['ar'])

export function useLocaleDirection() {
  const { i18n } = useTranslation()
  useEffect(() => {
    const dir = RTL_LOCALES.has(i18n.language) ? 'rtl' : 'ltr'
    document.documentElement.dir = dir
    document.documentElement.lang = i18n.language
  }, [i18n.language])
}
