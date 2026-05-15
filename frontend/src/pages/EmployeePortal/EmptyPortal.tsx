import { useTranslation } from 'react-i18next'

export function EmptyPortal() {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <svg
        className="h-16 w-16 text-muted-foreground mb-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        aria-hidden="true"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
        />
      </svg>
      <h2 className="text-lg font-semibold mb-2">{t('portal.empty.title')}</h2>
      <p className="text-sm text-muted-foreground max-w-xs">{t('portal.empty.description')}</p>
    </div>
  )
}
