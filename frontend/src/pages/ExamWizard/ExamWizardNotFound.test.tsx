import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { createAppQueryClient } from '@/lib/passwordChangeRequired'
import { ExamWizardEditPage } from './index'

// #470: editing an unknown exam shows the not-found state, not a wizard with no data. The real hook and
// the app's QueryClient retry policy are used, so a retried 404 would be too slow for the one-second check.

function makeToken(role: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ role, sub: 'user-1' }))
  return `${header}.${payload}.fake-sig`
}

function response(status: number, body: unknown) {
  return { ok: status < 400, status, json: async () => body }
}

const fetchMock = vi.fn(async (url: RequestInfo | URL) =>
  String(url).endsWith('/api/v1/exams/missing-exam')
    ? response(404, { data: null, error: { code: 'EXAM_NOT_FOUND', message: 'exam not found' } })
    : response(200, { data: null, error: null }),
)

function examCalls(): number {
  return fetchMock.mock.calls.filter(([url]) => String(url).endsWith('/api/v1/exams/missing-exam')).length
}

afterEach(() => {
  fetchMock.mockClear()
})

describe('ExamWizard edit mode for an unknown exam (#470)', () => {
  it('shows the not-found message and a way back to the exam list within a second', async () => {
    const qc = createAppQueryClient()
    qc.setQueryData(['auth', 'accessToken'], makeToken('super_admin'))
    global.fetch = fetchMock as unknown as typeof fetch
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/admin/exams/missing-exam/edit']}>
          <Routes>
            <Route path="/admin/exams/:id/edit" element={<ExamWizardEditPage />} />
            <Route path="/admin/exams" element={<div>Exam list</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(await screen.findByText('Exam not found. It may have been deleted.', undefined, { timeout: 1000 })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Exams' })).toHaveAttribute('href', '/admin/exams')
    expect(examCalls()).toBe(1)
  })
})
