import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { TabSwitchWarningModal } from '../TabSwitchWarningModal'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { count?: number }) =>
      opts?.count !== undefined ? `${key}:${opts.count}` : key,
  }),
}))

function setup(overrides: Partial<Parameters<typeof TabSwitchWarningModal>[0]> = {}) {
  const props = {
    open: true,
    onClose: vi.fn(),
    count: 0,
    canReturnFullscreen: false,
    onReturnFullscreen: vi.fn(),
    ...overrides,
  }
  render(<TabSwitchWarningModal {...props} />)
  return props
}

describe('TabSwitchWarningModal', () => {
  afterEach(() => {
    Reflect.deleteProperty(document.documentElement, 'requestFullscreen')
  })

  it('hides the count line when count is below 1', () => {
    setup({ count: 0 })
    expect(screen.queryByText(/exam\.taking\.tabswitch\.count/)).toBeNull()
  })

  it('shows the running count from the server when count is 1 or more', () => {
    setup({ count: 3 })
    expect(screen.getByText('exam.taking.tabswitch.count:3')).toBeInTheDocument()
  })

  it('hides the return button when the candidate cannot return to fullscreen', () => {
    setup({ canReturnFullscreen: false })
    expect(screen.queryByRole('button', { name: 'exam.taking.tabswitch.returnFullscreen' })).toBeNull()
  })

  it('shows the return button when canReturnFullscreen is true', () => {
    setup({ canReturnFullscreen: true })
    expect(screen.getByRole('button', { name: 'exam.taking.tabswitch.returnFullscreen' })).toBeInTheDocument()
  })

  it('return button requests fullscreen and calls onReturnFullscreen', async () => {
    const request = vi.fn(() => Promise.resolve())
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    const props = setup({ canReturnFullscreen: true })
    await userEvent.click(screen.getByRole('button', { name: 'exam.taking.tabswitch.returnFullscreen' }))
    expect(request).toHaveBeenCalledTimes(1)
    expect(props.onReturnFullscreen).toHaveBeenCalledTimes(1)
  })

  it('return button ignores a rejected fullscreen request and still calls onReturnFullscreen', async () => {
    const request = vi.fn(() => Promise.reject(new Error('denied')))
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    const props = setup({ canReturnFullscreen: true })
    await userEvent.click(screen.getByRole('button', { name: 'exam.taking.tabswitch.returnFullscreen' }))
    expect(props.onReturnFullscreen).toHaveBeenCalledTimes(1)
  })

  it('the close button calls onClose and does not request fullscreen', async () => {
    const request = vi.fn(() => Promise.resolve())
    Object.defineProperty(document.documentElement, 'requestFullscreen', { configurable: true, value: request })
    const props = setup({ canReturnFullscreen: true, count: 2 })
    await userEvent.click(screen.getByRole('button', { name: 'common.cancel' }))
    expect(props.onClose).toHaveBeenCalledTimes(1)
    expect(request).not.toHaveBeenCalled()
  })
})
