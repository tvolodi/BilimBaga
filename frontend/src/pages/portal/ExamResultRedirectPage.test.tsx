import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { createAppQueryClient } from '@/lib/passwordChangeRequired'
import { ExamResultRedirectPage } from './ExamResultRedirectPage'

// #470: an unknown exam answers 404 EXAM_NOT_FOUND. The app's QueryClient retry policy is used on purpose:
// a retried 404 kept the spinner up for about 7 s before the redirect.

describe('ExamResultRedirectPage for an unknown exam (#470)', () => {
  it('sends the user to the results list within a second, without retrying the 404', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      status: 404,
      json: async () => ({ data: null, error: { code: 'EXAM_NOT_FOUND', message: 'Exam not found.' } }),
    }))
    global.fetch = fetchMock as unknown as typeof fetch
    const qc = createAppQueryClient()
    qc.setQueryData(['auth', 'accessToken'], 'test-token')

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/portal/exams/missing-exam/result']}>
          <Routes>
            <Route path="/portal/exams/:examId/result" element={<ExamResultRedirectPage />} />
            <Route path="/portal/results" element={<div>My results</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(await screen.findByText('My results', undefined, { timeout: 1000 })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
