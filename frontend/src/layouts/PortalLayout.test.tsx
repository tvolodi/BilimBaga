import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { PortalLayout } from './PortalLayout'

// Mock useLogout from auth API
const mockMutateAsync = vi.fn().mockResolvedValue(undefined)
vi.mock('@/api/auth', () => ({
  useLogout: () => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
}))

function renderLayout(initialPath = '/portal') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/portal/*" element={<PortalLayout />} />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('PortalLayout', () => {
  it('renders the portal navigation tabs', () => {
    renderLayout()
    expect(screen.getByRole('navigation', { name: /portal navigation/i })).toBeInTheDocument()
  })

  it('renders a Sign Out button with correct aria-label', () => {
    renderLayout()
    expect(screen.getByRole('button', { name: /sign out/i })).toBeInTheDocument()
  })

  it('calls logout and navigates to /login when Sign Out is clicked', async () => {
    renderLayout()
    const signOutBtn = screen.getByRole('button', { name: /sign out/i })
    fireEvent.click(signOutBtn)
    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledTimes(1)
    })
    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument()
    })
  })
})
