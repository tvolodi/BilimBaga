import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useTabSwitchDetection } from './useTabSwitchDetection'

const mutate = vi.fn()
vi.mock('@/api/sessions', () => ({
  useReportEvent: () => ({ mutate }),
}))

// jsdom has no fullscreen API state of its own: back the two document properties with test-controlled values.
let visibility: DocumentVisibilityState = 'visible'
let fullscreenEl: Element | null = null
Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => visibility })
Object.defineProperty(document, 'fullscreenElement', { configurable: true, get: () => fullscreenEl })

const fakeElement = {} as Element

function setVisibility(state: DocumentVisibilityState) {
  visibility = state
  act(() => {
    document.dispatchEvent(new Event('visibilitychange'))
  })
}

/** Change visibility without dispatching visibilitychange (a page that is hidden but never notified). */
function setVisibilitySilently(state: DocumentVisibilityState) {
  visibility = state
}

function enterFullscreen() {
  fullscreenEl = fakeElement
  act(() => {
    document.dispatchEvent(new Event('fullscreenchange'))
  })
}

function exitFullscreen() {
  fullscreenEl = null
  act(() => {
    document.dispatchEvent(new Event('fullscreenchange'))
  })
}

function blur() {
  act(() => {
    window.dispatchEvent(new Event('blur'))
  })
}

function focus() {
  act(() => {
    window.dispatchEvent(new Event('focus'))
  })
}

/** Event types sent so far, in order. */
function sentTypes(): string[] {
  return mutate.mock.calls.map((call) => (call[0] as { type: string }).type)
}

/** Make the next mutate call resolve successfully with the given server response. */
function respondWith(data: { status?: string; warn?: boolean; event_count?: number }) {
  mutate.mockImplementationOnce((_payload, opts) => opts.onSuccess(data))
}

const onWarn = vi.fn()
const onAutoSubmit = vi.fn()

function renderDetection(enabled = true) {
  return renderHook(
    ({ on }: { on: boolean }) => useTabSwitchDetection('s1', onWarn, onAutoSubmit, on),
    { initialProps: { on: enabled } },
  )
}

describe('useTabSwitchDetection', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    mutate.mockReset()
    visibility = 'visible'
    fullscreenEl = null
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('basic reporting', () => {
    it('reports a tab_switch event when the page becomes hidden', () => {
      renderDetection()
      setVisibility('hidden')
      expect(mutate).toHaveBeenCalledWith({ type: 'tab_switch' }, expect.any(Object))
    })

    it('does not report when the page becomes visible again', () => {
      renderDetection()
      setVisibility('visible')
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).not.toHaveBeenCalled()
    })

    it('reports a blur event after the 300 ms window when the window loses focus', () => {
      renderDetection()
      blur()
      expect(mutate).not.toHaveBeenCalled()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).toHaveBeenCalledWith({ type: 'blur' }, expect.any(Object))
    })

    it('passes the server event_count to onWarn', () => {
      renderDetection()
      respondWith({ status: 'in_progress', warn: true, event_count: 2 })
      setVisibility('hidden')
      expect(onWarn).toHaveBeenCalledTimes(1)
      expect(onWarn).toHaveBeenCalledWith(2)
      expect(onAutoSubmit).not.toHaveBeenCalled()
    })

    it('passes 0 to onWarn when the server omits event_count', () => {
      renderDetection()
      respondWith({ status: 'in_progress', warn: true })
      setVisibility('hidden')
      expect(onWarn).toHaveBeenCalledWith(0)
    })

    it('invokes onAutoSubmit (and not onWarn) when the server auto-submitted the session', () => {
      renderDetection()
      respondWith({ status: 'auto_submitted', warn: true })
      setVisibility('hidden')
      expect(onAutoSubmit).toHaveBeenCalledTimes(1)
      expect(onWarn).not.toHaveBeenCalled()
    })

    it('does nothing when the server response needs no action', () => {
      renderDetection()
      respondWith({ status: 'in_progress', warn: false })
      setVisibility('hidden')
      expect(onWarn).not.toHaveBeenCalled()
      expect(onAutoSubmit).not.toHaveBeenCalled()
    })
  })

  // AC-12 (a)-(e4): the ordered rules.
  describe('ordered rules (FR-BB319 AC-4 to AC-7)', () => {
    it('(a) blur then visibilitychange hidden within 300 ms sends exactly one tab_switch and no blur', () => {
      renderDetection()
      blur()
      setVisibility('hidden')
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['tab_switch'])
    })

    it('(b) visibilitychange hidden then blur sends exactly one tab_switch', () => {
      renderDetection()
      setVisibility('hidden')
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['tab_switch'])
    })

    it('(c) blur with a visible document sends one blur after 300 ms and no second blur until focus', () => {
      renderDetection()
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['blur'])

      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['blur'])

      focus()
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['blur', 'blur'])
    })

    it('(d) two hidden notifications without visible in between send one tab_switch', () => {
      renderDetection()
      setVisibility('hidden')
      setVisibility('hidden')
      expect(sentTypes()).toEqual(['tab_switch'])
    })

    it('(e) fullscreenchange to null sends one fullscreen_exit per exit after 300 ms, after re-entry and a second exit', () => {
      const { result } = renderDetection()
      enterFullscreen()
      expect(result.current.enteredFullscreen).toBe(true)

      exitFullscreen()
      expect(sentTypes()).toEqual([])
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['fullscreen_exit'])

      enterFullscreen()
      exitFullscreen()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['fullscreen_exit', 'fullscreen_exit'])
    })

    it('(e) sends no fullscreen_exit when fullscreen was never entered', () => {
      renderDetection()
      fullscreenEl = null
      act(() => {
    document.dispatchEvent(new Event('fullscreenchange'))
  })
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual([])
    })

    it('(e) sends no fullscreen_exit when the document is hidden at exit time', () => {
      renderDetection()
      enterFullscreen()
      setVisibilitySilently('hidden')
      exitFullscreen()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual([])
    })

    it('(e) reports only tab_switch when a hide and the fullscreen exit happen together (hiddenReported)', () => {
      renderDetection()
      enterFullscreen()
      setVisibility('hidden')
      exitFullscreen()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['tab_switch'])
    })

    it('(e2) visibilitychange to visible clears blurReported so a later bare blur is sent without a focus event', () => {
      renderDetection()
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['blur'])

      setVisibility('visible')
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['blur', 'blur'])
    })

    it('(e3) a tab_switch after an already reported blur is sent', () => {
      renderDetection()
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['blur'])

      setVisibility('hidden')
      expect(sentTypes()).toEqual(['blur', 'tab_switch'])
    })

    it('(e4) blur then fullscreenchange to null within 300 ms (Alt-Tab out of fullscreen) sends exactly one event', () => {
      renderDetection()
      enterFullscreen()
      blur()
      act(() => {
        vi.advanceTimersByTime(100)
      })
      exitFullscreen()
      act(() => {
        vi.advanceTimersByTime(1000)
      })
      expect(sentTypes()).toHaveLength(1)
    })

    it('(e4) fullscreenchange to null then blur within 300 ms sends exactly one event', () => {
      renderDetection()
      enterFullscreen()
      exitFullscreen()
      act(() => {
        vi.advanceTimersByTime(100)
      })
      blur()
      act(() => {
        vi.advanceTimersByTime(1000)
      })
      expect(sentTypes()).toEqual(['fullscreen_exit'])
    })

    it('a sent fullscreen_exit suppresses a following blur for the same incident', () => {
      renderDetection()
      enterFullscreen()
      exitFullscreen()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['fullscreen_exit'])
    })

    it('a blur that is dropped because the page is hidden does not block the next blur after focus', () => {
      renderDetection()
      setVisibility('hidden')
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['tab_switch'])
      setVisibility('visible')
      focus()
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(sentTypes()).toEqual(['tab_switch', 'blur'])
    })
  })

  // AC-12 (f): enabled flag and listener cleanup.
  describe('enabled flag and cleanup (FR-BB319 AC-4 to AC-7)', () => {
    it('(f) enabled = false registers no listeners and reports nothing', () => {
      const docAdd = vi.spyOn(document, 'addEventListener')
      const winAdd = vi.spyOn(window, 'addEventListener')
      renderDetection(false)

      const docEvents = docAdd.mock.calls.map((c) => c[0])
      const winEvents = winAdd.mock.calls.map((c) => c[0])
      expect(docEvents).not.toContain('visibilitychange')
      expect(docEvents).not.toContain('fullscreenchange')
      expect(winEvents).not.toContain('blur')
      expect(winEvents).not.toContain('focus')

      blur()
      setVisibility('hidden')
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).not.toHaveBeenCalled()
      docAdd.mockRestore()
      winAdd.mockRestore()
    })

    it('(f) unmount removes visibilitychange, fullscreenchange (document) and blur, focus (window)', () => {
      const docRemove = vi.spyOn(document, 'removeEventListener')
      const winRemove = vi.spyOn(window, 'removeEventListener')
      const { unmount } = renderDetection()
      unmount()

      expect(docRemove).toHaveBeenCalledWith('visibilitychange', expect.any(Function))
      expect(docRemove).toHaveBeenCalledWith('fullscreenchange', expect.any(Function))
      expect(winRemove).toHaveBeenCalledWith('blur', expect.any(Function))
      expect(winRemove).toHaveBeenCalledWith('focus', expect.any(Function))
      docRemove.mockRestore()
      winRemove.mockRestore()
    })

    it('(f) unmount clears a pending blur timer', () => {
      const { unmount } = renderDetection()
      blur()
      unmount()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).not.toHaveBeenCalled()
    })

    it('(f) unmount clears a pending fullscreen timer', () => {
      const { unmount } = renderDetection()
      enterFullscreen()
      exitFullscreen()
      unmount()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).not.toHaveBeenCalled()
    })

    it('(f) turning enabled to false removes listeners and clears a pending blur timer', () => {
      const { rerender } = renderDetection(true)
      blur()
      rerender({ on: false })
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).not.toHaveBeenCalled()
      blur()
      act(() => {
        vi.advanceTimersByTime(300)
      })
      expect(mutate).not.toHaveBeenCalled()
    })
  })

  describe('fullscreen state', () => {
    it('starts with enteredFullscreen false when the page is not fullscreen at mount', () => {
      const { result } = renderDetection()
      expect(result.current.enteredFullscreen).toBe(false)
      expect(result.current.isFullscreen).toBe(false)
    })

    it('reports isFullscreen and enteredFullscreen as the document changes', () => {
      const { result } = renderDetection()
      enterFullscreen()
      expect(result.current.isFullscreen).toBe(true)
      expect(result.current.enteredFullscreen).toBe(true)
      exitFullscreen()
      expect(result.current.isFullscreen).toBe(false)
      expect(result.current.enteredFullscreen).toBe(true)
    })
  })
})
