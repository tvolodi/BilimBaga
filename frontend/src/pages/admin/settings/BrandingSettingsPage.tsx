/* design-ok-file: tenant colour editor */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useTenantConfig } from '@/api/useTenantConfig'
import { useUpdateTenantConfig, type TenantConfigUpdate } from '@/api/tenant'
import { FullPageSpinner } from '@/components/FullPageSpinner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LogoUploader } from '@/components/settings/LogoUploader'
import { canonicalHex } from '@/lib/color'
import { ColorPickerField } from '@/components/settings/ColorPickerField'
import { LocaleSelector } from '@/components/settings/LocaleSelector'
import { BrandingPreview } from '@/components/settings/BrandingPreview'
import { contrastRatio, MIN_TEXT_CONTRAST, normaliseHex } from '@/lib/color'

const WHITE = '#ffffff'
const DEFAULT_PRIMARY = '#2E6DB4'
const DEFAULT_ACCENT = '#C8A84B'

type PrimaryIssue = { kind: 'invalid' } | { kind: 'contrast'; ratio: string } | null

// #498: the colours are saved in one case, uppercase like the seeded default, however they were typed.
function canonicalColours(update: TenantConfigUpdate): TenantConfigUpdate {
  const next = { ...update }
  if (next.primary_color !== undefined) next.primary_color = canonicalHex(next.primary_color) ?? next.primary_color
  if (next.accent_color !== undefined) next.accent_color = canonicalHex(next.accent_color) ?? next.accent_color
  return next
}

// FR-BB320 AC-1: the server rule, applied to the draft value before save.
function primaryIssueFor(draftValue: string | undefined): PrimaryIssue {
  if (draftValue === undefined) return null
  const hex = normaliseHex(draftValue)
  if (hex === null) return { kind: 'invalid' }
  const ratio = contrastRatio(hex, WHITE)
  return ratio < MIN_TEXT_CONTRAST ? { kind: 'contrast', ratio: ratio.toFixed(2) } : null
}

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

  // The primary is checked only when this draft sets it, matching the server rule.
  // A stored colour that already fails is not blocked here, so other fields can still be saved.
  const primaryIssue = primaryIssueFor(draft.primary_color)

  const hasDraftChanges = Object.keys(draft).length > 0
  const canSave =
    hasDraftChanges && defaultIncluded && primaryIssue === null && !updateConfig.isPending

  async function handleSave() {
    if (!canSave) return
    try {
      await updateConfig.mutateAsync(canonicalColours(draft))
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
            value={draft.primary_color ?? config?.primary_color ?? DEFAULT_PRIMARY}
            onChange={(color) => setDraft((d) => ({ ...d, primary_color: color }))}
            contrastAgainst={WHITE}
          />

          {primaryIssue && (
            <p className="text-sm text-destructive" data-testid="primary-color-blocked">
              {primaryIssue.kind === 'invalid'
                ? t('settings.branding.primaryInvalid')
                : t('settings.branding.primaryContrastBlocked', { ratio: primaryIssue.ratio })}
            </p>
          )}

          <ColorPickerField
            label={t('settings.branding.accentColor')}
            value={draft.accent_color ?? config?.accent_color ?? DEFAULT_ACCENT}
            onChange={(color) => setDraft((d) => ({ ...d, accent_color: color }))}
            contrastAgainst={WHITE}
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
                    ? 'text-sm text-success'
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
