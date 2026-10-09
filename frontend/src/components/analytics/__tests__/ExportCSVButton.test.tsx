import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/i18n'
import { ExportCSVButton } from '../ExportCSVButton'

function blobResponse(status = 200): Response {
  return new Response('csv,data', { status })
}
function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function setup(examId = 'exam-123', token: string | null = 'tok-1') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], token)
  const { container } = render(
    <QueryClientProvider client={qc}>
      <ExportCSVButton examId={examId} />
    </QueryClientProvider>,
  )
  return { container, qc }
}

describe('ExportCSVButton', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  let clicked: HTMLAnchorElement[]

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    URL.revokeObjectURL = vi.fn()
    clicked = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      clicked.push(this)
    })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('renders an enabled export button', () => {
    const { container } = setup()
    expect(container.querySelector('button')).not.toBeDisabled()
  })

  it('downloads with Bearer token via the shared helper', async () => {
    fetchMock.mockResolvedValueOnce(blobResponse())
    const { container } = setup('exam-abc')
    fireEvent.click(container.querySelector('button')!)
    await waitFor(() => expect(clicked).toHaveLength(1))
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/admin/exams/exam-abc/results/export')
    expect(init.headers.Authorization).toBe('Bearer tok-1')
    expect(clicked[0].download).toBe('exam-exam-abc-results.csv')
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('disables the button while downloading', async () => {
    let resolveFetch!: (r: Response) => void
    fetchMock.mockReturnValueOnce(new Promise<Response>((r) => (resolveFetch = r)))
    const { container } = setup()
    const btn = container.querySelector('button')!
    fireEvent.click(btn)
    await waitFor(() => expect(btn).toBeDisabled())
    resolveFetch(blobResponse())
    await waitFor(() => expect(btn).not.toBeDisabled())
  })

  it('refreshes the token on 401 and retries', async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ error: { code: 'ERR_UNAUTHORIZED' } }, 401))
      .mockResolvedValueOnce(jsonResponse({ data: { access_token: 'new' }, error: null }, 200))
      .mockResolvedValueOnce(blobResponse())
    const { container, qc } = setup('e1', 'old')
    fireEvent.click(container.querySelector('button')!)
    await waitFor(() => expect(clicked).toHaveLength(1))
    expect(fetchMock.mock.calls[1][0]).toBe('/api/v1/auth/refresh')
    expect(fetchMock.mock.calls[2][1].headers.Authorization).toBe('Bearer new')
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe('new')
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('shows a session-expired error when refresh fails', async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({}, 401))
      .mockResolvedValueOnce(jsonResponse({}, 401))
    const { container } = setup()
    fireEvent.click(container.querySelector('button')!)
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Your session has expired')
    expect(clicked).toHaveLength(0)
    expect(container.querySelector('button')).not.toBeDisabled()
  })

  it('shows a generic error on server failure and clears it on retry', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: { code: 'ERR_FORBIDDEN' } }, 403))
    const { container } = setup()
    fireEvent.click(container.querySelector('button')!)
    expect(await screen.findByRole('alert')).toHaveTextContent('Download failed')
    fetchMock.mockResolvedValueOnce(blobResponse())
    fireEvent.click(container.querySelector('button')!)
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })
})
