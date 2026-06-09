import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { RequireAuth } from './RequireAuth'

/**
 * Seed the query cache to a specific state.
 * status: 'pending' — query exists but has never resolved (loading).
 * status: 'success' with data — query resolved with a token.
 * status: 'success' with null — query resolved but no token (unauthenticated).
 */
function makeQC(state: 'pending' | { token: string | null }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  if (state === 'pending') {
    // Inject a manual query state with status=pending so getQueryState returns it.
    qc.setQueryDefaults(['auth', 'accessToken'], { queryFn: () => new Promise(() => {}) })
    // Force-set internal state to pending without running the queryFn.
    // The simplest way: add a cache entry with fetchStatus=fetching but no data yet.
    // We use setQueryData + then manually mark it via the queryCache.
    // Easiest reliable approach: just don't set any data — getQueryState returns
    // undefined (no entry), which RequireAuth also treats as pending.
    // We leave the cache empty to simulate "query not yet settled".
    // (RequireAuth treats !queryState as pending and shows spinner.)
  } else {
    qc.setQueryData(['auth', 'accessToken'], state.token)
  }

  return qc
}

function renderWithQC(qc: QueryClient) {
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <RequireAuth>
                <div>Protected Content</div>
              </RequireAuth>
            }
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('RequireAuth', () => {
  it('shows a spinner when the auth query has not settled yet (ISS-029)', () => {
    // No cache entry at all — simulates the race condition where useRefreshToken
    // hasn't settled after a fresh login redirect.
    const qc = makeQC('pending')
    renderWithQC(qc)
    // The spinner renders a spinning div — no redirect to /login.
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
    // The spinner container is present.
    expect(document.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('redirects to /login when the auth query settled with no token', () => {
    const qc = makeQC({ token: null })
    renderWithQC(qc)
    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })

  it('renders children when the auth query settled with a valid token', () => {
    const qc = makeQC({ token: 'some.valid.token' })
    renderWithQC(qc)
    expect(screen.getByText('Protected Content')).toBeInTheDocument()
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
  })
})
