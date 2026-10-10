import { useTranslation } from 'react-i18next'

export function SkipLink() {
  const { t } = useTranslation()
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2
                 focus:z-50 focus:inline-flex focus:min-h-11 focus:items-center focus:rounded focus:bg-primary focus:px-4 focus:py-2 focus:text-primary-foreground"
    >
      {t('common.skipToMain')}
    </a>
  )
}
