import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { ForgotPasswordPage } from './ForgotPasswordPage'
import { ResetPasswordPage } from './ResetPasswordPage'

interface Captured {
  url: string
  auth: string | null
  credentials: RequestCredentials
  body: unknown
}
let captured: Captured[] = []

function record(request: Request, body: unknown) {
  captured.push({
    url: new URL(request.url).pathname,
    auth: request.headers.get('Authorization'),
    credentials: request.credentials,
    body,
  })
}

const server = setupServer()
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  captured = []
})
afterAll(() => server.close())

function LoginProbe() {
  const loc = useLocation()
  const state = loc.state as { passwordReset?: boolean } | null
  return <div data-testid="login">login reset={String(state?.passwordReset === true)}</div>
}

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/login" element={<LoginProbe />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const GENERIC = /if an account exists for that email/i

describe('ForgotPasswordPage (FR-BB115 AC-7)', () => {
  it('posts the email unauthenticated and shows the neutral confirmation', async () => {
    server.use(
      http.post('/api/v1/auth/forgot-password', async ({ request }) => {
        record(request, await request.json())
        return HttpResponse.json({ data: { message: 'x' }, error: null })
      }),
    )
    const user = userEvent.setup()
    renderAt('/forgot-password')

    await user.type(screen.getByLabelText(/email/i), 'alice@example.com')
    await user.click(screen.getByRole('button', { name: /send reset link/i }))

    expect(await screen.findByRole('status')).toHaveTextContent(GENERIC)
    expect(captured).toHaveLength(1)
    expect(captured[0].body).toEqual({ email: 'alice@example.com' })
    expect(captured[0].auth).toBeNull()
    expect(captured[0].credentials).not.toBe('include')
  })

  it('shows the same confirmation for an unknown address (server answers identically)', async () => {
    server.use(
      http.post('/api/v1/auth/forgot-password', () =>
        HttpResponse.json({ data: { message: 'same' }, error: null }),
      ),
    )
    const user = userEvent.setup()
    renderAt('/forgot-password')
    await user.type(screen.getByLabelText(/email/i), 'nobody@example.com')
    await user.click(screen.getByRole('button', { name: /send reset link/i }))
    expect(await screen.findByRole('status')).toHaveTextContent(GENERIC)
  })

  it('stays neutral when the request is throttled (429)', async () => {
    server.use(
      http.post('/api/v1/auth/forgot-password', () =>
        HttpResponse.json(
          { data: null, error: { code: 'RATE_LIMITED', message: 'Too many requests' } },
          { status: 429 },
        ),
      ),
    )
    const user = userEvent.setup()
    renderAt('/forgot-password')
    await user.type(screen.getByLabelText(/email/i), 'alice@example.com')
    await user.click(screen.getByRole('button', { name: /send reset link/i }))
    expect(await screen.findByRole('status')).toHaveTextContent(GENERIC)
  })

  it('shows a generic error when the service is unreachable', async () => {
    server.use(http.post('/api/v1/auth/forgot-password', () => HttpResponse.error()))
    const user = userEvent.setup()
    renderAt('/forgot-password')
    await user.type(screen.getByLabelText(/email/i), 'alice@example.com')
    await user.click(screen.getByRole('button', { name: /send reset link/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/service unavailable/i)
  })

  it('links back to the sign-in page', () => {
    renderAt('/forgot-password')
    expect(screen.getByRole('link', { name: /back to sign in/i })).toHaveAttribute('href', '/login')
  })
})

describe('ResetPasswordPage (FR-BB115 AC-7)', () => {
  async function fill(user: ReturnType<typeof userEvent.setup>, pw: string, confirm: string) {
    await user.type(screen.getByLabelText(/^new password/i), pw)
    await user.type(screen.getByLabelText(/confirm new password/i), confirm)
    await user.click(screen.getByRole('button', { name: /^reset password$/i }))
  }

  it('sends token + password unauthenticated and redirects to /login with a success flag', async () => {
    server.use(
      http.post('/api/v1/auth/reset-password', async ({ request }) => {
        record(request, await request.json())
        return HttpResponse.json({ data: { message: 'ok' }, error: null })
      }),
    )
    const user = userEvent.setup()
    renderAt('/reset-password?token=abc_DEF-123')

    await fill(user, 'Str0ngPass', 'Str0ngPass')

    expect(await screen.findByTestId('login')).toHaveTextContent('reset=true')
    expect(captured[0].body).toEqual({ token: 'abc_DEF-123', new_password: 'Str0ngPass' })
    expect(captured[0].auth).toBeNull()
  })

  it('blocks submit and shows an error when passwords do not match', async () => {
    server.use(
      http.post('/api/v1/auth/reset-password', async ({ request }) => {
        record(request, await request.json())
        return HttpResponse.json({ data: { message: 'ok' }, error: null })
      }),
    )
    const user = userEvent.setup()
    renderAt('/reset-password?token=t')

    await fill(user, 'Str0ngPass', 'Different1')

    expect(await screen.findByRole('alert')).toHaveTextContent(/do not match/i)
    expect(captured).toHaveLength(0)
  })

  it('shows the policy error and keeps the form for a weak password (token not consumed)', async () => {
    server.use(
      http.post('/api/v1/auth/reset-password', () =>
        HttpResponse.json(
          { data: null, error: { code: 'VALIDATION_ERROR', message: 'weak' } },
          { status: 400 },
        ),
      ),
    )
    const user = userEvent.setup()
    renderAt('/reset-password?token=t')

    await fill(user, 'weak', 'weak')

    expect(await screen.findByRole('alert')).toHaveTextContent(/does not meet requirements/i)
    expect(screen.getByLabelText(/^new password/i)).toBeInTheDocument()
    expect(screen.queryByTestId('login')).not.toBeInTheDocument()
  })

  it('shows the invalid-link error with a link to /forgot-password for INVALID_TOKEN', async () => {
    server.use(
      http.post('/api/v1/auth/reset-password', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INVALID_TOKEN', message: 'nope' } },
          { status: 400 },
        ),
      ),
    )
    const user = userEvent.setup()
    renderAt('/reset-password?token=expired')

    await fill(user, 'Str0ngPass', 'Str0ngPass')

    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(/invalid or has expired/i))
    expect(screen.getByRole('link', { name: /request a new reset link/i })).toHaveAttribute(
      'href',
      '/forgot-password',
    )
    expect(screen.queryByLabelText(/^new password/i)).not.toBeInTheDocument()
  })

  it('shows the invalid-link state immediately when the token is missing, without any request', () => {
    renderAt('/reset-password')
    expect(screen.getByRole('alert')).toHaveTextContent(/invalid or has expired/i)
    expect(screen.getByRole('link', { name: /request a new reset link/i })).toHaveAttribute(
      'href',
      '/forgot-password',
    )
    expect(captured).toHaveLength(0)
  })

  it('toggles password visibility', async () => {
    const user = userEvent.setup()
    renderAt('/reset-password?token=t')
    const field = screen.getByLabelText(/^new password/i)
    expect(field).toHaveAttribute('type', 'password')
    await user.click(screen.getAllByRole('button', { name: /show password/i })[0])
    expect(field).toHaveAttribute('type', 'text')
  })

  it('sets <meta name="referrer" content="no-referrer"> while mounted and removes it on unmount (ISS-105)', () => {
    const { unmount } = renderAt('/reset-password?token=abc')
    expect(document.querySelector('meta[name="referrer"][content="no-referrer"]')).not.toBeNull()
    unmount()
    expect(document.querySelector('meta[name="referrer"]')).toBeNull()
  })
})
