import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
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
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
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

  it('submit button is disabled when the name is only whitespace', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={() => {}} />
      </Wrapper>,
    )
    await userEvent.clear(screen.getByLabelText(/name/i))
    await userEvent.type(screen.getByLabelText(/name/i), '   ')
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

  it('PUTs the trimmed new name for the department and closes on success', async () => {
    let body: unknown
    let requestedId: string | undefined
    server.use(
      http.put('/api/v1/departments/:id', async ({ request, params }) => {
        requestedId = params.id as string
        body = await request.json()
        return HttpResponse.json({ data: { ...dept, name: 'Platform' }, error: null })
      }),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={onClose} />
      </Wrapper>,
    )
    const input = screen.getByLabelText(/name/i)
    await userEvent.clear(input)
    await userEvent.type(input, '  Platform  ')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(requestedId).toBe('dept-1')
    expect(body).toEqual({ name: 'Platform' })
  })

  it('shows the parent-not-found message inline on NOT_FOUND', async () => {
    server.use(
      http.put('/api/v1/departments/:id', () =>
        HttpResponse.json({ data: null, error: { code: 'NOT_FOUND', message: 'gone' } }, { status: 404 }),
      ),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() =>
      expect(screen.getByText('Selected parent department no longer exists')).toBeInTheDocument(),
    )
    expect(onClose).not.toHaveBeenCalled()
  })

  it('shows the raw API message inline for any other error code', async () => {
    server.use(
      http.put('/api/v1/departments/:id', () =>
        HttpResponse.json({ data: null, error: { code: 'INTERNAL', message: 'database is down' } }, { status: 500 }),
      ),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(screen.getByText('database is down')).toBeInTheDocument())
    expect(onClose).not.toHaveBeenCalled()
  })

  it('clears the inline error as soon as the name is edited', async () => {
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
    await waitFor(() => expect(screen.getByText(/already exists/i)).toBeInTheDocument())
    await userEvent.type(screen.getByLabelText(/name/i), 'x')
    expect(screen.queryByText(/already exists/i)).not.toBeInTheDocument()
  })

  it('calls onClose without saving when Cancel is clicked', async () => {
    let putCalled = false
    server.use(
      http.put('/api/v1/departments/:id', () => {
        putCalled = true
        return HttpResponse.json({ data: dept, error: null })
      }),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onClose).toHaveBeenCalledOnce()
    expect(putCalled).toBe(false)
  })

  it('calls onClose when the dialog is dismissed with the Escape key', async () => {
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  it('renders an empty name and a disabled save button when no department is supplied', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={null} onClose={() => {}} />
      </Wrapper>,
    )
    expect((screen.getByLabelText(/name/i) as HTMLInputElement).value).toBe('')
    expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
  })

  it('does not send a request when submitted without a department', async () => {
    let putCalled = false
    server.use(
      http.put('/api/v1/departments/:id', () => {
        putCalled = true
        return HttpResponse.json({ data: dept, error: null })
      }),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentRenameModal open department={null} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Orphan')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    expect(putCalled).toBe(false)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('re-fills the name from the newly supplied department when reopened', () => {
    const Wrapper = createWrapper()
    const { rerender } = render(
      <Wrapper>
        <DepartmentRenameModal open department={dept} onClose={() => {}} />
      </Wrapper>,
    )
    const other: Department = { ...dept, id: 'dept-2', name: 'Sales' }
    rerender(
      <Wrapper>
        <DepartmentRenameModal open department={other} onClose={() => {}} />
      </Wrapper>,
    )
    expect((screen.getByLabelText(/name/i) as HTMLInputElement).value).toBe('Sales')
  })
})
