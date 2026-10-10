import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import i18n from '@/i18n'
import { createAppQueryClient } from '@/lib/passwordChangeRequired'
import { ExamAnalyticsPage } from './ExamAnalyticsPage'

// #470: an unknown exam answers 404 EXAM_NOT_FOUND. The real hook and the app's QueryClient retry
// policy are used on purpose: the defect is that a 404 was retried for about 7 s behind a skeleton.

function makeToken(role: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ role, sub: 'user-1' }))
  return `${header}.${payload}.fake-sig`
}

function response(status: number, body: unknown) {
  return { ok: status < 400, status, json: async () => body }
}

const fetchMock = vi.fn(async (url: RequestInfo | URL) =>
  String(url).includes('/analytics')
    ? response(404, { data: null, error: { code: 'EXAM_NOT_FOUND', message: 'exam not found' } })
    : response(200, { data: null, error: null }),
)

function analyticsCalls(): number {
  return fetchMock.mock.calls.filter(([url]) => String(url).includes('/analytics')).length
}

function renderPage(examId = 'missing-exam') {
  const qc = createAppQueryClient()
  qc.setQueryData(['auth', 'accessToken'], makeToken('super_admin'))
  global.fetch = fetchMock as unknown as typeof fetch
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/admin/exams/${examId}/analytics`]}>
        <Routes>
          <Route path="/admin/exams/:examId/analytics" element={<ExamAnalyticsPage />} />
          <Route path="/admin/exams" element={<div>Exam list</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

afterEach(async () => {
  fetchMock.mockClear()
  await i18n.changeLanguage('en')
})

describe('ExamAnalyticsPage for an unknown exam (#470)', () => {
  it.each([
    ['en', 'Exam not found. It may have been deleted.'],
    ['kk', 'Емтихан табылмады. Ол жойылған болуы мүмкін.'],
    ['ru', 'Экзамен не найден. Возможно, его удалили.'],
  ])('shows the translated not-found message in %s within a second', async (lng, message) => {
    await i18n.changeLanguage(lng)
    renderPage()

    expect(await screen.findByText(message, undefined, { timeout: 1000 })).toBeInTheDocument()
  })

  it('does not retry the 404', async () => {
    renderPage()

    await screen.findByText('Exam not found. It may have been deleted.', undefined, { timeout: 1000 })
    expect(analyticsCalls()).toBe(1)
  })

  it('offers a way back to the exam list', async () => {
    renderPage()

    await screen.findByText('Exam not found. It may have been deleted.', undefined, { timeout: 1000 })
    const backLinks = screen.getAllByRole('link', { name: 'Exams' })
    expect(backLinks.some((link) => link.getAttribute('href') === '/admin/exams')).toBe(true)
  })
})
