import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import App from './App'

// FR-BB115 AC-7: the recovery pages are public — outside RequireAuth, no refresh bootstrap,
// no admin/portal navigation.
const calls: string[] = []

const server = setupServer(
  http.get('/api/v1/tenant/config', () =>
    HttpResponse.json({
      data: { app_name: 'BilimBaga', primary_color: '#000', accent_color: '#111', available_locales: ['en'] },
      error: null,
    }),
  ),
  http.post('/api/v1/auth/refresh', () => {
    calls.push('refresh')
    return HttpResponse.json({ data: null, error: { code: 'INVALID_REFRESH_TOKEN', message: 'x' } }, { status: 401 })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  calls.length = 0
})
afterAll(() => server.close())

describe('App public recovery routes', () => {
  it('renders /forgot-password without attempting a token refresh', async () => {
    window.history.pushState({}, '', '/forgot-password')
    render(<App />)
    expect(await screen.findByRole('heading', { name: /forgot your password/i })).toBeInTheDocument()
    expect(calls).toEqual([])
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })

  it('renders /reset-password without attempting a token refresh', async () => {
    window.history.pushState({}, '', '/reset-password?token=abc')
    render(<App />)
    expect(await screen.findByRole('heading', { name: /choose a new password/i })).toBeInTheDocument()
    expect(calls).toEqual([])
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })
})
