import { useEffect, useRef, useState } from 'react'
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
  const [announcement, setAnnouncement] = useState('')

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

  // Announce at key thresholds for screen readers (AC-4)
  useEffect(() => {
    if (remainingSeconds === 300) setAnnouncement(t('session.timerFiveMinutes'))
    else if (remainingSeconds === 60) setAnnouncement(t('session.timerOneMinute'))
    else if (remainingSeconds <= 0) setAnnouncement(t('session.timerExpired'))
    else setAnnouncement('')
  }, [remainingSeconds, t])

  return (
    <>
      {/* Screen reader live region — only speaks at key thresholds */}
      <div aria-live="assertive" aria-atomic="true" className="sr-only">
        {announcement}
      </div>
      <div
        aria-hidden="true"
        className={cn(
          'font-mono text-sm font-semibold px-2 py-1 rounded',
          isCritical
            ? 'bg-bg-danger text-danger'
            : isWarning
              ? 'bg-bg-warning text-warning'
              : 'bg-muted text-foreground',
        )}
        title={
          isCritical
            ? t('exam.taking.timer.critical')
            : isWarning
              ? t('exam.taking.timer.warning')
              : undefined
        }
      >
        {formatTime(remainingSeconds)}
      </div>
    </>
  )
}
