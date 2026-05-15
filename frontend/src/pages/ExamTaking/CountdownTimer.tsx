import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

interface CountdownTimerProps {
  remainingSeconds: number
  totalSeconds: number
  onExpire: () => void
}

function formatTime(seconds: number): string {
  const s = Math.max(0, seconds)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) {
    return [String(h).padStart(2, '0'), String(m).padStart(2, '0'), String(sec).padStart(2, '0')].join(':')
  }
  return [String(m).padStart(2, '0'), String(sec).padStart(2, '0')].join(':')
}

export function CountdownTimer({ remainingSeconds, totalSeconds, onExpire }: CountdownTimerProps) {
  const { t } = useTranslation()
  const expiredRef = useRef(false)

  const pct = totalSeconds > 0 ? remainingSeconds / totalSeconds : 0
  const isCritical = pct <= 0.05
  const isWarning = pct <= 0.2 && !isCritical

  // Trigger expiry callback once when reaching 0
  useEffect(() => {
    if (remainingSeconds <= 0 && !expiredRef.current) {
      expiredRef.current = true
      onExpire()
    }
  }, [remainingSeconds, onExpire])

  return (
    <div
      className={cn(
        'font-mono text-sm font-semibold px-2 py-1 rounded',
        isCritical
          ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
          : isWarning
            ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
            : 'bg-muted text-foreground',
      )}
      title={
        isCritical
          ? t('exam.taking.timer.critical')
          : isWarning
            ? t('exam.taking.timer.warning')
            : undefined
      }
      aria-label={`${t('exam.taking.timer.warning')} ${formatTime(remainingSeconds)}`}
    >
      {formatTime(remainingSeconds)}
    </div>
  )
}
