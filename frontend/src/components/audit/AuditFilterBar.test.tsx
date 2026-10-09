import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import '@/i18n'
import { AuditFilterBar } from './AuditFilterBar'
import { AUDIT_ACTIONS } from '@/api/audit'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('AuditFilterBar', () => {
  it('hides the clear button when no filters are set', () => {
    render(<AuditFilterBar filters={{}} onChange={vi.fn()} />)
    expect(screen.queryByText('Clear Filters')).toBeNull()
  })

  it('debounces actor input by 300ms', () => {
    const onChange = vi.fn()
    render(<AuditFilterBar filters={{ entityType: 'user' }} onChange={onChange} />)
    const input = screen.getByPlaceholderText('Actor name')
    fireEvent.change(input, { target: { value: 'al' } })
    fireEvent.change(input, { target: { value: 'alice' } })
    act(() => { vi.advanceTimersByTime(299) })
    expect(onChange).not.toHaveBeenCalled()
    act(() => { vi.advanceTimersByTime(1) })
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ entityType: 'user', actor: 'alice' })
  })

  it('toggles actions on and off', () => {
    const onChange = vi.fn()
    const first = AUDIT_ACTIONS[0]
    const { rerender } = render(<AuditFilterBar filters={{}} onChange={onChange} />)
    fireEvent.click(screen.getByText(first))
    expect(onChange).toHaveBeenLastCalledWith({ actions: [first] })
    rerender(<AuditFilterBar filters={{ actions: [first] }} onChange={onChange} />)
    fireEvent.click(screen.getByText(first))
    expect(onChange).toHaveBeenLastCalledWith({ actions: undefined })
  })

  it('changes entity type and clears all filters', () => {
    const onChange = vi.fn()
    render(<AuditFilterBar filters={{ actor: 'bob' }} onChange={onChange} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'exam' } })
    expect(onChange).toHaveBeenLastCalledWith({ actor: 'bob', entityType: 'exam' })
    fireEvent.click(screen.getByText('Clear Filters'))
    expect(onChange).toHaveBeenLastCalledWith({})
    expect((screen.getByPlaceholderText('Actor name') as HTMLInputElement).value).toBe('')
  })

  it('emits ISO dates for from/to and undefined when cleared', () => {
    const onChange = vi.fn()
    const { container } = render(
      <AuditFilterBar filters={{ from: '2026-10-01T10:00:00Z' }} onChange={onChange} />,
    )
    const [from, to] = Array.from(
      container.querySelectorAll('input[type="datetime-local"]'),
    ) as HTMLInputElement[]
    expect(from.value).toBe('2026-10-01T10:00')
    fireEvent.change(from, { target: { value: '' } })
    expect(onChange).toHaveBeenLastCalledWith({ from: undefined })
    fireEvent.change(to, { target: { value: '2026-10-02T09:30' } })
    const arg = onChange.mock.calls[onChange.mock.calls.length - 1][0]
    expect(arg.to).toBe(new Date('2026-10-02T09:30').toISOString())
  })
})
