import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

function formatRemaining(remainingMs: number): string {
  const totalSeconds = Math.floor(remainingMs / 1000)
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (days >= 1) {
    return `${days}d ${hours}h`
  }
  return [
    String(hours).padStart(2, '0'),
    String(minutes).padStart(2, '0'),
    String(seconds).padStart(2, '0'),
  ].join(':')
}

export function useCountdown(deadline: string | null): string {
  const { t } = useTranslation()
  const expiredStr = t('portal.deadline.expired')
  const noDeadlineStr = t('portal.card.noDeadline')

  const [display, setDisplay] = useState<string>(() => {
    if (!deadline) return noDeadlineStr
    const remaining = new Date(deadline).getTime() - Date.now()
    if (remaining <= 0) return expiredStr
    return formatRemaining(remaining)
  })

  useEffect(() => {
    if (!deadline) {
      setDisplay(noDeadlineStr)
      return
    }

    // ISS-027: declare interval with let before tick() so the closure
    // can safely call clearInterval(interval) without a TDZ ReferenceError.
    let interval: ReturnType<typeof setInterval>

    const tick = () => {
      const remaining = new Date(deadline).getTime() - Date.now()
      if (remaining <= 0) {
        setDisplay(expiredStr)
        clearInterval(interval)
        return
      }
      setDisplay(formatRemaining(remaining))
    }

    tick()
    interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [deadline, expiredStr, noDeadlineStr])

  return display
}
