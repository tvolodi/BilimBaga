import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { createAppQueryClient } from '@/lib/passwordChangeRequired'
import { ExamWizardEditPage } from './index'

// #487: the real answer for an unknown exam is 404 with the code ERR_NOT_FOUND (the wizard test with EXAM_NOT_FOUND
// passed while the page showed an empty form, because it used the shape the check expects). This test uses the real code.

function makeToken(role: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ role, sub: 'user-1' }))
  return `${header}.${payload}.fake-sig`
}

const fetchMock = vi.fn(async (_url: RequestInfo | URL) => ({
  ok: false,
  status: 404,
  json: async () => ({ data: null, error: { code: 'ERR_NOT_FOUND', message: 'exam not found' } }),
}))

afterEach(() => {
  fetchMock.mockClear()
  vi.unstubAllGlobals()
})

describe('ExamWizard edit mode for an unknown exam, real ERR_NOT_FOUND answer (#487)', () => {
  it('shows the not-found state within a second and does not retry the 404', async () => {
    const qc = createAppQueryClient()
    qc.setQueryData(['auth', 'accessToken'], makeToken('super_admin'))
    vi.stubGlobal('fetch', fetchMock)
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
    const examCalls = fetchMock.mock.calls.filter(([url]) => String(url).endsWith('/api/v1/exams/missing-exam')).length
    expect(examCalls).toBe(1)
  })
})
