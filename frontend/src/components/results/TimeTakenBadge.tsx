import { useTranslation } from 'react-i18next'

interface TimeTakenBadgeProps {
  seconds: number | null
}

export function TimeTakenBadge({ seconds }: TimeTakenBadgeProps) {
  const { t } = useTranslation()

  if (seconds === null) return null

  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  const formatted = `${minutes}m ${remainingSeconds}s`

  return (
    <p className="text-sm text-muted-foreground">
      <span className="font-medium">{t('result.time_taken')}: </span>
      {formatted}
    </p>
  )
}
