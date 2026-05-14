import { useId, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

interface ColorPickerFieldProps {
  label: string
  value: string
  onChange: (color: string) => void
  contrastAgainst: string
}

function normaliseHex(input: string): string | null {
  const trimmed = input.trim().replace(/^#/, '')
  if (!/^([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(trimmed)) return null
  const expanded =
    trimmed.length === 3
      ? trimmed
          .split('')
          .map((c) => c + c)
          .join('')
      : trimmed
  return `#${expanded.toLowerCase()}`
}

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace(/^#/, '')
  const full =
    h.length === 3
      ? h
          .split('')
          .map((c) => c + c)
          .join('')
      : h
  const num = parseInt(full, 16)
  return [(num >> 16) & 0xff, (num >> 8) & 0xff, num & 0xff]
}

function relativeLuminance(hex: string): number {
  const rgb = hexToRgb(hex)
  return rgb
    .map((c) => {
      const s = c / 255
      return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
    })
    .reduce((acc, c, i) => acc + c * [0.2126, 0.7152, 0.0722][i], 0)
}

export function contrastRatio(hex1: string, hex2: string): number {
  const l1 = relativeLuminance(hex1)
  const l2 = relativeLuminance(hex2)
  const [lighter, darker] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (lighter + 0.05) / (darker + 0.05)
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
  const passes = ratio >= 4.5

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
          className="rounded-md border border-yellow-400 bg-yellow-50 px-3 py-2 text-xs text-yellow-800"
        >
          {t('settings.branding.contrastWarning', { ratio: ratio.toFixed(2) })}
        </div>
      )}
    </div>
  )
}
