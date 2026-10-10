/* design-ok-file: tenant colour editor */
import { useTranslation } from 'react-i18next'
import { TenantLogo } from '@/components/TenantLogo'
import type { TenantConfig } from '@/api/useTenantConfig'
import { useTheme } from '@/components/ThemeProvider'
import { DESIGN_DEFAULT_ACCENT, DESIGN_DEFAULT_PRIMARY, tenantOverrides } from '@/lib/tenantColours'

interface BrandingPreviewProps {
  config: Partial<TenantConfig> & { logo?: string }
}

export function BrandingPreview({ config }: BrandingPreviewProps) {
  const { t } = useTranslation()
  // The preview shows the colours the current theme gives this config (FR-BB321 AC-8).
  const { resolved } = useTheme()
  const overrides = tenantOverrides(
    {
      primary_color: config.primary_color ?? DESIGN_DEFAULT_PRIMARY,
      accent_color: config.accent_color ?? DESIGN_DEFAULT_ACCENT,
    },
    resolved,
  )
  const primary = overrides.primary ?? 'var(--color-primary)'
  const primaryText = overrides.primaryForeground ?? 'var(--color-primary-foreground)'
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
          style={{ backgroundColor: primary, color: primaryText }}
          data-testid="branding-preview-header"
        >
          <TenantLogo appName={config.app_name} logoOverride={config.logo} className="h-8" />
          <span className="font-semibold">{config.app_name}</span>
          <nav className="ml-auto flex gap-2">
            {navItems.map((label) => (
              <span
                key={label}
                className="rounded px-2 py-1 text-sm opacity-80 hover:opacity-100"
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
            className="mt-4 rounded px-4 py-2 text-sm"
            style={{ backgroundColor: overrides.accent ?? 'var(--color-accent)', color: 'var(--color-accent-foreground)' }}
            data-testid="branding-preview-accent-button"
          >
            {t('settings.branding.previewButton')}
          </button>
        </div>
      </div>
    </div>
  )
}
