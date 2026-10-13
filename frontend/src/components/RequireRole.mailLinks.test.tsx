import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { RequireRole } from './RequireRole'

function LoginProbe() {
  const loc = useLocation()
  const from = (loc.state as { from?: string } | null)?.from
  return <div data-testid="login-from">{from ?? 'none'}</div>
}

// FR-BB68 AC-5: a mail link opened while signed out goes to /login and carries the target path to return to.
function renderSignedOutAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], null)
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route
            path="/portal"
            element={
              <RequireRole roles={['employee']}>
                <div>Portal</div>
              </RequireRole>
            }
          />
          <Route
            path="/portal/sessions/:sessionId/result"
            element={
              <RequireRole roles={['employee']}>
                <div>Session Result</div>
              </RequireRole>
            }
          />
          <Route path="/login" element={<LoginProbe />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('RequireRole mail link targets (FR-BB68 AC-5)', () => {
  it('sends a signed-out visitor of /portal to /login with from=/portal', () => {
    renderSignedOutAt('/portal')
    expect(screen.getByTestId('login-from')).toHaveTextContent('/portal')
    expect(screen.queryByText('Portal')).not.toBeInTheDocument()
  })

  it('sends a signed-out visitor of the session result route to /login with that path', () => {
    renderSignedOutAt('/portal/sessions/sess-1/result')
    expect(screen.getByTestId('login-from')).toHaveTextContent('/portal/sessions/sess-1/result')
    expect(screen.queryByText('Session Result')).not.toBeInTheDocument()
  })
})
