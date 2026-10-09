import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
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

const nestedDepts: Department[] = [
  {
    id: 'dept-1',
    name: 'Engineering',
    parent_id: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    children: [
      {
        id: 'dept-2',
        name: 'Frontend',
        parent_id: 'dept-1',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        children: [],
      },
    ],
  },
]

const createdDept = {
  id: 'dept-new',
  name: 'New Dept',
  parent_id: null,
  created_at: '',
  updated_at: '',
  children: [],
}

const server = setupServer(
  http.post('/api/v1/departments', () =>
    HttpResponse.json({ data: createdDept, error: null }, { status: 201 }),
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

  it('submit button is disabled when the name is only whitespace', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), '   ')
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

  it('does not close the modal when the name is a duplicate', async () => {
    server.use(
      http.post('/api/v1/departments', () =>
        HttpResponse.json({ data: null, error: { code: 'DUPLICATE_NAME', message: 'duplicate' } }, { status: 409 }),
      ),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Engineering')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(screen.getByText(/already exists/i)).toBeInTheDocument())
    expect(onClose).not.toHaveBeenCalled()
  })

  it('clears the inline duplicate error as soon as the name is edited', async () => {
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
    await waitFor(() => expect(screen.getByText(/already exists/i)).toBeInTheDocument())
    await userEvent.type(screen.getByLabelText(/name/i), 'X')
    expect(screen.queryByText(/already exists/i)).not.toBeInTheDocument()
  })

  it('shows the parent-not-found message inline on NOT_FOUND', async () => {
    server.use(
      http.post('/api/v1/departments', () =>
        HttpResponse.json({ data: null, error: { code: 'NOT_FOUND', message: 'gone' } }, { status: 404 }),
      ),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Sales')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() =>
      expect(screen.getByText('Selected parent department no longer exists')).toBeInTheDocument(),
    )
    expect(onClose).not.toHaveBeenCalled()
  })

  it('passes a non-validation API error message up to the page via onClose', async () => {
    server.use(
      http.post('/api/v1/departments', () =>
        HttpResponse.json({ data: null, error: { code: 'INTERNAL', message: 'database is down' } }, { status: 500 }),
      ),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Sales')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(onClose).toHaveBeenCalledWith('database is down'))
  })

  it('POSTs the trimmed name with a null parent and closes without an error', async () => {
    let body: unknown
    server.use(
      http.post('/api/v1/departments', async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ data: createdDept, error: null }, { status: 201 })
      }),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), '  Sales  ')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(onClose).toHaveBeenCalledWith()
    expect(body).toEqual({ name: 'Sales', parent_id: null })
  })

  it('POSTs the selected parent id when a parent is chosen from the select', async () => {
    let body: unknown
    server.use(
      http.post('/api/v1/departments', async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ data: createdDept, error: null }, { status: 201 })
      }),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={nestedDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Mobile')
    await userEvent.selectOptions(screen.getByLabelText(/parent/i), 'dept-2')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(body).toEqual({ name: 'Mobile', parent_id: 'dept-2' })
  })

  it('keeps the submit button disabled while the create request is in flight', async () => {
    let release!: () => void
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.post('/api/v1/departments', async () => {
        await gate
        return HttpResponse.json({ data: createdDept, error: null }, { status: 201 })
      }),
    )
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Sales')
    await userEvent.click(screen.getByRole('button', { name: /save/i }))
    await waitFor(() => expect(screen.getByRole('button', { name: /save/i })).toBeDisabled())
    expect(onClose).not.toHaveBeenCalled()
    release()
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('calls onClose without an error when Cancel is clicked', async () => {
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onClose).toHaveBeenCalledOnce()
    expect(onClose).toHaveBeenCalledWith()
  })

  it('calls onClose when the dialog is dismissed with the Escape key', async () => {
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={onClose} />
      </Wrapper>,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  it('caps the name at 100 characters', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    const input = screen.getByLabelText(/name/i) as HTMLInputElement
    await userEvent.type(input, 'a'.repeat(105))
    expect(input.value).toHaveLength(100)
  })

  it('lists every department, including nested children, and a top-level option', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentCreateModal open departments={nestedDepts} onClose={() => {}} />
      </Wrapper>,
    )
    const select = screen.getByLabelText(/parent/i) as HTMLSelectElement
    const labels = Array.from(select.options).map((o) => o.textContent)
    expect(labels).toEqual(['(Top level)', 'Engineering', 'Frontend'])
    expect(select.value).toBe('')
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

  it('resets the name and parent fields each time the dialog is reopened', async () => {
    const Wrapper = createWrapper()
    const { rerender } = render(
      <Wrapper>
        <DepartmentCreateModal open defaultParentId="dept-1" departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    await userEvent.type(screen.getByLabelText(/name/i), 'Draft')
    rerender(
      <Wrapper>
        <DepartmentCreateModal open={false} defaultParentId="dept-1" departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    rerender(
      <Wrapper>
        <DepartmentCreateModal open defaultParentId={null} departments={sampleDepts} onClose={() => {}} />
      </Wrapper>,
    )
    const input = screen.getByLabelText(/name/i) as HTMLInputElement
    const select = screen.getByLabelText(/parent/i) as HTMLSelectElement
    expect(input.value).toBe('')
    expect(select.value).toBe('')
  })
})
