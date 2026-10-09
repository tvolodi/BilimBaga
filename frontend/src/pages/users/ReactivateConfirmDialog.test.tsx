import { describe, it, expect, vi, beforeAll, afterAll, afterEach, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { ReactivateConfirmDialog } from './ReactivateConfirmDialog'
import type { User } from '@/api/users'

// FR-BB18 AC-13: the confirmation dialog for reactivating a deactivated user.
const deactivatedUser = {
  id: 'u-7',
  full_name: 'Aliya Sarsen',
  email: 'aliya@example.com',
  status: 'inactive',
} as unknown as User

const reactivated: string[] = []

const server = setupServer(
  http.post('/api/v1/users/:id/reactivate', ({ params }) => {
    reactivated.push(params.id as string)
    return HttpResponse.json({ data: {}, error: null })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  reactivated.length = 0
})
afterAll(() => server.close())

beforeEach(() => {
  vi.clearAllMocks()
})

function renderDialog(user: User | null, onClose = vi.fn()) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  render(<ReactivateConfirmDialog user={user} onClose={onClose} />, { wrapper: Wrapper })
  return { onClose }
}

describe('ReactivateConfirmDialog', () => {
  it('names the user in the confirmation message', () => {
    renderDialog(deactivatedUser)
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByText('Reactivate User')).toBeInTheDocument()
    expect(within(dialog).getByText(/Aliya Sarsen/)).toBeInTheDocument()
  })

  it('reactivates the user and closes when confirmed', async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog(deactivatedUser)

    await user.click(screen.getByRole('button', { name: 'Reactivate' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(reactivated).toEqual(['u-7'])
  })

  it('closes without sending a request when cancelled', async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog(deactivatedUser)

    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onClose).toHaveBeenCalledTimes(1)
    expect(reactivated).toEqual([])
  })

  it('shows the error and stays open when the request fails', async () => {
    server.use(
      http.post('/api/v1/users/:id/reactivate', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INTERNAL', message: 'reactivation failed on the server' } },
          { status: 500 },
        ),
      ),
    )
    const user = userEvent.setup()
    const { onClose } = renderDialog(deactivatedUser)

    await user.click(screen.getByRole('button', { name: 'Reactivate' }))

    expect(await screen.findByText('reactivation failed on the server')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('renders no dialog when there is no user', () => {
    renderDialog(null)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
