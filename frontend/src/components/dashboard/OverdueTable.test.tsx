import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/i18n'
import { OverdueTable } from './OverdueTable'
import type { OverdueEmployee } from '@/api/dashboard'

const employee: OverdueEmployee = {
  user_id: 'u-1',
  name: 'Alice Smith',
  exam_title: 'Safety Exam',
  exam_id: 'e-1',
  deadline: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
}

function setup(employees: OverdueEmployee[]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <OverdueTable employees={employees} />
    </QueryClientProvider>,
  )
}

describe('OverdueTable', () => {
  it('shows empty state message when no employees', () => {
    setup([])
    expect(screen.getByText('No overdue assignments.')).toBeInTheDocument()
  })

  it('renders employee name, exam name, and deadline', () => {
    setup([employee])
    expect(screen.getByText('Alice Smith')).toBeInTheDocument()
    expect(screen.getByText('Safety Exam')).toBeInTheDocument()
    // deadline is relative — just check a "Send Reminder" button exists
    expect(screen.getByRole('button', { name: /send reminder/i })).toBeInTheDocument()
  })

  it('renders Send Reminder button per row', () => {
    const second: OverdueEmployee = {
      ...employee,
      user_id: 'u-2',
      name: 'Bob Jones',
      exam_id: 'e-2',
    }
    setup([employee, second])
    expect(screen.getAllByRole('button', { name: /send reminder/i })).toHaveLength(2)
  })

  it('posts the real exam_id and shows success, then keeps the row disabled', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { sent_at: '2026-01-01T00:00:00Z' }, error: null }),
    })
    global.fetch = fetchMock
    setup([employee])
    fireEvent.click(screen.getByRole('button', { name: /send reminder/i }))
    await waitFor(() => {
      expect(screen.getByText('Reminder sent successfully.')).toBeInTheDocument()
    })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/admin/users/u-1/remind')
    expect(JSON.parse(init.body)).toEqual({ exam_id: 'e-1' })
    const sent = screen.getByRole('button', { name: 'Reminder sent' })
    expect(sent).toBeDisabled()
  })

  it('disables the button while the request is pending', async () => {
    let resolve!: (v: unknown) => void
    global.fetch = vi.fn().mockReturnValue(new Promise((r) => (resolve = r)))
    setup([employee])
    const btn = screen.getByRole('button', { name: /send reminder/i })
    fireEvent.click(btn)
    await waitFor(() => expect(btn).toBeDisabled())
    resolve({ ok: true, json: async () => ({ data: { sent_at: 'x' }, error: null }) })
    await waitFor(() => expect(screen.getByText('Reminder sent successfully.')).toBeInTheDocument())
  })

  it.each([
    ['REMINDER_RATE_LIMITED', 'Too many reminders: this employee was already reminded in the last 24 hours.'],
    ['NOT_OVERDUE', 'This employee has no pending overdue assignment for this exam.'],
    ['USER_INACTIVE', 'This employee has no pending overdue assignment for this exam.'],
    ['EMAIL_SEND_FAILED', 'Failed to send reminder.'],
    ['ERR', 'Failed to send reminder.'],
  ])('maps error code %s to its message', async (code, message) => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ data: null, error: { code, message: 'fail' } }),
    })
    setup([employee])
    fireEvent.click(screen.getByRole('button', { name: /send reminder/i }))
    await waitFor(() => {
      expect(screen.getByText(message)).toBeInTheDocument()
    })
    // failure leaves the button usable
    expect(screen.getByRole('button', { name: /send reminder/i })).toBeEnabled()
  })
})
