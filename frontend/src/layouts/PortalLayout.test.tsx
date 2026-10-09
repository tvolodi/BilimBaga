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

describe('PortalLayout narrow-viewport layout (ISS-130)', () => {
  it('lets the nav wrap and the tabs shrink so nothing overflows at 375px', () => {
    renderLayout()
    const nav = screen.getByRole('navigation', { name: /portal navigation/i })
    expect(nav).toHaveClass('flex-wrap')
    const tabGroup = nav.firstElementChild as HTMLElement
    expect(tabGroup).toHaveClass('min-w-0', 'max-w-full')
    const link = screen.getAllByRole('link')[0]
    expect(link).toHaveClass('px-2', 'sm:px-4')
    const cluster = nav.lastElementChild as HTMLElement
    expect(cluster).toHaveClass('shrink-0')
    expect(screen.getByRole('combobox')).toHaveClass('w-[104px]', 'sm:w-[120px]')
  })
})
