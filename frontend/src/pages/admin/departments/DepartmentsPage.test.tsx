import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { DepartmentsPage } from './DepartmentsPage'
import { findDepartmentById, type Department } from '@/api/departments'

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

const rootDept = {
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
}

const server = setupServer(
  http.get('/api/v1/users/me', () => HttpResponse.json({ data: meAdmin, error: null })),
  http.get('/api/v1/departments', () => HttpResponse.json({ data: [rootDept], error: null })),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  vi.useRealTimers()
})
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

function makeDept(id: string, name: string, parent_id: string | null): Department {
  return {
    id,
    name,
    parent_id,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    children: [],
  }
}

// Serves a mutable department tree so that create, rename and delete are verified
// through the refetched list the user actually sees, not only through request bodies.
function serveDepartmentTree(initial: Department[]) {
  const state = { tree: structuredClone(initial) }
  let nextId = 100
  server.use(
    http.get('/api/v1/departments', () => HttpResponse.json({ data: state.tree, error: null })),
    http.post('/api/v1/departments', async ({ request }) => {
      const body = (await request.json()) as { name: string; parent_id: string | null }
      const created = makeDept(`dept-${nextId++}`, body.name, body.parent_id)
      if (body.parent_id) {
        findDepartmentById(state.tree, body.parent_id)?.children.push(created)
      } else {
        state.tree.push(created)
      }
      return HttpResponse.json({ data: created, error: null }, { status: 201 })
    }),
    http.put('/api/v1/departments/:id', async ({ request, params }) => {
      const body = (await request.json()) as { name: string }
      const node = findDepartmentById(state.tree, params.id as string)
      if (node) node.name = body.name
      return HttpResponse.json({ data: node, error: null })
    }),
    http.delete('/api/v1/departments/:id', ({ params }) => {
      const id = params.id as string
      const removeFrom = (nodes: Department[]): Department[] =>
        nodes
          .filter((n) => n.id !== id)
          .map((n) => ({ ...n, children: removeFrom(n.children ?? []) }))
      state.tree = removeFrom(state.tree)
      return new HttpResponse(null, { status: 204 })
    }),
  )
  return state
}

function openActionsMenu(nodeName: string) {
  const row = screen.getByText(nodeName).closest('div.group') as HTMLElement
  const actions = within(row).getByRole('button', { name: 'Actions' })
  return userEvent.click(actions)
}

function dismissBannerFor(message: string) {
  return screen.getByText(message).parentElement!.querySelector('button') as HTMLButtonElement
}

describe('DepartmentsPage', () => {
  it('renders root department node from the API', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
  })

  it('renders child department node (tree is expanded by default)', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Frontend')).toBeInTheDocument()
    })
  })

  it('shows child count badge on parent node', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
    expect(screen.getByText(/1.*children|children.*1/i)).toBeInTheDocument()
  })

  it('shows New Department button for super_admin', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /new department/i })).toBeInTheDocument()
  })

  it('hides New Department button for non-super_admin', async () => {
    server.use(
      http.get('/api/v1/users/me', () =>
        HttpResponse.json({ data: { ...meAdmin, role_name: 'employee' }, error: null }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
    })
    expect(screen.queryByRole('button', { name: /new department/i })).not.toBeInTheDocument()
  })

  it('opens the create modal when New Department is clicked', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => screen.getByText('Engineering'))
    await userEvent.click(screen.getByRole('button', { name: /new department/i }))
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
  })

  it('shows empty state when no departments exist', async () => {
    server.use(
      http.get('/api/v1/departments', () => HttpResponse.json({ data: [], error: null })),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText(/no departments yet/i)).toBeInTheDocument()
    })
  })

  it('shows the loading indicator while the tree is being fetched', async () => {
    let release!: () => void
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.get('/api/v1/departments', async () => {
        await gate
        return HttpResponse.json({ data: [rootDept], error: null })
      }),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    expect(await screen.findByText('Loading…')).toBeInTheDocument()
    expect(screen.queryByText('Engineering')).not.toBeInTheDocument()
    release()
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    expect(screen.queryByText('Loading…')).not.toBeInTheDocument()
  })

  it('shows the load error and no tree when the departments request fails', async () => {
    server.use(
      http.get('/api/v1/departments', () =>
        HttpResponse.json({ data: null, error: { code: 'INTERNAL', message: 'boom' } }, { status: 500 }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    expect(await screen.findByText('Failed to load. Please try again.')).toBeInTheDocument()
    expect(screen.queryByText('Engineering')).not.toBeInTheDocument()
    expect(screen.queryByText(/no departments yet/i)).not.toBeInTheDocument()
  })

  it('hides the row action menu from a non-super_admin user', async () => {
    server.use(
      http.get('/api/v1/users/me', () =>
        HttpResponse.json({ data: { ...meAdmin, role_name: 'employee' }, error: null }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Actions' })).not.toBeInTheDocument()
  })

  it('creates a top-level department from the New Department dialog and shows it in the tree', async () => {
    serveDepartmentTree([rootDept])
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    await userEvent.click(screen.getByRole('button', { name: /new department/i }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(/name/i), 'Sales')
    await userEvent.click(within(dialog).getByRole('button', { name: /save/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Sales')).toBeInTheDocument()
  })

  it('opens Add Child with the clicked department preselected and creates the child under it', async () => {
    serveDepartmentTree([rootDept])
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    await openActionsMenu('Engineering')
    await userEvent.click(screen.getByRole('button', { name: 'Add Child' }))
    const dialog = await screen.findByRole('dialog')
    expect((within(dialog).getByLabelText(/parent/i) as HTMLSelectElement).value).toBe('dept-1')
    await userEvent.type(within(dialog).getByLabelText(/name/i), 'Backend')
    await userEvent.click(within(dialog).getByRole('button', { name: /save/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Backend')).toBeInTheDocument()
  })

  it('shows a page-level error banner when the create fails with a non-validation error and dismisses it', async () => {
    server.use(
      http.post('/api/v1/departments', () =>
        HttpResponse.json({ data: null, error: { code: 'INTERNAL', message: 'database is down' } }, { status: 500 }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    await userEvent.click(screen.getByRole('button', { name: /new department/i }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(/name/i), 'Sales')
    await userEvent.click(within(dialog).getByRole('button', { name: /save/i }))
    expect(await screen.findByText('database is down')).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    await userEvent.click(dismissBannerFor('database is down'))
    expect(screen.queryByText('database is down')).not.toBeInTheDocument()
  })

  it('renames a department through the Rename dialog and shows the new name in the tree', async () => {
    serveDepartmentTree([rootDept])
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    await openActionsMenu('Engineering')
    await userEvent.click(screen.getByRole('button', { name: 'Rename' }))
    const dialog = await screen.findByRole('dialog')
    const input = within(dialog).getByLabelText(/name/i) as HTMLInputElement
    expect(input.value).toBe('Engineering')
    await userEvent.clear(input)
    await userEvent.type(input, 'Platform')
    await userEvent.click(within(dialog).getByRole('button', { name: /save/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(await screen.findByText('Platform')).toBeInTheDocument()
    expect(screen.queryByText('Engineering')).not.toBeInTheDocument()
  })

  it('closes the Rename dialog without renaming when Cancel is clicked', async () => {
    let putCalled = false
    server.use(
      http.put('/api/v1/departments/:id', () => {
        putCalled = true
        return HttpResponse.json({ data: rootDept, error: null })
      }),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    await openActionsMenu('Engineering')
    await userEvent.click(screen.getByRole('button', { name: 'Rename' }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /cancel/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(putCalled).toBe(false)
    expect(screen.getByText('Engineering')).toBeInTheDocument()
  })

  it('deletes a child department after confirmation, removes it from the tree and shows a success banner', async () => {
    serveDepartmentTree([rootDept])
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Frontend')).toBeInTheDocument())
    await openActionsMenu('Frontend')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Frontend')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: /^delete$/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    await waitFor(() => expect(screen.queryByText('Frontend')).not.toBeInTheDocument())
    expect(screen.getByText('Engineering')).toBeInTheDocument()
    // The banner must show the success message, not the Delete button label (#326).
    const banner = document.querySelector('.bg-green-50')
    expect(banner).not.toBeNull()
    expect(banner?.textContent?.trim()).toBe('Department deleted')
  })

  it('does not send a delete request when the confirmation is cancelled', async () => {
    let deleteCalled = false
    server.use(
      http.delete('/api/v1/departments/:id', () => {
        deleteCalled = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Frontend')).toBeInTheDocument())
    await openActionsMenu('Frontend')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /cancel/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(deleteCalled).toBe(false)
    expect(screen.getByText('Frontend')).toBeInTheDocument()
  })

  it.each([
    ['DEPARTMENT_NOT_EMPTY', 'Cannot delete: department has users assigned', 'has users'],
    ['DEPARTMENT_HAS_CHILDREN', 'Cannot delete: department has child departments', 'has children'],
    ['NOT_FOUND', 'Selected parent department no longer exists', 'gone'],
    ['FORBIDDEN', 'not allowed to delete', 'not allowed to delete'],
  ])('maps the %s delete error to its banner message', async (code, expectedText, apiMessage) => {
    server.use(
      http.delete('/api/v1/departments/:id', () =>
        HttpResponse.json({ data: null, error: { code, message: apiMessage } }, { status: 409 }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Frontend')).toBeInTheDocument())
    await openActionsMenu('Frontend')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /^delete$/i }))
    expect(await screen.findByText(expectedText)).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByText('Frontend')).toBeInTheDocument()
  })

  it('hides the error banner automatically after five seconds', async () => {
    server.use(
      http.delete('/api/v1/departments/:id', () =>
        HttpResponse.json({ data: null, error: { code: 'DEPARTMENT_NOT_EMPTY', message: 'has users' } }, { status: 409 }),
      ),
    )
    // Fake only the timer APIs, and let the clock advance in real time so that
    // user-event and waitFor keep working. The page's 5s dismiss timer is then driven manually.
    vi.useFakeTimers({ shouldAdvanceTime: true, toFake: ['setTimeout', 'clearTimeout'] })
    const Wrapper = createWrapper()
    render(<Wrapper><DepartmentsPage /></Wrapper>)
    await waitFor(() => expect(screen.getByText('Frontend')).toBeInTheDocument())
    await openActionsMenu('Frontend')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /^delete$/i }))
    const message = 'Cannot delete: department has users assigned'
    expect(await screen.findByText(message)).toBeInTheDocument()
    act(() => {
      vi.advanceTimersByTime(5000)
    })
    expect(screen.queryByText(message)).not.toBeInTheDocument()
  })
})
