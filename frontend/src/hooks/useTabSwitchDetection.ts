import { useEffect } from 'react'
import { useReportEvent, type EventResponse } from '@/api/sessions'

export function useTabSwitchDetection(
  sessionId: string,
  onWarn: () => void,
  onAutoSubmit: () => void,
) {
  const reportEvent = useReportEvent(sessionId)

  useEffect(() => {
    const handleEventResponse = (data: EventResponse) => {
      if (data.status === 'auto_submitted') {
        onAutoSubmit()
      } else if (data.warn) {
        onWarn()
      }
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'hidden') {
        reportEvent.mutate({ type: 'tab_switch' }, { onSuccess: handleEventResponse })
      }
    }

    const handleBlur = () => {
      reportEvent.mutate({ type: 'blur' }, { onSuccess: handleEventResponse })
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    window.addEventListener('blur', handleBlur)

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      window.removeEventListener('blur', handleBlur)
    }
    // reportEvent.mutate is stable; onWarn/onAutoSubmit are stable via useCallback in caller
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])
}
