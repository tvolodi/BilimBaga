import { useState, useEffect, useRef, useCallback } from 'react'

export interface CountdownTimerResult {
  remaining: number
  syncFromServer: (serverRemainingSeconds: number) => void
}

export function useCountdownTimer(
  _initialSeconds: number,
  expiresAt: string,
): CountdownTimerResult {
  const effectiveExpiryRef = useRef<number>(new Date(expiresAt).getTime())

  const [remaining, setRemaining] = useState<number>(() => {
    const fromExpiry = Math.floor((effectiveExpiryRef.current - Date.now()) / 1000)
    return Math.max(0, fromExpiry)
  })

  useEffect(() => {
    effectiveExpiryRef.current = new Date(expiresAt).getTime()

    const tick = () => {
      const fromExpiry = Math.floor((effectiveExpiryRef.current - Date.now()) / 1000)
      setRemaining(Math.max(0, fromExpiry))
    }

    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [expiresAt])

  const syncFromServer = useCallback((serverRemainingSeconds: number) => {
    const adjusted = Math.max(0, Math.floor(serverRemainingSeconds))
    effectiveExpiryRef.current = Date.now() + adjusted * 1000
    setRemaining(adjusted)
  }, [])

  return { remaining, syncFromServer }
}
