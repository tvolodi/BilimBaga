import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { type ReactNode } from 'react'
import i18n from '@/i18n'
import { CategoryTreeNode } from './CategoryTreeNode'
import { type CategoryNode } from '@/api/categories'

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

const noop = () => {}

// --------------------------------------------------------------------------
// Fixtures
// --------------------------------------------------------------------------

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

// Row for the node named `name`: the element that holds its name span
function rowFor(name: string): HTMLElement {
  return screen.getByText(name).parentElement as HTMLElement
}

// Actions trigger inside a row: the only small (h-6) button in the row
function menuTriggerFor(name: string): HTMLButtonElement {
  const trigger = rowFor(name).querySelector<HTMLButtonElement>('button.h-6')
  if (!trigger) throw new Error(`no actions trigger rendered for ${name}`)
  return trigger
}

// Expand/collapse toggle inside a row: the first button when there is no drag handle
function toggleFor(name: string): HTMLButtonElement {
  const toggle = rowFor(name).querySelector<HTMLButtonElement>("button")
  if (!toggle) throw new Error(`no expand toggle rendered for ${name}`)
  return toggle
}

// Drag handle inside a row: the button with tabIndex -1 (the expand toggle is focusable)
function dragHandleFor(name: string): HTMLButtonElement | null {
  return rowFor(name).querySelector<HTMLButtonElement>('button[tabindex="-1"]')
}

// Two-level tree: Root > [ChildA (track "safety", with GrandChild), ChildB]
const grandChild = makeNode({ id: 'gc', name: 'GrandChild', sort_order: 0, parent_id: 'child-a' })
const childA = makeNode({
  id: 'child-a',
  name: 'ChildA',
  sort_order: 0,
  parent_id: 'root',
  track: 'safety',
  children: [grandChild],
})
const childB = makeNode({ id: 'child-b', name: 'ChildB', sort_order: 1, parent_id: 'root' })
const root = makeNode({
  id: 'root',
  name: 'RootCat',
  sort_order: 0,
  track: 'compliance',
  children: [childA, childB],
})

// --------------------------------------------------------------------------
// Tests: row content (name, track badge, child count, indentation)
// --------------------------------------------------------------------------

describe('CategoryTreeNode — row content', () => {
  it('shows the name, track badge and child count for a node with children', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(screen.getByText('RootCat')).toBeInTheDocument()
    expect(screen.getByText('compliance')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('omits the track badge and child count for a leaf without a track', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'leaf', name: 'LeafCat', sort_order: 0 })}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(screen.getByText('LeafCat')).toBeInTheDocument()
    expect(screen.queryByText('0')).not.toBeInTheDocument()
    expect(screen.queryByText('compliance')).not.toBeInTheDocument()
  })

  it('indents the row by depth * 20 + 8 pixels', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'indented', name: 'IndentedCat', sort_order: 0 })}
          depth={3}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(rowFor('IndentedCat')).toHaveStyle({ paddingLeft: '68px' })
  })
})

// --------------------------------------------------------------------------
// Tests: expand / collapse and nested rendering
// --------------------------------------------------------------------------

describe('CategoryTreeNode — expand and collapse', () => {
  it('shows children by default and collapses them on toggle', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(screen.getByText('ChildA')).toBeInTheDocument()
    expect(screen.getByText('ChildB')).toBeInTheDocument()

    fireEvent.click(toggleFor('RootCat'))

    expect(screen.queryByText('ChildA')).not.toBeInTheDocument()
    expect(screen.queryByText('ChildB')).not.toBeInTheDocument()
    // The node itself stays visible when collapsed
    expect(screen.getByText('RootCat')).toBeInTheDocument()
    expect(toggleFor('RootCat')).toHaveAttribute('aria-expanded', 'false')
  })

  it('names the toggle from the translated label in the current language (#390)', async () => {
    await i18n.changeLanguage('kk')
    try {
      const Wrapper = createWrapper()
      render(
        <Wrapper>
          <CategoryTreeNode
            node={root}
            depth={0}
            canManage={false}
            onAddChild={noop}
            onEdit={noop}
            onDelete={noop}
          />
        </Wrapper>,
      )
      expect(toggleFor('RootCat')).toHaveAccessibleName(i18n.t('common.toggleExpand', { lng: 'kk' }))
      expect(toggleFor('RootCat')).not.toHaveAccessibleName('Expand')
    } finally {
      await i18n.changeLanguage('en')
    }
  })

  it('re-expands children on a second toggle', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    fireEvent.click(toggleFor('RootCat'))
    expect(toggleFor('RootCat')).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(toggleFor('RootCat'))

    expect(screen.getByText('ChildA')).toBeInTheDocument()
    expect(screen.getByText('ChildB')).toBeInTheDocument()
    expect(toggleFor('RootCat')).toHaveAttribute('aria-expanded', 'true')
  })

  it('renders nested children one level deeper than their parent', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(rowFor('RootCat')).toHaveStyle({ paddingLeft: '8px' })
    expect(rowFor('ChildA')).toHaveStyle({ paddingLeft: '28px' })
    expect(rowFor('GrandChild')).toHaveStyle({ paddingLeft: '48px' })
  })

  it('collapsing a parent hides its whole subtree, and a child can be collapsed independently', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    // Collapse ChildA only: GrandChild hides, ChildB stays
    fireEvent.click(toggleFor('ChildA'))
    expect(screen.queryByText('GrandChild')).not.toBeInTheDocument()
    expect(screen.getByText('ChildA')).toBeInTheDocument()
    expect(screen.getByText('ChildB')).toBeInTheDocument()

    // Collapse the root: the whole subtree hides
    fireEvent.click(toggleFor('RootCat'))
    expect(screen.queryByText('ChildA')).not.toBeInTheDocument()
    expect(screen.queryByText('ChildB')).not.toBeInTheDocument()
    expect(screen.queryByText('GrandChild')).not.toBeInTheDocument()
  })
})

// --------------------------------------------------------------------------
// Tests: canManage=false (read-only)
// --------------------------------------------------------------------------

describe('CategoryTreeNode — read-only (canManage=false)', () => {
  it('renders no drag handle and no actions trigger', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(dragHandleFor('RootCat')).toBeNull()
    expect(rowFor('RootCat').querySelector('button.h-6')).toBeNull()
    expect(rowFor('ChildA').querySelector('button.h-6')).toBeNull()
  })

  it('renders only the expand toggle as a button on a leaf node', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'leaf', name: 'LeafCat', sort_order: 0 })}
          depth={0}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(screen.getAllByRole('button')).toHaveLength(1)
  })
})

// --------------------------------------------------------------------------
// Tests: canManage=true (drag handle and actions menu)
// --------------------------------------------------------------------------

describe('CategoryTreeNode — manage mode (canManage=true)', () => {
  it('renders a drag handle for the node', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(dragHandleFor('ManagedCat')).not.toBeNull()
  })

  it('keeps the actions menu closed until its trigger is clicked', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(screen.queryByRole('button', { name: 'Add child' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  })

  it('opens the menu with add child, edit and delete actions', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('ManagedCat'))

    expect(screen.getByRole('button', { name: 'Add child' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
  })

  it('calls onAddChild with the node and closes the menu', () => {
    const onAddChild = vi.fn()
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    const node = makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={node}
          depth={0}
          canManage={true}
          onAddChild={onAddChild}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('ManagedCat'))
    fireEvent.click(screen.getByRole('button', { name: 'Add child' }))

    expect(onAddChild).toHaveBeenCalledTimes(1)
    expect(onAddChild).toHaveBeenCalledWith(node)
    expect(onEdit).not.toHaveBeenCalled()
    expect(onDelete).not.toHaveBeenCalled()
    expect(screen.queryByRole('button', { name: 'Add child' })).not.toBeInTheDocument()
  })

  it('calls onEdit with the node and closes the menu', () => {
    const onEdit = vi.fn()
    const node = makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={node}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={onEdit}
          onDelete={noop}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('ManagedCat'))
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(onEdit).toHaveBeenCalledTimes(1)
    expect(onEdit).toHaveBeenCalledWith(node)
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
  })

  it('calls onDelete with the node and closes the menu', () => {
    const onDelete = vi.fn()
    const node = makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={node}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={onDelete}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('ManagedCat'))
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(onDelete).toHaveBeenCalledTimes(1)
    expect(onDelete).toHaveBeenCalledWith(node)
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  })

  it('closes the menu when the trigger is clicked again without firing an action', () => {
    const onAddChild = vi.fn()
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })}
          depth={0}
          canManage={true}
          onAddChild={onAddChild}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('ManagedCat'))
    fireEvent.click(menuTriggerFor('ManagedCat'))

    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
    expect(onAddChild).not.toHaveBeenCalled()
    expect(onEdit).not.toHaveBeenCalled()
    expect(onDelete).not.toHaveBeenCalled()
  })

  it('closes the menu when the backdrop behind it is clicked', () => {
    const onEdit = vi.fn()
    const Wrapper = createWrapper()
    const { container } = render(
      <Wrapper>
        <CategoryTreeNode
          node={makeNode({ id: 'managed', name: 'ManagedCat', sort_order: 0 })}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={onEdit}
          onDelete={noop}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('ManagedCat'))
    // lucide icons also carry aria-hidden, so target the fixed-position backdrop specifically
    const backdrop = container.querySelector('div[aria-hidden="true"].fixed')
    expect(backdrop).not.toBeNull()
    fireEvent.click(backdrop as HTMLElement)

    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
    expect(onEdit).not.toHaveBeenCalled()
  })

  it('passes action callbacks to nested children, each bound to its own node', () => {
    const onEdit = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTreeNode
          node={root}
          depth={0}
          canManage={true}
          onAddChild={noop}
          onEdit={onEdit}
          onDelete={noop}
        />
      </Wrapper>,
    )

    fireEvent.click(menuTriggerFor('GrandChild'))
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(onEdit).toHaveBeenCalledWith(grandChild)
  })
})
