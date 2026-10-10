import { useTranslation } from 'react-i18next'
import { TenantLogo } from '@/components/TenantLogo'
import type { TenantConfig } from '@/api/useTenantConfig'

interface BrandingPreviewProps {
  config: Partial<TenantConfig> & { logo?: string }
}

export function BrandingPreview({ config }: BrandingPreviewProps) {
  const { t } = useTranslation()
  const navItems = [
    t('nav.dashboard'),
    t('nav.users'),
    t('nav.exams'),
  ]

  return (
    <div className="space-y-3">
      <h2 className="text-sm font-medium text-muted-foreground">
        {t('settings.branding.preview')}
      </h2>
      <div className="rounded-lg border shadow-sm overflow-hidden">
        <div
          className="flex items-center gap-3 px-4 py-3"
          style={{ backgroundColor: config.primary_color ?? '#2E6DB4' }}
          data-testid="branding-preview-header"
        >
          <TenantLogo appName={config.app_name} logoOverride={config.logo} className="h-8" />
          <span className="font-semibold text-white">{config.app_name}</span>
          <nav className="ml-auto flex gap-2">
            {navItems.map((label) => (
              <span
                key={label}
                className="rounded px-2 py-1 text-sm text-white/80 hover:text-white"
              >
                {label}
              </span>
            ))}
          </nav>
        </div>
        <div className="bg-muted p-6 text-sm text-muted-foreground">
          <p>{t('settings.branding.previewBody')}</p>
          <button
            type="button"
            className="mt-4 rounded px-4 py-2 text-white text-sm"
            style={{ backgroundColor: config.accent_color ?? '#C8A84B' }}
            data-testid="branding-preview-accent-button"
          >
            {t('settings.branding.previewButton')}
          </button>
        </div>
      </div>
    </div>
  )
}
