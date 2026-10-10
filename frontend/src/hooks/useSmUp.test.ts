import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useSmUp } from './useSmUp'

interface MediaStub {
  matches: boolean
  listeners: Set<() => void>
}

// jsdom has no matchMedia. The stub answers the sm query and lets a test fire a viewport change.
function stubMatchMedia(matches: boolean): MediaStub {
  const stub: MediaStub = { matches, listeners: new Set() }
  window.matchMedia = vi.fn(
    () =>
      ({
        get matches() {
          return stub.matches
        },
        addEventListener: (_type: string, listener: () => void) => stub.listeners.add(listener),
        removeEventListener: (_type: string, listener: () => void) => stub.listeners.delete(listener),
      }) as unknown as MediaQueryList,
  )
  return stub
}

afterEach(() => {
  delete (window as { matchMedia?: unknown }).matchMedia
})

describe('useSmUp (FR-BB321 AC-6, #472)', () => {
  it('is false below the sm breakpoint', () => {
    stubMatchMedia(false)
    const { result } = renderHook(() => useSmUp())
    expect(result.current).toBe(false)
  })

  it('is true at the sm breakpoint and up', () => {
    stubMatchMedia(true)
    const { result } = renderHook(() => useSmUp())
    expect(result.current).toBe(true)
  })

  it('treats the layout as wide when matchMedia is missing', () => {
    const { result } = renderHook(() => useSmUp())
    expect(result.current).toBe(true)
  })

  it('follows a viewport change while mounted', () => {
    const stub = stubMatchMedia(false)
    const { result } = renderHook(() => useSmUp())

    act(() => {
      stub.matches = true
      stub.listeners.forEach((listener) => listener())
    })
    expect(result.current).toBe(true)
  })

  it('removes its listener on unmount', () => {
    const stub = stubMatchMedia(false)
    const { unmount } = renderHook(() => useSmUp())
    expect(stub.listeners.size).toBe(1)

    unmount()
    expect(stub.listeners.size).toBe(0)
  })
})
