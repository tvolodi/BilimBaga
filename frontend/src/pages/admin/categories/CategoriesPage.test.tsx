import { describe, it, expect, beforeAll, afterAll, afterEach, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { CategoriesPage } from './CategoriesPage'

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

const sampleCategory = {
  id: 'cat-1',
  name: 'Science',
  parent_id: null,
  track: null,
  sort_order: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
}

// Mutable list served by the default GET/POST/PUT/DELETE handlers, reset before each test.
let store: Array<Record<string, unknown>> = []
// Request bodies the page sent to the mutating endpoints, in order.
let sent: Array<{ method: string; path: string; body: Record<string, unknown> | null }> = []

const server = setupServer(
  http.get('/api/v1/users/me', () => HttpResponse.json({ data: meAdmin, error: null })),
  http.get('/api/v1/categories', () => HttpResponse.json({ data: store, error: null })),
  http.post('/api/v1/categories', async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    sent.push({ method: 'POST', path: '/api/v1/categories', body })
    const node = {
      ...sampleCategory,
      id: `cat-new-${store.length + 1}`,
      name: body.name,
      parent_id: (body.parent_id as string | undefined) ?? null,
      track: (body.track as string | undefined) ?? null,
      sort_order: body.sort_order,
    }
    store = [...store, node]
    return HttpResponse.json({ data: node, error: null })
  }),
  http.put('/api/v1/categories/:id', async ({ request, params }) => {
    const body = (await request.json()) as Record<string, unknown>
    sent.push({ method: 'PUT', path: `/api/v1/categories/${params.id}`, body })
    store = store.map((c) => (c.id === params.id ? { ...c, ...body } : c))
    return HttpResponse.json({ data: store.find((c) => c.id === params.id), error: null })
  }),
  http.delete('/api/v1/categories/:id', ({ params }) => {
    sent.push({ method: 'DELETE', path: `/api/v1/categories/${params.id}`, body: null })
    store = store.filter((c) => c.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
beforeEach(() => {
  store = [sampleCategory]
  sent = []
})
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

function renderPage() {
  const Wrapper = createWrapper()
  render(
    <Wrapper>
      <CategoriesPage />
    </Wrapper>,
  )
}

// The row's actions menu is the last button in the row (after the drag handle and expand toggle).
async function openRowMenu(name: string) {
  const row = screen.getByText(name).closest('.group') as HTMLElement
  const buttons = within(row).getAllByRole('button')
  await userEvent.click(buttons[buttons.length - 1])
}

describe('CategoriesPage', () => {
  it('renders a category from the API', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Science')).toBeInTheDocument()
    })
  })

  it('shows the New Category button for super_admin', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Science')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /new category/i })).toBeInTheDocument()
  })

  it('hides New Category button for employee role', async () => {
    server.use(
      http.get('/api/v1/users/me', () =>
        HttpResponse.json({
          data: { ...meAdmin, role_name: 'employee' },
          error: null,
        }),
      ),
    )
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Science')).toBeInTheDocument()
    })
    expect(screen.queryByRole('button', { name: /new category/i })).not.toBeInTheDocument()
  })

  it('shows empty state when no categories exist', async () => {
    store = []
    renderPage()
    expect(
      await screen.findByText('No categories yet. Click "New Category" to start.'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Science')).not.toBeInTheDocument()
  })

  it('opens the create modal when New Category is clicked', async () => {
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await userEvent.click(screen.getByRole('button', { name: /new category/i }))
    // Modal or dialog should appear
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
  })

  it('shows the loading text while the tree is being fetched', async () => {
    let release: () => void = () => {}
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.get('/api/v1/categories', async () => {
        await gate
        return HttpResponse.json({ data: [sampleCategory], error: null })
      }),
    )
    renderPage()
    expect(await screen.findByText('Loading…')).toBeInTheDocument()
    release()
    expect(await screen.findByText('Science')).toBeInTheDocument()
    expect(screen.queryByText('Loading…')).not.toBeInTheDocument()
  })

  it('shows the load error and no tree when the categories request fails', async () => {
    server.use(
      http.get('/api/v1/categories', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INTERNAL', message: 'boom' } },
          { status: 500 },
        ),
      ),
    )
    renderPage()
    expect(await screen.findByText('Failed to load. Please try again.')).toBeInTheDocument()
    expect(screen.queryByText('Science')).not.toBeInTheDocument()
    expect(screen.queryByText(/no categories yet/i)).not.toBeInTheDocument()
  })

  it('creates a category through the modal and shows it in the refreshed tree', async () => {
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await userEvent.click(screen.getByRole('button', { name: /new category/i }))
    await userEvent.type(screen.getByLabelText('Name'), 'Mathematics')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('Mathematics')).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(sent).toEqual([
      { method: 'POST', path: '/api/v1/categories', body: { name: 'Mathematics', sort_order: 0 } },
    ])
  })

  it('opens the modal with the parent preselected from the row "Add child" action', async () => {
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await openRowMenu('Science')
    await userEvent.click(screen.getByRole('button', { name: 'Add child' }))

    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByRole('heading', { name: 'New Category' })).toBeInTheDocument()
    expect((within(dialog).getByLabelText('Parent category') as HTMLSelectElement).value).toBe('cat-1')

    await userEvent.type(within(dialog).getByLabelText('Name'), 'Physics')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(sent).toEqual([
      {
        method: 'POST',
        path: '/api/v1/categories',
        body: { name: 'Physics', parent_id: 'cat-1', sort_order: 0 },
      },
    ])
  })

  it('edits a category through the row Edit action and shows the new name', async () => {
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await openRowMenu('Science')
    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    const dialog = await screen.findByRole('dialog')
    const nameField = within(dialog).getByLabelText('Name') as HTMLInputElement
    expect(nameField.value).toBe('Science')
    await userEvent.clear(nameField)
    await userEvent.type(nameField, 'Life Science')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('Life Science')).toBeInTheDocument()
    expect(screen.queryByText('Science')).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(sent).toEqual([
      { method: 'PUT', path: '/api/v1/categories/cat-1', body: { name: 'Life Science' } },
    ])
  })

  it('deletes a category after confirmation, removes it and shows the success notice', async () => {
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await openRowMenu('Science')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    const confirm = await screen.findByRole('dialog')
    expect(within(confirm).getByText('Delete category')).toBeInTheDocument()
    await userEvent.click(within(confirm).getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText('Category deleted.')).toBeInTheDocument()
    expect(await screen.findByText(/no categories yet/i)).toBeInTheDocument()
    expect(screen.queryByText('Science')).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(sent).toEqual([{ method: 'DELETE', path: '/api/v1/categories/cat-1', body: null }])
  })

  it('cancelling the delete confirmation keeps the category and sends no request', async () => {
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await openRowMenu('Science')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    const confirm = await screen.findByRole('dialog')
    await userEvent.click(within(confirm).getByRole('button', { name: 'Cancel' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByText('Science')).toBeInTheDocument()
    expect(screen.queryByText('Category deleted.')).not.toBeInTheDocument()
    expect(sent).toEqual([])
  })

  it('shows the in-use message with the usage count when the delete is refused', async () => {
    server.use(
      http.delete('/api/v1/categories/:id', () =>
        HttpResponse.json(
          {
            data: null,
            error: {
              code: 'CATEGORY_IN_USE',
              message: 'in use',
              details: { in_use_count: 3 },
            },
          },
          { status: 409 },
        ),
      ),
    )
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await openRowMenu('Science')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const confirm = await screen.findByRole('dialog')
    await userEvent.click(within(confirm).getByRole('button', { name: 'Delete' }))

    expect(
      await screen.findByText('Cannot delete: 3 question(s) still use this category'),
    ).toBeInTheDocument()
    expect(screen.getByText('Science')).toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows the API message when the delete fails for another reason, and the banner can be dismissed', async () => {
    server.use(
      http.delete('/api/v1/categories/:id', () =>
        HttpResponse.json(
          { data: null, error: { code: 'FORBIDDEN', message: 'Delete blocked by server' } },
          { status: 403 },
        ),
      ),
    )
    renderPage()
    await waitFor(() => screen.getByText('Science'))
    await openRowMenu('Science')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const confirm = await screen.findByRole('dialog')
    await userEvent.click(within(confirm).getByRole('button', { name: 'Delete' }))

    const message = await screen.findByText('Delete blocked by server')
    expect(screen.getByText('Science')).toBeInTheDocument()

    // The banner's close button is the only button inside the notification.
    await userEvent.click(message.parentElement!.querySelector('button')!)
    expect(screen.queryByText('Delete blocked by server')).not.toBeInTheDocument()
  })
})
