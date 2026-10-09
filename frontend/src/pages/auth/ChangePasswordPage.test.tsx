import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { ChangePasswordPage } from './ChangePasswordPage'

// ISS-171: change-password revokes every earlier access token and returns the caller's
// replacement; the page must switch to it and continue (forced-change flow included).

function jwt(role: string): string {
  const b64 = (o: object) => btoa(JSON.stringify(o)).replace(/=+$/, '')
  return `${b64({ alg: 'none' })}.${b64({ sub: 'u1', role, exp: 9999999999 })}.sig`
}

const server = setupServer()
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function setup(qc: QueryClient) {
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/change-password']}>
        <Routes>
          <Route path="/change-password" element={<ChangePasswordPage />} />
          <Route path="/portal" element={<div>portal home</div>} />
          <Route path="/admin" element={<div>admin home</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function submit() {
  fireEvent.change(screen.getByLabelText('Current password'), { target: { value: 'OldPass123' } })
  fireEvent.change(screen.getByLabelText('New password'), { target: { value: 'NewPass456' } })
  fireEvent.change(screen.getByLabelText('Confirm new password'), { target: { value: 'NewPass456' } })
  fireEvent.click(screen.getByRole('button', { name: /change password/i }))
}

describe('ChangePasswordPage (ISS-171)', () => {
  it('stores the returned token, clears the forced flag and continues to the role home', async () => {
    const oldTok = jwt('employee')
    const newTok = jwt('employee')
    let sentAuth: string | null = null
    server.use(
      http.post('/api/v1/auth/change-password', ({ request }) => {
        sentAuth = request.headers.get('Authorization')
        return HttpResponse.json({
          data: { message: 'password changed', access_token: newTok, token_type: 'Bearer', expires_in: 900 },
          error: null,
        })
      }),
    )
    const qc = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], oldTok)
    qc.setQueryData(['auth', 'currentUser'], {
      id: 'u1', full_name: 'E', email: 'e@x', role: 'employee', force_password_change: true,
    })
    setup(qc)
    submit()

    await waitFor(() => expect(screen.getByText('portal home')).toBeInTheDocument())
    expect(sentAuth).toBe(`Bearer ${oldTok}`)
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe(newTok)
    expect(qc.getQueryData<{ force_password_change: boolean }>(['auth', 'currentUser'])?.force_password_change).toBe(false)
  })

  it('keeps the old token and stays on the page when the change is rejected', async () => {
    const oldTok = jwt('super_admin')
    server.use(
      http.post('/api/v1/auth/change-password', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INVALID_CREDENTIALS', message: 'current password is incorrect' } },
          { status: 400 },
        ),
      ),
    )
    const qc = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], oldTok)
    setup(qc)
    submit()

    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe(oldTok)
    expect(screen.queryByText('admin home')).not.toBeInTheDocument()
  })
})
