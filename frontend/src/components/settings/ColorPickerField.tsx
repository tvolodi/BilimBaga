import { useId, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { contrastRatio, MIN_TEXT_CONTRAST, normaliseHex } from '@/lib/color'

// Re-exported for existing callers and tests.
export { contrastRatio }

interface ColorPickerFieldProps {
  label: string
  value: string
  onChange: (color: string) => void
  contrastAgainst: string
}

export function ColorPickerField({
  label,
  value,
  onChange,
  contrastAgainst,
}: ColorPickerFieldProps) {
  const { t } = useTranslation()
  const colorInputId = useId()
  const hexInputId = useId()

  function handleColorChange(e: ChangeEvent<HTMLInputElement>) {
    onChange(e.target.value)
  }

  function handleHexChange(e: ChangeEvent<HTMLInputElement>) {
    const next = e.target.value
    const normalised = normaliseHex(next)
    if (normalised) {
      onChange(normalised)
    } else {
      onChange(next)
    }
  }

  const safeValue = normaliseHex(value) ?? '#000000'
  const ratio = contrastRatio(safeValue, contrastAgainst)
  const passes = ratio >= MIN_TEXT_CONTRAST

  return (
    <div className="space-y-2">
      <Label htmlFor={colorInputId}>{label}</Label>
      <div className="flex items-center gap-3">
        <input
          id={colorInputId}
          type="color"
          value={safeValue}
          onChange={handleColorChange}
          aria-label={label}
          className="h-10 w-14 cursor-pointer rounded-md border border-input bg-background p-1"
        />
        <Input
          id={hexInputId}
          type="text"
          value={value}
          onChange={handleHexChange}
          aria-label={`${label} hex`}
          className="font-mono uppercase"
          maxLength={7}
        />
      </div>
      {!passes && (
        <div
          role="alert"
          className="rounded-md border border-warning bg-bg-warning px-3 py-2 text-xs text-warning"
        >
          {t('settings.branding.contrastWarning', { ratio: ratio.toFixed(2) })}
        </div>
      )}
    </div>
  )
}
