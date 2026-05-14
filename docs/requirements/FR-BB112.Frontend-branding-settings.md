# FR-BB112 — Frontend: Branding Settings

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB112 |
| Phase | 1 — Foundation |
| Priority | 3 |
| Status | Implemented |
| Depends On | FR-BB13, FR-BB111 |

## Description
Provides the Settings → Branding page where super admins can update the organisation's visual identity: application name, logo, primary colour, accent colour, default locale, and available locales. A live preview panel shows how the selected logo and colours will appear in the platform header before the changes are saved. WCAG AA contrast validation warns admins if a selected colour combination will be inaccessible.

## Acceptance Criteria
- [ ] AC-1: The Branding Settings page is accessible only to `super_admin`; any other role receives a `403` redirect to `/admin`.
- [ ] AC-2: The form loads current values from `GET /api/v1/tenant/config` (and the logo URL) when the page mounts; fields are pre-populated before the user makes any changes.
- [ ] AC-3: Logo upload supports drag-and-drop onto a drop zone or a click-to-open file picker; accepted formats are PNG, JPEG, SVG, and WebP; maximum file size is 1 MB.
- [ ] AC-4: Uploading a file larger than 1 MB shows an inline validation error before sending any request to the server.
- [ ] AC-5: The primary colour picker and accent colour picker each render a native `<input type="color">` element enhanced with a hex text input that stays in sync.
- [ ] AC-6: After any colour or logo change the live preview panel immediately reflects the new values (before saving); the preview renders a sample header containing the logo, `app_name`, and navigation pill coloured with the selected primary colour.
- [ ] AC-7: When the contrast ratio between the selected primary colour and white falls below 4.5:1 (WCAG AA for normal text), a yellow warning banner is displayed near the primary colour picker. The same check applies to the accent colour.
- [ ] AC-8: The available locales multi-select always contains the default locale (selecting fewer locales than the current default triggers a validation error: "Default locale must be included in available locales").
- [ ] AC-9: On form save, `PUT /api/v1/tenant/config` is called with only the changed fields; on success, the `tenant_config` query is invalidated globally so the top bar and login page immediately reflect the new branding without a full reload.
- [ ] AC-10: All user-visible strings (labels, warnings, error messages, button text) come from i18n locale files; no hardcoded English text in component files.

## Technical Specification

### Frontend Components

#### File layout

```
src/
  pages/
    admin/
      settings/
        BrandingSettingsPage.tsx
  components/
    settings/
      LogoUploader.tsx        # drag-drop + click-to-open, preview thumbnail
      ColorPickerField.tsx    # <input type="color"> + hex text input + contrast warning
      LocaleSelector.tsx      # multi-select available locales + default locale select
      BrandingPreview.tsx     # live preview panel (sample header)
  api/
    tenant.ts                 # useTenantConfig, useUpdateTenantConfig (mutation)
```

#### `useUpdateTenantConfig` mutation (`src/api/tenant.ts`)

```typescript
interface TenantConfigUpdate {
  app_name?:          string;
  logo?:              string;   // base64 data URL: "data:image/png;base64,..."
  primary_color?:     string;
  accent_color?:      string;
  default_locale?:    string;
  available_locales?: string[];
}

export function useUpdateTenantConfig() {
  const qc = useQueryClient();
  return useMutation<void, ApiError, TenantConfigUpdate>({
    mutationFn: async (payload) => {
      const res = await apiFetch('/api/v1/tenant/config', {
        method: 'PUT',
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw res.error;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tenant', 'config'] });
    },
  });
}
```

#### `BrandingSettingsPage.tsx` — key behaviour

```typescript
export function BrandingSettingsPage() {
  const { data: config, isLoading } = useTenantConfig();
  const updateConfig = useUpdateTenantConfig();
  const { t } = useTranslation();
  const [draft, setDraft] = useState<TenantConfigUpdate>({});

  // Merge draft over config for preview
  const preview = { ...config, ...draft };

  if (isLoading) return <FullPageSpinner />;

  const handleSave = async () => {
    if (Object.keys(draft).length === 0) return;
    await updateConfig.mutateAsync(draft);
    setDraft({});
  };

  return (
    <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
      <div className="space-y-6">
        <FormField label={t('settings.branding.appName')}>
          <Input
            defaultValue={config?.app_name}
            onChange={(e) => setDraft(d => ({ ...d, app_name: e.target.value }))}
          />
        </FormField>

        <LogoUploader
          currentLogoUrl="/api/v1/tenant/logo"
          onFile={(dataUrl) => setDraft(d => ({ ...d, logo: dataUrl }))}
        />

        <ColorPickerField
          label={t('settings.branding.primaryColor')}
          value={draft.primary_color ?? config?.primary_color ?? '#0ea5e9'}
          onChange={(color) => setDraft(d => ({ ...d, primary_color: color }))}
          contrastAgainst="#ffffff"
        />

        <ColorPickerField
          label={t('settings.branding.accentColor')}
          value={draft.accent_color ?? config?.accent_color ?? '#f59e0b'}
          onChange={(color) => setDraft(d => ({ ...d, accent_color: color }))}
          contrastAgainst="#ffffff"
        />

        <LocaleSelector
          availableLocales={draft.available_locales ?? config?.available_locales ?? ['kk']}
          defaultLocale={draft.default_locale ?? config?.default_locale ?? 'kk'}
          onAvailableChange={(locs) => setDraft(d => ({ ...d, available_locales: locs }))}
          onDefaultChange={(loc) => setDraft(d => ({ ...d, default_locale: loc }))}
        />

        <Button onClick={handleSave} disabled={updateConfig.isPending}>
          {t('settings.branding.save')}
        </Button>
      </div>

      <BrandingPreview config={preview} />
    </div>
  );
}
```

#### `ColorPickerField.tsx` — WCAG AA contrast check

```typescript
// Contrast ratio calculation (WCAG 2.1 formula)
function relativeLuminance(hex: string): number {
  const rgb = hexToRgb(hex);
  return rgb.map((c) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  }).reduce((acc, c, i) => acc + c * [0.2126, 0.7152, 0.0722][i], 0);
}

function contrastRatio(hex1: string, hex2: string): number {
  const l1 = relativeLuminance(hex1);
  const l2 = relativeLuminance(hex2);
  const [lighter, darker] = l1 > l2 ? [l1, l2] : [l2, l1];
  return (lighter + 0.05) / (darker + 0.05);
}

// Inside ColorPickerField render:
const ratio = contrastRatio(value, contrastAgainst);
const passes = ratio >= 4.5;
```

#### `BrandingPreview.tsx` — sample header

```typescript
export function BrandingPreview({ config }: { config: Partial<TenantConfig> }) {
  return (
    <div className="rounded-lg border shadow-sm overflow-hidden">
      <div
        className="flex items-center gap-3 px-4 py-3"
        style={{ backgroundColor: config.primary_color ?? '#0ea5e9' }}
      >
        <TenantLogo appName={config.app_name} logoOverride={config.logo} className="h-8" />
        <span className="font-semibold text-white">{config.app_name}</span>
        <nav className="ml-auto flex gap-2">
          {['Dashboard', 'Users', 'Exams'].map((label) => (
            <span key={label} className="rounded px-2 py-1 text-sm text-white/80 hover:text-white">
              {label}
            </span>
          ))}
        </nav>
      </div>
      <div className="bg-muted p-6 text-sm text-muted-foreground">
        {/* Sample content area */}
        <p>Preview of the main content area.</p>
        <button
          className="mt-4 rounded px-4 py-2 text-white text-sm"
          style={{ backgroundColor: config.accent_color ?? '#f59e0b' }}
        >
          Sample Button
        </button>
      </div>
    </div>
  );
}
```

#### i18n keys required (excerpt `en.json`)

```json
{
  "settings": {
    "branding": {
      "title":          "Branding",
      "appName":        "Application Name",
      "logo":           "Logo",
      "logoHint":       "PNG, JPEG, SVG or WebP · max 1 MB",
      "logoDragDrop":   "Drag and drop or click to upload",
      "primaryColor":   "Primary Colour",
      "accentColor":    "Accent Colour",
      "contrastWarning":"Low contrast — text may be hard to read (ratio: {{ratio}}:1)",
      "locales":        "Available Languages",
      "defaultLocale":  "Default Language",
      "localeError":    "Default language must be in the available languages list.",
      "save":           "Save Changes",
      "saveSuccess":    "Branding updated successfully.",
      "preview":        "Live Preview"
    }
  }
}
```

## Notes
- The logo is converted to a base64 data URL in the browser (`FileReader.readAsDataURL`) before being sent in the PUT body. The server strips the `data:image/...;base64,` prefix when storing in the DB and re-adds it when serving `GET /api/v1/tenant/logo`.
- File type validation is done both client-side (via `accept` attribute and MIME type check) and server-side (magic-byte inspection) to prevent MIME-type spoofing.
- The contrast ratio check is purely advisory; it warns but does not block the save action. Teams may have brand colours that fail WCAG AA and must accept the trade-off.
- `BrandingPreview` uses inline `style` attributes deliberately for the dynamic colour values — these cannot be expressed as Tailwind utility classes because the values are not known at build time.
- The `LocaleSelector` lists supported locales as `kk` (Қазақша), `ru` (Русский), `en` (English); the available set is defined in a constants file and matched against what the frontend i18n JSON files cover.
