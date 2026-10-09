import { renderHook } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useTabSwitchDetection } from './useTabSwitchDetection'

const mutate = vi.fn()
vi.mock('@/api/sessions', () => ({
  useReportEvent: () => ({ mutate }),
}))

function setVisibility(state: 'hidden' | 'visible') {
  Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => state })
  document.dispatchEvent(new Event('visibilitychange'))
}

/** Make the next mutate call resolve successfully with the given server response. */
function respondWith(data: { status?: string; warn?: boolean }) {
  mutate.mockImplementationOnce((_payload, opts) => opts.onSuccess(data))
}

describe('useTabSwitchDetection', () => {
  const onWarn = vi.fn()
  const onAutoSubmit = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    mutate.mockReset()
  })

  it('reports a tab_switch event when the page becomes hidden', () => {
    renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    setVisibility('hidden')
    expect(mutate).toHaveBeenCalledWith({ type: 'tab_switch' }, expect.any(Object))
  })

  it('does not report when the page becomes visible again', () => {
    renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    setVisibility('visible')
    expect(mutate).not.toHaveBeenCalled()
  })

  it('reports a blur event when the window loses focus', () => {
    renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    window.dispatchEvent(new Event('blur'))
    expect(mutate).toHaveBeenCalledWith({ type: 'blur' }, expect.any(Object))
  })

  it('invokes onWarn when the server asks to warn', () => {
    renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    respondWith({ status: 'in_progress', warn: true })
    window.dispatchEvent(new Event('blur'))
    expect(onWarn).toHaveBeenCalledTimes(1)
    expect(onAutoSubmit).not.toHaveBeenCalled()
  })

  it('invokes onAutoSubmit (and not onWarn) when the server auto-submitted the session', () => {
    renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    respondWith({ status: 'auto_submitted', warn: true })
    window.dispatchEvent(new Event('blur'))
    expect(onAutoSubmit).toHaveBeenCalledTimes(1)
    expect(onWarn).not.toHaveBeenCalled()
  })

  it('does nothing when the server response needs no action', () => {
    renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    respondWith({ status: 'in_progress', warn: false })
    window.dispatchEvent(new Event('blur'))
    expect(onWarn).not.toHaveBeenCalled()
    expect(onAutoSubmit).not.toHaveBeenCalled()
  })

  it('removes its listeners on unmount', () => {
    const { unmount } = renderHook(() => useTabSwitchDetection('s1', onWarn, onAutoSubmit))
    unmount()
    window.dispatchEvent(new Event('blur'))
    setVisibility('hidden')
    expect(mutate).not.toHaveBeenCalled()
  })
})
