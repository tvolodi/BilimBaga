import { useTranslation } from 'react-i18next'

interface ShortTextInputProps {
  value: string
  onChange: (text: string) => void
}

export function ShortTextInput({ value, onChange }: ShortTextInputProps) {
  const { t } = useTranslation()

  return (
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      rows={4}
      className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
      placeholder={t('exam.taking.shortTextPlaceholder')}
    />
  )
}
