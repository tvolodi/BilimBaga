import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { AuditLogPage } from './AuditLogPage'
import { useAuditLog, exportAuditLog } from '@/api/audit'
import { waitFor } from '@testing-library/react'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('@/components/audit/AuditFilterBar', () => ({
  AuditFilterBar: () => null,
}))

vi.mock('@/components/audit/AuditLogTable', () => ({
  AuditLogTable: () => null,
}))

vi.mock('@/api/audit', () => ({
  useAuditLog: vi.fn(() => ({
    data: { items: [], meta: { page: 1, per_page: 50, total: 0 } },
    isLoading: false,
    isError: false,
  })),
  exportAuditLog: vi.fn(async () => {}),
}))

function renderPage(initialEntry: string = '/admin/audit') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })

  function Harness() {
    const [tick, setTick] = useState(0)
    return (
      <>
        <button type="button" onClick={() => setTick((v) => v + 1)}>
          rerender
        </button>
        <div data-testid="tick">{tick}</div>
        <AuditLogPage />
      </>
    )
  }

  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/admin/audit" element={<Harness />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AuditLogPage', () => {
  it('keeps default from filter stable across rerenders when URL has no from param (ISS-024)', () => {
    const useAuditLogMock = vi.mocked(useAuditLog)

    renderPage('/admin/audit')
    fireEvent.click(screen.getByRole('button', { name: 'rerender' }))

    expect(useAuditLogMock.mock.calls.length).toBeGreaterThanOrEqual(2)
    const firstFilters = useAuditLogMock.mock.calls[0][0]
    const secondFilters = useAuditLogMock.mock.calls[1][0]

    expect(firstFilters.from).toBeTruthy()
    expect(secondFilters.from).toBeTruthy()
    expect(secondFilters.from).toBe(firstFilters.from)
  })

  it('shows a translated error when the export fails (401 after refresh)', async () => {
    vi.mocked(exportAuditLog).mockRejectedValueOnce(
      Object.assign(new Error('ERR_UNAUTHORIZED'), { code: 'ERR_UNAUTHORIZED' }),
    )
    renderPage('/admin/audit')
    fireEvent.click(screen.getByRole('button', { name: /audit\.export_csv/ }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('download.session_expired')
    await waitFor(() => expect(screen.getByRole('button', { name: /audit\.export_csv/ })).not.toBeDisabled())
  })

  it('shows a generic error for other export failures and clears it on retry', async () => {
    vi.mocked(exportAuditLog).mockRejectedValueOnce(new Error('boom'))
    renderPage('/admin/audit')
    fireEvent.click(screen.getByRole('button', { name: /audit\.export_csv/ }))
    expect(await screen.findByRole('alert')).toHaveTextContent('download.failed')
    fireEvent.click(screen.getByRole('button', { name: /audit\.export_csv/ }))
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })
})
