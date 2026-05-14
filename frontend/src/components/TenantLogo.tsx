import { useTranslation } from 'react-i18next'

interface TenantLogoProps {
  appName?: string
}

export function TenantLogo({ appName }: TenantLogoProps) {
  const { t } = useTranslation()

  function handleError(e: React.SyntheticEvent<HTMLImageElement>) {
    const img = e.currentTarget
    img.style.display = 'none'
    const next = img.nextElementSibling as HTMLElement | null
    if (next) {
      next.style.display = ''
    }
  }

  const fallbackText = appName ?? t('common.logo_placeholder')

  return (
    <div className="flex justify-center">
      <img
        src="/api/v1/tenant/logo"
        alt={fallbackText}
        className="h-12 object-contain"
        onError={handleError}
      />
      <span style={{ display: 'none' }} className="text-lg font-bold">
        {fallbackText}
      </span>
    </div>
  )
}
