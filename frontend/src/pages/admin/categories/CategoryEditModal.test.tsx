import { describe, it, expect, vi, beforeAll, afterAll, afterEach, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import type { ComponentProps, ReactNode } from 'react'
import '@/i18n'
import { CategoryEditModal } from './CategoryEditModal'
import type { CategoryNode } from '@/api/categories'

// ---- Fixtures ---------------------------------------------------------------

const makeNode = (overrides: Partial<CategoryNode>): CategoryNode => ({
  id: 'cat-default',
  name: 'Default',
  parent_id: null,
  track: null,
  sort_order: 0,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
  ...overrides,
})

// Alpha (track compliance) > Beta > Gamma ; Delta (top level)
const gamma = makeNode({ id: 'cat-gamma', name: 'Gamma', parent_id: 'cat-beta', sort_order: 0 })
const beta = makeNode({
  id: 'cat-beta',
  name: 'Beta',
  parent_id: 'cat-alpha',
  sort_order: 2,
  children: [gamma],
})
const alpha = makeNode({
  id: 'cat-alpha',
  name: 'Alpha',
  track: 'compliance',
  sort_order: 1,
  children: [beta],
})
const delta = makeNode({ id: 'cat-delta', name: 'Delta', sort_order: 3 })
const tree: CategoryNode[] = [alpha, delta]

// ---- MSW --------------------------------------------------------------------

type Sent = { method: string; path: string; body: Record<string, unknown> }
let sent: Sent[] = []

async function recordOk({ request }: { request: Request }) {
  const body = (await request.json()) as Record<string, unknown>
  sent.push({ method: request.method, path: new URL(request.url).pathname, body })
  return HttpResponse.json({ data: { id: 'saved' }, error: null })
}

const server = setupServer(
  http.post('/api/v1/categories', recordOk),
  http.put('/api/v1/categories/:id', recordOk),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
beforeEach(() => {
  sent = []
})
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// ---- Helpers ----------------------------------------------------------------

function renderModal(props: Partial<ComponentProps<typeof CategoryEditModal>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const onClose = vi.fn()
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
  const utils = render(
    <CategoryEditModal open onClose={onClose} tree={tree} editing={null} {...props} />,
    { wrapper },
  )
  return { ...utils, onClose, queryClient }
}

const nameInput = () => screen.getByLabelText('Name') as HTMLInputElement
const parentSelect = () => screen.getByLabelText('Parent category') as HTMLSelectElement
const trackInput = () => screen.getByLabelText('Track') as HTMLInputElement
const sortInput = () => screen.getByLabelText('Sort order') as HTMLInputElement
const saveButton = () => screen.getByRole('button', { name: 'Save' })

// ---- Create mode ------------------------------------------------------------

describe('CategoryEditModal - create mode', () => {
  it('shows the new-category title with an empty form and top-level parent', () => {
    renderModal()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'New Category' })).toBeInTheDocument()
    expect(nameInput().value).toBe('')
    expect(trackInput().value).toBe('')
    expect(sortInput().value).toBe('0')
    expect(parentSelect().value).toBe('')
  })

  it('lists every category as an indented option under the top-level entry', () => {
    renderModal()
    const labels = Array.from(parentSelect().options).map((o) => o.textContent)
    expect(labels).toEqual([
      '(Top level)',
      'Alpha',
      'Alpha › Beta',
      'Alpha › Beta › Gamma',
      'Delta',
    ])
  })

  it('lists a category whose parent is missing from the tree without an ancestor prefix', () => {
    const orphan = makeNode({ id: 'cat-orphan', name: 'Orphan', parent_id: 'cat-ghost' })
    renderModal({ tree: [orphan] })
    const labels = Array.from(parentSelect().options).map((o) => o.textContent)
    expect(labels).toEqual(['(Top level)', 'Orphan'])
  })

  it('creates a top-level category with a trimmed name and no parent or track', async () => {
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), '  Physics  ')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent).toEqual([
      { method: 'POST', path: '/api/v1/categories', body: { name: 'Physics', sort_order: 0 } },
    ])
  })

  it('sends parent, trimmed track and sort order when they are set', async () => {
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), 'Chemistry')
    await userEvent.selectOptions(parentSelect(), 'cat-beta')
    await userEvent.type(trackInput(), '  safety  ')
    fireEvent.change(sortInput(), { target: { value: '5' } })
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({
      name: 'Chemistry',
      parent_id: 'cat-beta',
      track: 'safety',
      sort_order: 5,
    })
  })

  it('pre-fills the parent from defaultParentId ("Add child")', async () => {
    const { onClose } = renderModal({ defaultParentId: 'cat-delta' })
    expect(parentSelect().value).toBe('cat-delta')
    await userEvent.type(nameInput(), 'Epsilon')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ name: 'Epsilon', parent_id: 'cat-delta', sort_order: 0 })
  })

  it('omits the track when it is only whitespace', async () => {
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), 'Biology')
    await userEvent.type(trackInput(), '   ')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).not.toHaveProperty('track')
  })

  it('blocks submit and shows the name error when the name is empty', async () => {
    const { onClose } = renderModal()
    await userEvent.click(saveButton())
    expect(await screen.findByText('Name is required (max 100 characters)')).toBeInTheDocument()
    expect(sent).toEqual([])
    expect(onClose).not.toHaveBeenCalled()
  })

  it('blocks submit when the name is longer than 100 characters', async () => {
    const { onClose } = renderModal()
    fireEvent.change(nameInput(), { target: { value: 'x'.repeat(101) } })
    await userEvent.click(saveButton())
    expect(await screen.findByText('Name is required (max 100 characters)')).toBeInTheDocument()
    expect(sent).toEqual([])
    expect(onClose).not.toHaveBeenCalled()
  })

  it('blocks submit when the sort order is not a number', async () => {
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), 'Geography')
    fireEvent.change(sortInput(), { target: { value: '' } })
    await userEvent.click(saveButton())
    // Validation stops the request; the modal stays open for correction.
    await waitFor(() => expect(saveButton()).toBeEnabled())
    expect(sent).toEqual([])
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('shows the cycle error under Parent when the API reports CATEGORY_CYCLE', async () => {
    server.use(
      http.post('/api/v1/categories', () =>
        HttpResponse.json(
          { data: null, error: { code: 'CATEGORY_CYCLE', message: 'cycle' } },
          { status: 400 },
        ),
      ),
    )
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), 'Loop')
    await userEvent.click(saveButton())
    expect(
      await screen.findByText('Cannot set this parent — it would create a cycle'),
    ).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('shows the parent-not-found error under Parent when the API reports PARENT_NOT_FOUND', async () => {
    server.use(
      http.post('/api/v1/categories', () =>
        HttpResponse.json(
          { data: null, error: { code: 'PARENT_NOT_FOUND', message: 'missing' } },
          { status: 404 },
        ),
      ),
    )
    renderModal({ defaultParentId: 'cat-delta' })
    await userEvent.type(nameInput(), 'Orphan')
    await userEvent.click(saveButton())
    expect(await screen.findByText('Selected parent no longer exists')).toBeInTheDocument()
  })

  it('shows the invalid-name error under Name when the API reports INVALID_NAME', async () => {
    server.use(
      http.post('/api/v1/categories', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INVALID_NAME', message: 'bad name' } },
          { status: 400 },
        ),
      ),
    )
    renderModal()
    await userEvent.type(nameInput(), 'Server rejects me')
    await userEvent.click(saveButton())
    expect(await screen.findByText('Name is required (max 100 characters)')).toBeInTheDocument()
  })

  it('shows the API message in the form for any other error code', async () => {
    server.use(
      http.post('/api/v1/categories', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INTERNAL', message: 'Server exploded' } },
          { status: 500 },
        ),
      ),
    )
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), 'Crash')
    await userEvent.click(saveButton())
    expect(await screen.findByText('Server exploded')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('closes without sending anything when Cancel is clicked', async () => {
    const { onClose } = renderModal()
    await userEvent.type(nameInput(), 'Discarded')
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(sent).toEqual([])
  })

  it('closes when the dialog is dismissed with Escape', async () => {
    const { onClose } = renderModal()
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

// ---- Edit mode --------------------------------------------------------------

describe('CategoryEditModal - edit mode', () => {
  it('pre-fills the form from the node being edited', () => {
    renderModal({ editing: beta })
    expect(screen.getByRole('heading', { name: 'Edit' })).toBeInTheDocument()
    expect(nameInput().value).toBe('Beta')
    expect(parentSelect().value).toBe('cat-alpha')
    expect(trackInput().value).toBe('')
    expect(sortInput().value).toBe('2')
  })

  it('pre-fills the track of a node that has one', () => {
    renderModal({ editing: alpha })
    expect(trackInput().value).toBe('compliance')
    expect(parentSelect().value).toBe('')
  })

  it('excludes the node and its descendants from the parent options', () => {
    renderModal({ editing: beta })
    const labels = Array.from(parentSelect().options).map((o) => o.textContent)
    expect(labels).toEqual(['(Top level)', 'Alpha', 'Delta'])
  })

  it('sends an empty PUT body when nothing changed', async () => {
    const { onClose } = renderModal({ editing: beta })
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent).toEqual([{ method: 'PUT', path: '/api/v1/categories/cat-beta', body: {} }])
  })

  it('sends only the changed name, trimmed', async () => {
    const { onClose } = renderModal({ editing: beta })
    await userEvent.clear(nameInput())
    await userEvent.type(nameInput(), '  Beta Two ')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ name: 'Beta Two' })
  })

  it('sends a new track when one is entered', async () => {
    const { onClose } = renderModal({ editing: beta })
    await userEvent.type(trackInput(), ' ops ')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ track: 'ops' })
  })

  it('sends track: null when the track is cleared', async () => {
    const { onClose } = renderModal({ editing: alpha })
    await userEvent.clear(trackInput())
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ track: null })
  })

  it('sends the new sort order when it changes', async () => {
    const { onClose } = renderModal({ editing: beta })
    fireEvent.change(sortInput(), { target: { value: '7' } })
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ sort_order: 7 })
  })

  it('sends clear_parent when the node is moved to the top level', async () => {
    const { onClose } = renderModal({ editing: beta })
    await userEvent.selectOptions(parentSelect(), '')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ clear_parent: true })
  })

  it('sends the new parent_id when the node is moved under another category', async () => {
    const { onClose } = renderModal({ editing: beta })
    await userEvent.selectOptions(parentSelect(), 'cat-delta')
    await userEvent.click(saveButton())
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(sent[0].body).toEqual({ parent_id: 'cat-delta' })
  })

  it('shows the cycle error when the API rejects the new parent', async () => {
    server.use(
      http.put('/api/v1/categories/:id', () =>
        HttpResponse.json(
          { data: null, error: { code: 'CATEGORY_CYCLE', message: 'cycle' } },
          { status: 400 },
        ),
      ),
    )
    const { onClose } = renderModal({ editing: beta })
    await userEvent.selectOptions(parentSelect(), 'cat-delta')
    await userEvent.click(saveButton())
    expect(
      await screen.findByText('Cannot set this parent — it would create a cycle'),
    ).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('shows the generic API message when the update fails for another reason', async () => {
    server.use(
      http.put('/api/v1/categories/:id', () =>
        HttpResponse.json(
          { data: null, error: { code: 'FORBIDDEN', message: 'Not allowed to edit' } },
          { status: 403 },
        ),
      ),
    )
    renderModal({ editing: beta })
    await userEvent.type(trackInput(), 'x')
    await userEvent.click(saveButton())
    expect(await screen.findByText('Not allowed to edit')).toBeInTheDocument()
  })

  it('disables the form and shows a placeholder while the save is in flight', async () => {
    let release: () => void = () => {}
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.put('/api/v1/categories/:id', async () => {
        await gate
        return HttpResponse.json({ data: { id: 'cat-beta' }, error: null })
      }),
    )
    const { onClose } = renderModal({ editing: beta })
    await userEvent.type(nameInput(), ' 2')
    await userEvent.click(saveButton())

    await waitFor(() => expect(nameInput()).toBeDisabled())
    expect(parentSelect()).toBeDisabled()
    expect(trackInput()).toBeDisabled()
    expect(sortInput()).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '…' })).toBeDisabled()

    release()
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('reloads the form when a different node is opened for editing', () => {
    const { rerender } = renderModal({ editing: beta })
    expect(nameInput().value).toBe('Beta')
    rerender(<CategoryEditModal open onClose={vi.fn()} tree={tree} editing={alpha} />)
    expect(nameInput().value).toBe('Alpha')
    expect(trackInput().value).toBe('compliance')
  })
})
