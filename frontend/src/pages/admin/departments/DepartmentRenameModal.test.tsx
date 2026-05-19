import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { DepartmentRenameModal } from './DepartmentRenameModal'
import { type Department } from '@/api/departments'

const dept: Department = {
  id: 'dept-1',
  name: 'Engineering',
  parent_id: null,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
}

const server = setupServer(
  http.put('/api/v1/departments/:id', () =>
    HttpResponse.json({ data: { ...dept, name: 'Renamed' }, error: null }),
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

describe('DepartmentRenameModal', () => {
  it('renders name field pre-filled with current department name', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={() => {}} />
      </Wrapper>,
    )
    const input = screen.getByLabelText(/name/i) as HTMLInputElement
    expect(input.value).toBe('Engineering')
  })

  it('does not render when closed', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open={false} department={dept} onClose={() => {}} />
      </Wrapper>,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('submit button is disabled when name is empty', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={() => {}} />
      </Wrapper>,
    )
    const input = screen.getByLabelText(/name/i)
    await userEvent.clear(input)
    expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
  })

  it('shows inline error on DUPLICATE_NAME response', async () => {
    server.use(
      http.put('/api/v1/departments/:id', () =>
        HttpResponse.json({ data: null, error: { code: 'DUPLICATE_NAME', message: 'duplicate' } }, { status: 409 }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={() => {}} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => {
      expect(screen.getByText(/already exists/i)).toBeInTheDocument()
    })
  })
})
