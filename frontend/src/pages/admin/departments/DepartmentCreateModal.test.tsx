import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { DepartmentCreateModal } from './DepartmentCreateModal'
import { type Department } from '@/api/departments'

const sampleDepts: Department[] = [
  {
    id: 'dept-1',
    name: 'Engineering',
    parent_id: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    children: [],
  },
]

const server = setupServer(
  http.post('/api/v1/departments', () =>
    HttpResponse.json({
      data: { id: 'dept-new', name: 'New Dept', parent_id: null, created_at: '', updated_at: '', children: [] },
      error: null,
    }, { status: 201 }),
  ),
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

describe('DepartmentCreateModal', () => {
  it('renders name field and parent select when open', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    expect(screen.getByLabelText(/name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/parent/i)).toBeInTheDocument()
  })

  it('does not render when closed', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open={false} departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('submit button is disabled when name is empty', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
  })

  it('shows inline error on DUPLICATE_NAME response', async () => {
    server.use(
      http.post('/api/v1/departments', () =>
        HttpResponse.json({ data: null, error: { code: 'DUPLICATE_NAME', message: 'duplicate' } }, { status: 409 }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Engineering')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => {
      expect(screen.getByText(/already exists/i)).toBeInTheDocument()
    })
  })

  it('pre-fills parent when defaultParentId is provided', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open defaultParentId="dept-1" departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    const select = screen.getByLabelText(/parent/i) as HTMLSelectElement
    expect(select.value).toBe('dept-1')
  })
})
