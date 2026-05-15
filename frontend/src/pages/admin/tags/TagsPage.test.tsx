import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { TagsPage } from './TagsPage'

const meAdmin = {
  id: 'u-1',
  email: 'admin@example.com',
  full_name: 'Admin',
  department_id: null,
  department_name: null,
  role_id: 'role-sa',
  role_name: 'super_admin',
  status: 'active',
  force_password_change: false,
  created_at: '2024-01-01T00:00:00Z',
}

const sampleTags = [
  { id: 'tag-1', name: 'Biology', created_at: '2024-01-01T00:00:00Z', usage_count: 5 },
  { id: 'tag-2', name: 'Chemistry', created_at: '2024-01-01T00:00:00Z', usage_count: 3 },
]

const server = setupServer(
  http.get('/api/v1/users/me', () => HttpResponse.json({ data: meAdmin, error: null })),
  http.get('/api/v1/tags', () => HttpResponse.json({ data: sampleTags, error: null })),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('TagsPage', () => {
  it('renders tags loaded from the API', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Biology')).toBeInTheDocument()
      expect(screen.getByText('Chemistry')).toBeInTheDocument()
    })
  })

  it('shows the usage count for each tag', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))
    // Usage counts should be visible
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('filters tags by search term', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    const searchInput = screen.getByRole('textbox')
    await userEvent.type(searchInput, 'Bio')

    await waitFor(() => {
      expect(screen.getByText('Biology')).toBeInTheDocument()
      expect(screen.queryByText('Chemistry')).not.toBeInTheDocument()
    })
  })

  it('shows empty state when no tags match search', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    const searchInput = screen.getByRole('textbox')
    await userEvent.type(searchInput, 'XYZ_NO_MATCH')

    await waitFor(() => {
      expect(screen.queryByText('Biology')).not.toBeInTheDocument()
      expect(screen.queryByText('Chemistry')).not.toBeInTheDocument()
    })
  })

  it('shows New Tag button for super_admin', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))
    expect(screen.getByRole('button', { name: /new tag/i })).toBeInTheDocument()
  })
})
