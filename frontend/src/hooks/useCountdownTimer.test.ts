import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useCountdownTimer } from './useCountdownTimer'

const NOW = new Date('2026-01-01T12:00:00Z')
const inSeconds = (s: number) => new Date(NOW.getTime() + s * 1000).toISOString()

describe('useCountdownTimer', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('derives remaining seconds from the expiry timestamp, ignoring the initial value', () => {
    const { result } = renderHook(() => useCountdownTimer(999, inSeconds(90)))
    expect(result.current.remaining).toBe(90)
  })

  it('counts down each second', () => {
    const { result } = renderHook(() => useCountdownTimer(0, inSeconds(10)))
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(result.current.remaining).toBe(7)
  })

  it('never goes below zero once expired', () => {
    const { result } = renderHook(() => useCountdownTimer(0, inSeconds(2)))
    act(() => {
      vi.advanceTimersByTime(10_000)
    })
    expect(result.current.remaining).toBe(0)
  })

  it('starts at zero for an expiry already in the past', () => {
    const { result } = renderHook(() => useCountdownTimer(0, inSeconds(-5)))
    expect(result.current.remaining).toBe(0)
  })

  it('re-anchors to the server value on syncFromServer and keeps ticking from there', () => {
    const { result } = renderHook(() => useCountdownTimer(0, inSeconds(100)))
    act(() => {
      result.current.syncFromServer(30.9)
    })
    expect(result.current.remaining).toBe(30)
    act(() => {
      vi.advanceTimersByTime(5000)
    })
    expect(result.current.remaining).toBe(25)
  })

  it('clamps negative server values to zero', () => {
    const { result } = renderHook(() => useCountdownTimer(0, inSeconds(100)))
    act(() => {
      result.current.syncFromServer(-4)
    })
    expect(result.current.remaining).toBe(0)
  })

  it('stops its interval on unmount', () => {
    const { unmount } = renderHook(() => useCountdownTimer(0, inSeconds(100)))
    expect(vi.getTimerCount()).toBe(1)
    unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
