import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import '@/i18n'
import { AuditFilterBar } from './AuditFilterBar'
import { useAuditFilterOptions, type AuditFilterOptions } from '@/api/audit'

// The options come from the tenant's audit log through a query (#557). Each test chooses the query state it needs:
// loaded, loading or failed.
vi.mock('@/api/audit', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/audit')>()),
  useAuditFilterOptions: vi.fn(),
}))

const OPTIONS: AuditFilterOptions = {
  actions: ['auth.login.success', 'answer.ai_grade', 'exam.create'],
  entity_types: ['exam', 'user'],
}

function mockOptions(state: { data?: AuditFilterOptions; isPending?: boolean; isError?: boolean }) {
  vi.mocked(useAuditFilterOptions).mockReturnValue({
    data: state.data,
    isPending: state.isPending ?? false,
    isError: state.isError ?? false,
  } as unknown as ReturnType<typeof useAuditFilterOptions>)
}

beforeEach(() => {
  vi.useFakeTimers()
  mockOptions({ data: OPTIONS })
})
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
    const first = OPTIONS.actions[0]
    const { rerender } = render(<AuditFilterBar filters={{}} onChange={onChange} />)
    fireEvent.click(screen.getByText(first))
    expect(onChange).toHaveBeenLastCalledWith({ actions: [first] })
    rerender(<AuditFilterBar filters={{ actions: [first] }} onChange={onChange} />)
    fireEvent.click(screen.getByText(first))
    expect(onChange).toHaveBeenLastCalledWith({ actions: undefined })
  })

  it('filters by the AI grading action (#540)', () => {
    const onChange = vi.fn()
    render(<AuditFilterBar filters={{}} onChange={onChange} />)
    fireEvent.click(screen.getByText('answer.ai_grade'))
    expect(onChange).toHaveBeenLastCalledWith({ actions: ['answer.ai_grade'] })
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

  // #557: the dropdown and the chips are the tenant's own codes, in the order the backend sends them.
  it('offers the actions and entity types the tenant log holds', () => {
    render(<AuditFilterBar filters={{}} onChange={vi.fn()} />)
    for (const action of OPTIONS.actions) {
      expect(screen.getByText(action)).toBeTruthy()
    }
    const values = Array.from((screen.getByRole('combobox') as HTMLSelectElement).options).map((o) => o.value)
    expect(values).toEqual(['', 'exam', 'user'])
  })

  it('disables the dropdown and the chips while the options load', () => {
    mockOptions({ isPending: true })
    const { container } = render(<AuditFilterBar filters={{}} onChange={vi.fn()} />)
    expect((screen.getByRole('combobox') as HTMLSelectElement).disabled).toBe(true)
    expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0)
    expect(screen.queryByText('auth.login.success')).toBeNull()
  })

  it('shows an inline error when the options fail, and the date, actor and clear controls still work', () => {
    mockOptions({ isError: true })
    const onChange = vi.fn()
    const { container } = render(<AuditFilterBar filters={{ actor: 'bob' }} onChange={onChange} />)
    expect(screen.getByRole('alert').textContent).toBe(
      'Filter options could not be loaded. You can still filter by date and actor.',
    )
    expect(container.querySelectorAll('input[type="datetime-local"]')).toHaveLength(2)
    fireEvent.click(screen.getByText('Clear Filters'))
    expect(onChange).toHaveBeenLastCalledWith({})
  })
})
