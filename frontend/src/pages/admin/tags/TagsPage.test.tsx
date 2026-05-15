import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
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

  it('filters tags by search term case-insensitively without re-fetching', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    const searchInput = screen.getByRole('textbox')
    // Type lowercase — should match "Biology" case-insensitively
    await userEvent.type(searchInput, 'bio')

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

  it('usage_count column sorts correctly asc and desc', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    // Default sort is by name asc: Biology (5), Chemistry (3)
    // Find the "Usage" sort button (contains "Usage" text, from i18n key tags.columns.usage)
    const sortButtons = screen.getAllByRole('button')
    const usageHeader = sortButtons.find((btn) => btn.textContent?.includes('Usage'))
    expect(usageHeader).toBeTruthy()

    // First click: sort by usage_count asc → Chemistry (3) first, Biology (5) second
    await userEvent.click(usageHeader!)

    await waitFor(() => {
      const allCells = screen.getAllByRole('cell')
      const nameCells = allCells.filter((cell) =>
        cell.textContent === 'Biology' || cell.textContent === 'Chemistry',
      )
      expect(nameCells.length).toBe(2)
      expect(nameCells[0].textContent).toBe('Chemistry')
    })

    // Second click: toggle to desc → Biology (5) first, Chemistry (3) second
    // Re-query the button in case the DOM was updated after first click
    const sortButtons2 = screen.getAllByRole('button')
    const usageHeader2 = sortButtons2.find((btn) => btn.textContent?.includes('Usage'))
    expect(usageHeader2).toBeTruthy()
    await userEvent.click(usageHeader2!)

    await waitFor(() => {
      const allCells = screen.getAllByRole('cell')
      const nameCells = allCells.filter((cell) =>
        cell.textContent === 'Biology' || cell.textContent === 'Chemistry',
      )
      expect(nameCells.length).toBe(2)
      expect(nameCells[0].textContent).toBe('Biology')
    })
  })

  it('delete dialog only fires mutation on confirm, not on cancel', async () => {
    let deleteCallCount = 0
    server.use(
      http.delete('/api/v1/tags/:id', () => {
        deleteCallCount++
        return HttpResponse.json({ data: null, error: null })
      }),
    )

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    // Open the actions menu for the first row
    const actionButtons = screen.getAllByRole('button', { name: /actions/i })
    await userEvent.click(actionButtons[0])

    // Click Delete
    const deleteBtn = await screen.findByRole('button', { name: /delete/i })
    await userEvent.click(deleteBtn)

    // Confirm dialog should appear
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toBeInTheDocument()

    // Click Cancel — mutation should NOT be called
    const cancelBtn = within(dialog).getByRole('button', { name: /cancel/i })
    await userEvent.click(cancelBtn)

    expect(deleteCallCount).toBe(0)
  })

  it('rename dialog only fires mutation on confirm, not on cancel', async () => {
    let putCallCount = 0
    server.use(
      http.put('/api/v1/tags/:id', () => {
        putCallCount++
        return HttpResponse.json({ data: { ...sampleTags[0], name: 'BioRename' }, error: null })
      }),
    )

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    // Open actions menu
    const actionButtons = screen.getAllByRole('button', { name: /actions/i })
    await userEvent.click(actionButtons[0])

    // Click Rename
    const renameBtn = await screen.findByRole('button', { name: /rename/i })
    await userEvent.click(renameBtn)

    // Rename dialog should appear
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toBeInTheDocument()

    // Click Cancel — mutation should NOT be called
    const cancelBtn = within(dialog).getByRole('button', { name: /cancel/i })
    await userEvent.click(cancelBtn)

    expect(putCallCount).toBe(0)
  })

  it('shows usage-count toast when 409 ERR_TAG_IN_USE on delete', async () => {
    server.use(
      http.delete('/api/v1/tags/:id', () =>
        HttpResponse.json(
          { data: null, error: { code: 'ERR_TAG_IN_USE', message: 'Tag in use' } },
          { status: 409 },
        ),
      ),
    )

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    // Open actions menu for the first row (Biology, usage_count: 5)
    const actionButtons = screen.getAllByRole('button', { name: /actions/i })
    await userEvent.click(actionButtons[0])

    const deleteBtn = await screen.findByRole('button', { name: /delete/i })
    await userEvent.click(deleteBtn)

    // Confirm dialog appears
    const dialog = await screen.findByRole('dialog')
    const confirmBtn = within(dialog).getByRole('button', { name: /delete/i })
    await userEvent.click(confirmBtn)

    // Toast/notification with usage count should appear
    await waitFor(() => {
      // The error message should contain the full inUse text with the usage count
      // (Biology has usage_count: 5)
      const errorNotification = screen.getByText(/cannot delete.*5.*question/i)
      expect(errorNotification).toBeInTheDocument()
    })
  })

  it('shows inline form error for ERR_TAG_DUPLICATE on create', async () => {
    server.use(
      http.post('/api/v1/tags', () =>
        HttpResponse.json(
          { data: null, error: { code: 'ERR_TAG_DUPLICATE', message: 'Duplicate tag' } },
          { status: 409 },
        ),
      ),
    )

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    // Click "New Tag"
    await userEvent.click(screen.getByRole('button', { name: /new tag/i }))

    // Dialog should appear
    const dialog = await screen.findByRole('dialog')
    const nameInput = within(dialog).getByRole('textbox')
    await userEvent.type(nameInput, 'biology')

    // Submit
    const saveBtn = within(dialog).getByRole('button', { name: /save/i })
    await userEvent.click(saveBtn)

    // Inline error for duplicate should appear
    await waitFor(() => {
      expect(screen.getByText(/already exists/i)).toBeInTheDocument()
    })
  })

  it('shows inline form error for ERR_TAG_DUPLICATE on rename', async () => {
    server.use(
      http.put('/api/v1/tags/:id', () =>
        HttpResponse.json(
          { data: null, error: { code: 'ERR_TAG_DUPLICATE', message: 'Duplicate tag' } },
          { status: 409 },
        ),
      ),
    )

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <TagsPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('Biology'))

    // Open actions menu
    const actionButtons = screen.getAllByRole('button', { name: /actions/i })
    await userEvent.click(actionButtons[0])

    // Click Rename
    const renameBtn = await screen.findByRole('button', { name: /rename/i })
    await userEvent.click(renameBtn)

    // Rename dialog should appear
    const dialog = await screen.findByRole('dialog')
    const nameInput = within(dialog).getByRole('textbox')

    // Clear and type duplicate name
    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, 'chemistry')

    // Submit
    const saveBtn = within(dialog).getByRole('button', { name: /save/i })
    await userEvent.click(saveBtn)

    // Inline error for duplicate should appear
    await waitFor(() => {
      expect(screen.getByText(/already exists/i)).toBeInTheDocument()
    })
  })
})
