import { render, screen, act } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { CountdownTimer } from '@/pages/ExamTaking/CountdownTimer'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

function renderTimer(remainingSeconds: number, totalSeconds = 600) {
  return render(
    <CountdownTimer
      remainingSeconds={remainingSeconds}
      totalSeconds={totalSeconds}
      onExpire={vi.fn()}
    />,
  )
}

describe('CountdownTimer', () => {
  it('renders without crashing', () => {
    renderTimer(300)
    // aria-live region is present
    expect(document.querySelector('[aria-live="assertive"]')).toBeInTheDocument()
  })

  it('live region is empty during normal countdown', () => {
    renderTimer(500)
    const liveRegion = document.querySelector('[aria-live="assertive"]')
    expect(liveRegion?.textContent).toBe('')
  })

  it('announces 5-minute warning at exactly 300 seconds', () => {
    renderTimer(300)
    const liveRegion = document.querySelector('[aria-live="assertive"]')
    expect(liveRegion?.textContent).toBe('session.timerFiveMinutes')
  })

  it('announces 1-minute warning at exactly 60 seconds', () => {
    renderTimer(60)
    const liveRegion = document.querySelector('[aria-live="assertive"]')
    expect(liveRegion?.textContent).toBe('session.timerOneMinute')
  })

  it('announces expiry at 0 seconds', () => {
    renderTimer(0)
    const liveRegion = document.querySelector('[aria-live="assertive"]')
    expect(liveRegion?.textContent).toBe('session.timerExpired')
  })

  it('live region is empty at values other than thresholds', () => {
    renderTimer(250)
    const liveRegion = document.querySelector('[aria-live="assertive"]')
    expect(liveRegion?.textContent).toBe('')
  })

  it('visible timer display is aria-hidden', () => {
    renderTimer(300)
    const visibleTimer = document.querySelector('[aria-hidden="true"]')
    expect(visibleTimer).toBeInTheDocument()
  })

  it('calls onExpire when remainingSeconds reaches 0', () => {
    const onExpire = vi.fn()
    render(
      <CountdownTimer remainingSeconds={0} totalSeconds={600} onExpire={onExpire} />,
    )
    expect(onExpire).toHaveBeenCalledTimes(1)
  })

  it('live region switches to empty when moving away from threshold', () => {
    const { rerender } = renderTimer(300)
    const liveRegion = document.querySelector('[aria-live="assertive"]')
    expect(liveRegion?.textContent).toBe('session.timerFiveMinutes')

    act(() => {
      rerender(
        <CountdownTimer remainingSeconds={299} totalSeconds={600} onExpire={vi.fn()} />,
      )
    })
    expect(liveRegion?.textContent).toBe('')
  })
})
