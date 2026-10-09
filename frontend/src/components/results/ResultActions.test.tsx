import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/i18n'
import { ResultActions } from './ResultActions'

function renderActions() {
  const qc = new QueryClient()
  qc.setQueryData(['auth', 'accessToken'], 'tok-9')
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <ResultActions sessionId="sess-1" passed certificateEnabled canRetake={false} examId="e1" />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('ResultActions certificate download', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    URL.createObjectURL = vi.fn(() => 'blob:mock')
    URL.revokeObjectURL = vi.fn()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('sends the Bearer token to the portal certificate endpoint', async () => {
    fetchMock.mockResolvedValueOnce(new Response(new Blob(['pdf']), { status: 200 }))
    renderActions()
    fireEvent.click(screen.getByRole('button', { name: /certificate/i }))
    await waitFor(() => expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock'))
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/portal/sessions/sess-1/certificate')
    expect(init.headers.Authorization).toBe('Bearer tok-9')
  })

  it('shows an error message when the download fails', async () => {
    fetchMock.mockResolvedValueOnce(new Response('{}', { status: 500 }))
    renderActions()
    fireEvent.click(screen.getByRole('button', { name: /certificate/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Download failed')
  })
})
