import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import i18n from '@/i18n'
import { VerifyCertificatePage } from './VerifyCertificatePage'

const CODE = '550e8400-e29b-41d4-a716-446655440000'

function json(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

let verifyHandler: () => Promise<Response>
const fetchMock = vi.fn()

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/verify/${CODE}`]}>
        <Routes>
          <Route path="/verify/:code" element={<VerifyCertificatePage />} />
          <Route path="/login" element={<div>login page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  fetchMock.mockReset()
  fetchMock.mockImplementation((url: string) => {
    if (url.startsWith('/api/v1/tenant/config')) {
      return json({
        data: {
          app_name: 'Acme Corp',
          primary_color: '#000',
          accent_color: '#111',
          default_locale: 'en',
          available_locales: ['en', 'ru', 'kk'],
        },
        error: null,
      })
    }
    return verifyHandler()
  })
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('VerifyCertificatePage', () => {
  it('shows a loading skeleton while the request is pending', () => {
    verifyHandler = () => new Promise(() => {})
    renderPage()
    expect(screen.getByTestId('verify-skeleton')).toBeInTheDocument()
  })

  it('renders the valid state with certificate details and no login redirect', async () => {
    verifyHandler = () =>
      json({
        data: {
          valid: true,
          employee_name: 'Aigerim S.',
          exam_title: 'Safety 101',
          score_pct: 84.5,
          issued_at: '2026-05-14T10:35:00Z',
        },
        error: null,
      })
    renderPage()
    const status = await screen.findByRole('status')
    expect(status).toHaveTextContent('Certificate is valid')
    expect(screen.getByText('Aigerim S.')).toBeInTheDocument()
    expect(screen.getByText('Safety 101')).toBeInTheDocument()
    expect(screen.getByText('84.5%')).toBeInTheDocument()
    expect(screen.getByText(/May 14, 2026/)).toBeInTheDocument()
    expect(screen.queryByText('login page')).not.toBeInTheDocument()
    expect(await screen.findByText('Acme Corp', { selector: 'p' })).toBeInTheDocument()
  })

  it('renders the invalid state without personal fields', async () => {
    verifyHandler = () => json({ data: { valid: false }, error: null })
    renderPage()
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Certificate not found or invalid')
    expect(screen.queryByText('Employee')).not.toBeInTheDocument()
  })

  it('shows unavailable state on HTTP 500 and retries', async () => {
    let calls = 0
    verifyHandler = () => {
      calls++
      return calls === 1
        ? json({ data: null, error: { code: 'INTERNAL', message: 'boom' } }, 500)
        : json({ data: { valid: false }, error: null })
    }
    renderPage()
    expect(await screen.findByText('Verification temporarily unavailable')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('Certificate not found or invalid'),
    )
  })

  it('shows unavailable state on network failure', async () => {
    verifyHandler = () => Promise.reject(new TypeError('network'))
    renderPage()
    expect(await screen.findByText('Verification temporarily unavailable')).toBeInTheDocument()
  })

  it('sets noindex meta while mounted, removes on unmount, and sends no Authorization header', async () => {
    verifyHandler = () => json({ data: { valid: false }, error: null })
    const { unmount } = renderPage()
    expect(document.querySelector('meta[name="robots"][content="noindex"]')).not.toBeNull()
    await screen.findByRole('alert')
    for (const call of fetchMock.mock.calls) {
      const init = call[1] as RequestInit | undefined
      expect(new Headers(init?.headers).has('Authorization')).toBe(false)
    }
    unmount()
    expect(document.querySelector('meta[name="robots"]')).toBeNull()
  })

  it('switches locale without reload', async () => {
    verifyHandler = () => json({ data: { valid: false }, error: null })
    renderPage()
    await screen.findByRole('alert')
    await i18n.changeLanguage('ru')
    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('Сертификат не найден или недействителен'),
    )
  })
})
