import { useEffect, useRef, useState } from 'react'
import { useReportEvent, type EventResponse, type EventPayload } from '@/api/sessions'

/** FR-BB319 AC-5: a window blur is reported only if it survives this delay (a tab switch also hides the page). */
const BLUR_DELAY_MS = 300
/** FR-BB319 AC-7: a fullscreen exit is reported only if it survives this delay. Separate from the blur timer. */
const FULLSCREEN_DELAY_MS = 300

/** True when a fullscreen element is active. Boolean so that a missing value (undefined) counts as not fullscreen. */
function isFullscreenActive(): boolean {
  return Boolean(document.fullscreenElement)
}

export interface TabSwitchDetection {
  /** True once the document has been in fullscreen during this exam page (FR-BB319 AC-7). */
  enteredFullscreen: boolean
  /** Mirror of `isFullscreenActive()`, updated from the same fullscreenchange handler. */
  isFullscreen: boolean
}

/**
 * Detects tab switches, window blur and fullscreen exit and reports each physical incident once
 * (FR-BB319). Event rules are ordered: a tab switch suppresses the blur that accompanies it, and a
 * fullscreen exit that is part of an Alt-Tab (blur too) is reported once.
 */
export function useTabSwitchDetection(
  sessionId: string,
  onWarn: (eventCount: number) => void,
  onAutoSubmit: () => void,
  enabled = true,
): TabSwitchDetection {
  const { mutate } = useReportEvent(sessionId)

  // Flags read by the listeners live in refs so that changing them never re-runs the effect.
  const hiddenReportedRef = useRef(false)
  const blurReportedRef = useRef(false)
  const exitReportedRef = useRef(false)
  const lastTabSwitchAtRef = useRef<number | null>(null)
  const enteredFullscreenRef = useRef(isFullscreenActive())

  const [enteredFullscreen, setEnteredFullscreen] = useState(enteredFullscreenRef.current)
  const [isFullscreen, setIsFullscreen] = useState(isFullscreenActive())

  // Latest callbacks and mutate function, read at event time (the effect depends on sessionId and enabled only).
  const mutateRef = useRef(mutate)
  mutateRef.current = mutate
  const onWarnRef = useRef(onWarn)
  onWarnRef.current = onWarn
  const onAutoSubmitRef = useRef(onAutoSubmit)
  onAutoSubmitRef.current = onAutoSubmit

  useEffect(() => {
    if (!enabled) return

    let blurTimer: ReturnType<typeof setTimeout> | undefined
    let fullscreenTimer: ReturnType<typeof setTimeout> | undefined

    const handleResponse = (data: EventResponse) => {
      if (data.status === 'auto_submitted') {
        onAutoSubmitRef.current()
      } else if (data.warn) {
        onWarnRef.current(data.event_count ?? 0)
      }
    }

    const send = (payload: EventPayload) => {
      mutateRef.current(payload, { onSuccess: handleResponse })
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'hidden') {
        // AC-4: a repeated hidden notification without a return to 'visible' sends nothing.
        if (hiddenReportedRef.current) return
        hiddenReportedRef.current = true
        lastTabSwitchAtRef.current = Date.now()
        // AC-6: the blur that accompanies a tab switch is never sent.
        blurReportedRef.current = true
        send({ type: 'tab_switch' })
      } else {
        // AC-4 / AC-6: returning to the page clears both flags (a browser may return without a window focus).
        hiddenReportedRef.current = false
        blurReportedRef.current = false
      }
    }

    const handleBlur = () => {
      // AC-5: restart the 300 ms window on every blur.
      clearTimeout(blurTimer)
      blurTimer = setTimeout(() => {
        blurTimer = undefined
        if (document.visibilityState === 'hidden') return
        if (hiddenReportedRef.current || blurReportedRef.current) return
        blurReportedRef.current = true
        send({ type: 'blur' })
      }, BLUR_DELAY_MS)
    }

    const handleFocus = () => {
      // AC-6: focus clears blurReported so the next focus loss is reported again.
      blurReportedRef.current = false
    }

    const handleFullscreenChange = () => {
      const active = isFullscreenActive()
      setIsFullscreen(active)

      if (active) {
        // Re-entry: the next exit is a new incident (AC-7). A pending exit timer is no longer valid.
        // Re-entering fullscreen means the window has focus again, so the blur flag of the previous exit is cleared too.
        exitReportedRef.current = false
        blurReportedRef.current = false
        clearTimeout(fullscreenTimer)
        fullscreenTimer = undefined
        if (!enteredFullscreenRef.current) {
          enteredFullscreenRef.current = true
          setEnteredFullscreen(true)
        }
        return
      }

      if (!enteredFullscreenRef.current) return
      clearTimeout(fullscreenTimer)
      fullscreenTimer = setTimeout(() => {
        fullscreenTimer = undefined
        if (isFullscreenActive()) return
        if (exitReportedRef.current) return
        if (document.visibilityState === 'hidden') return
        if (hiddenReportedRef.current || blurReportedRef.current) return
        exitReportedRef.current = true
        // A sent fullscreen_exit suppresses a following blur for the same incident.
        blurReportedRef.current = true
        send({ type: 'fullscreen_exit' })
      }, FULLSCREEN_DELAY_MS)
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    window.addEventListener('blur', handleBlur)
    window.addEventListener('focus', handleFocus)

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      document.removeEventListener('fullscreenchange', handleFullscreenChange)
      window.removeEventListener('blur', handleBlur)
      window.removeEventListener('focus', handleFocus)
      clearTimeout(blurTimer)
      clearTimeout(fullscreenTimer)
    }
  }, [sessionId, enabled])

  return { enteredFullscreen, isFullscreen }
}
