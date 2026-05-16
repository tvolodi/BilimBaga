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

  it('shows success toast after reminder API resolves', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: null, error: null }),
    })
    setup([employee])
    fireEvent.click(screen.getByRole('button', { name: /send reminder/i }))
    await waitFor(() => {
      expect(screen.getByText('Reminder sent successfully.')).toBeInTheDocument()
    })
  })

  it('shows error toast when reminder API fails', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ data: null, error: { code: 'ERR', message: 'fail' } }),
    })
    setup([employee])
    fireEvent.click(screen.getByRole('button', { name: /send reminder/i }))
    await waitFor(() => {
      expect(screen.getByText('Failed to send reminder.')).toBeInTheDocument()
    })
  })
})
