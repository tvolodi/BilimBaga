import { useTranslation } from 'react-i18next'
import { useTheme } from '@/components/ThemeProvider'
import { cn } from '@/lib/utils'

interface TenantLogoProps {
  appName?: string
  logoOverride?: string
  className?: string
}

export function TenantLogo({ appName, logoOverride, className }: TenantLogoProps) {
  const { t } = useTranslation()
  const { resolved } = useTheme()

  function handleError(e: React.SyntheticEvent<HTMLImageElement>) {
    const img = e.currentTarget
    img.style.display = 'none'
    const next = img.nextElementSibling as HTMLElement | null
    if (next) {
      next.style.display = ''
    }
  }

  const fallbackText = appName ?? t('common.logo_placeholder')
  const src = logoOverride && logoOverride.length > 0 ? logoOverride : '/api/v1/tenant/logo'

  return (
    <div className="flex justify-center">
      <img
        src={src}
        alt={fallbackText}
        className={cn('h-12 object-contain', resolved === 'dark' && 'bg-white rounded-md p-1', className)}
        onError={handleError}
      />
      <span style={{ display: 'none' }} className="text-lg font-bold">
        {fallbackText}
      </span>
    </div>
  )
}
