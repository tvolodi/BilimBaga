import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useTenantConfig } from '@/api/useTenantConfig'
import { useUpdateTenantConfig, type TenantConfigUpdate } from '@/api/tenant'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LogoUploader } from '@/components/settings/LogoUploader'
import { ColorPickerField } from '@/components/settings/ColorPickerField'
import { LocaleSelector } from '@/components/settings/LocaleSelector'
import { BrandingPreview } from '@/components/settings/BrandingPreview'

export function BrandingSettingsPage() {
  const { t } = useTranslation()
  const { data: config, isLoading } = useTenantConfig()
  const updateConfig = useUpdateTenantConfig()
  const [draft, setDraft] = useState<TenantConfigUpdate>({})
  const [feedback, setFeedback] = useState<{ kind: 'success' | 'error'; message: string } | null>(
    null,
  )

  if (isLoading) return <FullPageSpinner />

  const merged = {
    ...config,
    ...draft,
  }

  const availableLocales =
    draft.available_locales ?? config?.available_locales ?? ['kk']
  const defaultLocale = draft.default_locale ?? config?.default_locale ?? 'kk'
  const defaultIncluded = availableLocales.includes(defaultLocale)

  const hasDraftChanges = Object.keys(draft).length > 0
  const canSave =
    hasDraftChanges && defaultIncluded && !updateConfig.isPending

  async function handleSave() {
    if (!canSave) return
    try {
      await updateConfig.mutateAsync(draft)
      setDraft({})
      setFeedback({ kind: 'success', message: t('settings.branding.saveSuccess') })
    } catch (err) {
      const message =
        (err as { message?: string })?.message ?? t('settings.branding.saveError')
      setFeedback({ kind: 'error', message })
    }
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold">{t('settings.branding.title')}</h1>
      </header>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
        <div className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="branding-app-name">{t('settings.branding.appName')}</Label>
            <Input
              id="branding-app-name"
              defaultValue={config?.app_name ?? ''}
              onChange={(e) =>
                setDraft((d) => ({ ...d, app_name: e.target.value }))
              }
            />
          </div>

          <LogoUploader
            currentLogoUrl="/api/v1/tenant/logo"
            onFile={(dataUrl) => setDraft((d) => ({ ...d, logo: dataUrl }))}
          />

          <ColorPickerField
            label={t('settings.branding.primaryColor')}
            value={draft.primary_color ?? config?.primary_color ?? '#0ea5e9'}
            onChange={(color) => setDraft((d) => ({ ...d, primary_color: color }))}
            contrastAgainst="#ffffff"
          />

          <ColorPickerField
            label={t('settings.branding.accentColor')}
            value={draft.accent_color ?? config?.accent_color ?? '#f59e0b'}
            onChange={(color) => setDraft((d) => ({ ...d, accent_color: color }))}
            contrastAgainst="#ffffff"
          />

          <LocaleSelector
            availableLocales={availableLocales}
            defaultLocale={defaultLocale}
            onAvailableChange={(locs) =>
              setDraft((d) => ({ ...d, available_locales: locs }))
            }
            onDefaultChange={(loc) =>
              setDraft((d) => ({ ...d, default_locale: loc }))
            }
          />

          <div className="flex items-center gap-3">
            <Button onClick={handleSave} disabled={!canSave}>
              {t('settings.branding.save')}
            </Button>
            {feedback && (
              <span
                role="status"
                className={
                  feedback.kind === 'success'
                    ? 'text-sm text-green-700'
                    : 'text-sm text-destructive'
                }
              >
                {feedback.message}
              </span>
            )}
          </div>
        </div>

        <BrandingPreview config={merged} />
      </div>
    </div>
  )
}
