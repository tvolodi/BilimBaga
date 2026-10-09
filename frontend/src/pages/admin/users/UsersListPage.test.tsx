import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { UsersListPage } from './UsersListPage'

const tree = [
  {
    id: 'dept-root',
    name: 'Headquarters',
    parent_id: null,
    children: [
      { id: 'dept-child', name: 'Engineering', parent_id: 'dept-root', children: [] },
    ],
  },
]

let lastUsersQuery = ''

const server = setupServer(
  http.get('/api/v1/departments', () => HttpResponse.json({ data: tree, error: null })),
  http.get('/api/v1/users/roles', () =>
    HttpResponse.json({
      data: [
        { id: 'role-uuid-ex', name: 'examiner' },
        { id: 'role-uuid-emp', name: 'employee' },
      ],
      error: null,
    }),
  ),
  http.get('/api/v1/users', ({ request }) => {
    lastUsersQuery = new URL(request.url).search
    return HttpResponse.json({
      data: { items: [], meta: { page: 1, per_page: 20, total: 0 } },
      error: null,
    })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => {
  server.resetHandlers()
  lastUsersQuery = ''
})
afterAll(() => server.close())

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  queryClient.setQueryData(['auth', 'accessToken'], `h.${btoa(JSON.stringify({ role: 'super_admin', sub: 'me-1' }))}.s`)
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
  return render(<UsersListPage />, { wrapper: Wrapper })
}

describe('admin UsersListPage department filter (FR-BB317.4)', () => {
  it('renders the department filter as a tree combobox, not a native select', async () => {
    renderPage()
    const trigger = await screen.findByRole('combobox', { name: /department/i })
    expect(trigger).toHaveAttribute('aria-haspopup', 'tree')
    expect(trigger.tagName).toBe('BUTTON')
  })

  it('shows nested departments in the tree and filters users by the chosen id', async () => {
    const user = userEvent.setup()
    renderPage()
    const trigger = await screen.findByRole('combobox', { name: /department/i })
    await waitFor(() => expect(trigger).not.toBeDisabled())
    await user.click(trigger)

    expect(await screen.findByText('Headquarters')).toBeInTheDocument()
    // child hidden until the parent is expanded (hierarchy is preserved)
    expect(screen.queryByText('Engineering')).not.toBeInTheDocument()

    await user.type(screen.getByRole('textbox', { name: /search/i }), 'Engin')
    const child = await screen.findByText('Engineering')
    await user.click(child)

    await waitFor(() => expect(lastUsersQuery).toContain('department_id=dept-child'))
    expect(trigger).toHaveTextContent('Engineering')
  })

  it('clears the department filter when the clear control is used', async () => {
    const user = userEvent.setup()
    renderPage()
    const trigger = await screen.findByRole('combobox', { name: /department/i })
    await waitFor(() => expect(trigger).not.toBeDisabled())
    await user.click(trigger)
    await user.click(await screen.findByText('Headquarters'))
    await waitFor(() => expect(lastUsersQuery).toContain('department_id=dept-root'))

    await user.click(screen.getByLabelText(/clear/i))
    await waitFor(() => expect(lastUsersQuery).not.toContain('department_id'))
  })
})

describe('admin UsersListPage role filter (ISS-133)', () => {
  it('sends the selected role UUID (not the role name) as role_id and clears it', async () => {
    const user = userEvent.setup()
    renderPage()
    const select = await screen.findByRole('combobox', { name: /role/i })
    await screen.findByRole('option', { name: 'Examiner' })

    await user.selectOptions(select, 'role-uuid-ex')
    await waitFor(() => expect(lastUsersQuery).toContain('role_id=role-uuid-ex'))
    expect(lastUsersQuery).not.toContain('role_id=examiner')

    await user.selectOptions(select, '')
    await waitFor(() => expect(lastUsersQuery).not.toContain('role_id'))
  })
})
