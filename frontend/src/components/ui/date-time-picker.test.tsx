import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DateTimePicker } from './date-time-picker'

/** The picker renders two native inputs: a date input (linked to the label) and a time input. */
function getInputs(container: HTMLElement) {
  const date = container.querySelector<HTMLInputElement>('input[type="date"]')
  const time = container.querySelector<HTMLInputElement>('input[type="time"]')
  if (!date || !time) throw new Error('DateTimePicker did not render both inputs')
  return { date, time }
}

describe('DateTimePicker', () => {
  it('links the label to the date input and renders both inputs', () => {
    const { container } = render(
      <DateTimePicker id="startsAt" label="Starts" value={null} onChange={vi.fn()} />,
    )
    const { date, time } = getInputs(container)
    expect(screen.getByLabelText('Starts')).toBe(date)
    expect(date.id).toBe('startsAt')
    expect(time).toBeInTheDocument()
  })

  it('renders no label element when no label is given', () => {
    const { container } = render(<DateTimePicker value={null} onChange={vi.fn()} />)
    expect(container.querySelector('label')).toBeNull()
  })

  it('shows empty inputs and a disabled time input when value is null', () => {
    const { container } = render(<DateTimePicker value={null} onChange={vi.fn()} />)
    const { date, time } = getInputs(container)
    expect(date.value).toBe('')
    expect(time.value).toBe('')
    expect(time).toBeDisabled()
  })

  it('shows the UTC date and time of an ISO value', () => {
    const { container } = render(
      <DateTimePicker value="2026-05-01T09:30:00.000Z" onChange={vi.fn()} />,
    )
    const { date, time } = getInputs(container)
    expect(date.value).toBe('2026-05-01')
    expect(time.value).toBe('09:30')
    expect(time).toBeEnabled()
  })

  it('shows empty inputs for an unparseable value instead of throwing', () => {
    const { container } = render(<DateTimePicker value="not-a-date" onChange={vi.fn()} />)
    const { date, time } = getInputs(container)
    expect(date.value).toBe('')
    expect(time.value).toBe('')
    expect(time).toBeDisabled()
  })

  it('emits midnight UTC when a date is picked with no time set', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(<DateTimePicker value={null} onChange={onChange} />)
    const { date, time } = getInputs(container)

    await user.type(date, '2026-05-01')

    expect(onChange).toHaveBeenLastCalledWith('2026-05-01T00:00:00.000Z')
    expect(date.value).toBe('2026-05-01')
    expect(time).toBeEnabled()
  })

  it('keeps the existing time when only the date changes', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(
      <DateTimePicker value="2026-05-01T09:30:00.000Z" onChange={onChange} />,
    )
    const { date } = getInputs(container)

    await user.clear(date)
    await user.type(date, '2026-06-15')

    expect(onChange).toHaveBeenLastCalledWith('2026-06-15T09:30:00.000Z')
  })

  it('emits the combined ISO value when a time is entered for a set date', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(
      <DateTimePicker value="2026-05-01T00:00:00.000Z" onChange={onChange} />,
    )
    const { time } = getInputs(container)

    await user.clear(time)
    await user.type(time, '14:45')

    expect(onChange).toHaveBeenLastCalledWith('2026-05-01T14:45:00.000Z')
    expect(time.value).toBe('14:45')
  })

  it('falls back to 00:00 when the time is cleared but the date is kept', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(
      <DateTimePicker value="2026-05-01T09:30:00.000Z" onChange={onChange} />,
    )
    const { time } = getInputs(container)

    await user.clear(time)

    expect(onChange).toHaveBeenLastCalledWith('2026-05-01T00:00:00.000Z')
  })

  it('emits null and disables the time input when the date is cleared', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(
      <DateTimePicker value="2026-05-01T09:30:00.000Z" onChange={onChange} />,
    )
    const { date, time } = getInputs(container)

    await user.clear(date)

    expect(onChange).toHaveBeenLastCalledWith(null)
    expect(date.value).toBe('')
    expect(time).toBeDisabled()
  })

  it('keeps the time input disabled while no date is set, even when enabled', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(<DateTimePicker value={null} onChange={onChange} />)
    const { time } = getInputs(container)

    expect(time).toBeDisabled()
    await user.type(time, '10:00')

    expect(onChange).not.toHaveBeenCalled()
  })

  it('disables both inputs and ignores edits when disabled is set', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { container } = render(
      <DateTimePicker value="2026-05-01T09:30:00.000Z" onChange={onChange} disabled />,
    )
    const { date, time } = getInputs(container)

    expect(date).toBeDisabled()
    expect(time).toBeDisabled()

    await user.type(date, '2026-06-15')
    expect(onChange).not.toHaveBeenCalled()
  })

  it('resyncs the inputs when the value prop changes', () => {
    const { container, rerender } = render(
      <DateTimePicker value="2026-05-01T09:30:00.000Z" onChange={vi.fn()} />,
    )
    rerender(<DateTimePicker value="2027-01-20T18:05:00.000Z" onChange={vi.fn()} />)
    const { date, time } = getInputs(container)
    expect(date.value).toBe('2027-01-20')
    expect(time.value).toBe('18:05')

    rerender(<DateTimePicker value={null} onChange={vi.fn()} />)
    expect(date.value).toBe('')
    expect(time.value).toBe('')
    expect(time).toBeDisabled()
  })

  it('applies the className to the wrapper element', () => {
    const { container } = render(
      <DateTimePicker value={null} onChange={vi.fn()} className="custom-wrapper" />,
    )
    expect(container.firstElementChild).toHaveClass('custom-wrapper')
  })
})
