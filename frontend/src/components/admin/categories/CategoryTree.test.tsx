import { describe, it, expect, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { type ReactNode } from 'react'
import { type DragEndEvent } from '@dnd-kit/core'
import '@/i18n'
import { CategoryTree } from './CategoryTree'
import { type CategoryNode } from '@/api/categories'

// Captures the onDragEnd handler that CategoryTree hands to DndContext, so the
// reorder logic can be driven deterministically (jsdom has no layout, so a real
// pointer or keyboard drag cannot produce a meaningful collision result).
const dragEnd = vi.hoisted(() => ({
  handler: undefined as undefined | ((event: DragEndEvent) => void),
}))

vi.mock('@dnd-kit/core', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@dnd-kit/core')>()
  const { createElement } = await import('react')
  return {
    ...actual,
    DndContext: (props: Parameters<typeof actual.DndContext>[0]) => {
      dragEnd.handler = props.onDragEnd
      return createElement(actual.DndContext, props)
    },
  }
})

// Opens the actions menu of the row showing `name`. The trigger is the only
// small (h-6) button in a row; the drag handle and expand toggle have no such class.
function openMenu(name: string) {
  const row = screen.getByText(name).parentElement as HTMLElement
  const trigger = row.querySelector<HTMLButtonElement>('button.h-6')
  if (!trigger) throw new Error(`no actions trigger rendered for ${name}`)
  fireEvent.click(trigger)
}

function fireDragEnd(activeId: string, overId: string | null) {
  const handler = dragEnd.handler
  if (!handler) throw new Error('DndContext was not rendered with canManage=true')
  act(() => {
    handler({
      active: { id: activeId },
      over: overId === null ? null : { id: overId },
    } as unknown as DragEndEvent)
  })
}

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

// Three sibling nodes with mixed sort_order and name ordering
const nodeAlpha = makeNode({ id: 'cat-alpha', name: 'Alpha', sort_order: 10 })
const nodeZeta = makeNode({ id: 'cat-zeta', name: 'Zeta', sort_order: 5 })
const nodeMu = makeNode({ id: 'cat-mu', name: 'Mu', sort_order: 5 })   // same sort_order as Zeta

// --------------------------------------------------------------------------
// Tests: sort_order then name
// --------------------------------------------------------------------------

describe('CategoryTree — sort order', () => {
  it('renders children sorted by sort_order ascending, then name ascending as tie-breaker', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[nodeAlpha, nodeZeta, nodeMu]}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    // All three nodes must be in the document
    expect(screen.getByText('Alpha')).toBeInTheDocument()
    expect(screen.getByText('Zeta')).toBeInTheDocument()
    expect(screen.getByText('Mu')).toBeInTheDocument()

    // Verify DOM order: Mu (sort_order 5, 'M') < Zeta (sort_order 5, 'Z') < Alpha (sort_order 10)
    const all = screen.getAllByText(/Alpha|Zeta|Mu/)
    const names = all.map((el) => el.textContent)
    expect(names[0]).toBe('Mu')
    expect(names[1]).toBe('Zeta')
    expect(names[2]).toBe('Alpha')
  })

  it('renders a single node without crashing', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[makeNode({ id: 'solo', name: 'SoloCategory', sort_order: 0 })]}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )
    expect(screen.getByText('SoloCategory')).toBeInTheDocument()
  })

  it('renders an empty list without crashing', () => {
    const Wrapper = createWrapper()
    const { container } = render(
      <Wrapper>
        <CategoryTree
          nodes={[]}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )
    // Should render an empty div with no visible text nodes from category names
    expect(container.querySelectorAll('[data-testid]')).toHaveLength(0)
  })
})

// --------------------------------------------------------------------------
// Tests: drag handle visibility
// --------------------------------------------------------------------------

describe('CategoryTree — drag handle', () => {
  it('shows drag handle when canManage is true', () => {
    const Wrapper = createWrapper()
    const { container } = render(
      <Wrapper>
        <CategoryTree
          nodes={[makeNode({ id: 'n1', name: 'Node1', sort_order: 0 })]}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )
    // Drag handle button is rendered (GripVertical icon button)
    // The button has no accessible text so we check for its presence via the SVG
    const svgs = container.querySelectorAll('svg')
    expect(svgs.length).toBeGreaterThan(0)
  })
})

// --------------------------------------------------------------------------
// Tests: action callbacks
// --------------------------------------------------------------------------

describe('CategoryTree — callbacks', () => {
  it('calls onEdit with the node when the edit action is chosen', () => {
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    const onAddChild = vi.fn()
    const editable = makeNode({ id: 'editable', name: 'EditMe', sort_order: 0 })
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[editable]}
          canManage={true}
          onAddChild={onAddChild}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      </Wrapper>,
    )

    openMenu('EditMe')
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(onEdit).toHaveBeenCalledTimes(1)
    expect(onEdit).toHaveBeenCalledWith(editable)
    expect(onDelete).not.toHaveBeenCalled()
    expect(onAddChild).not.toHaveBeenCalled()
  })

  it('passes action callbacks down to nested children', () => {
    const onEdit = vi.fn()
    const child = makeNode({ id: 'kid', name: 'KidNode', sort_order: 0, parent_id: 'top' })
    const top = makeNode({ id: 'top', name: 'TopNode', sort_order: 0, children: [child] })
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[top]}
          canManage={true}
          onAddChild={noop}
          onEdit={onEdit}
          onDelete={noop}
        />
      </Wrapper>,
    )

    openMenu('KidNode')
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(onEdit).toHaveBeenCalledWith(child)
  })
})

// --------------------------------------------------------------------------
// Tests: rendering (nesting, depth, empty state, canManage variants)
// --------------------------------------------------------------------------

describe('CategoryTree — rendering', () => {
  it('renders nested children inside their parent when canManage is false', () => {
    const Wrapper = createWrapper()
    const grandchild = makeNode({ id: 'gc', name: 'GrandKid', sort_order: 0, parent_id: 'kid' })
    const kid = makeNode({ id: 'kid', name: 'Kid', sort_order: 0, parent_id: 'top', children: [grandchild] })
    const top = makeNode({ id: 'top', name: 'Top', sort_order: 0, children: [kid] })
    render(
      <Wrapper>
        <CategoryTree
          nodes={[top]}
          canManage={false}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )

    expect(screen.getByText('Top')).toBeInTheDocument()
    expect(screen.getByText('Kid')).toBeInTheDocument()
    expect(screen.getByText('GrandKid')).toBeInTheDocument()
    // Each nesting level is indented 20px further than its parent (depth * 20 + 8)
    expect(screen.getByText('Top').parentElement).toHaveStyle({ paddingLeft: '8px' })
    expect(screen.getByText('Kid').parentElement).toHaveStyle({ paddingLeft: '28px' })
    expect(screen.getByText('GrandKid').parentElement).toHaveStyle({ paddingLeft: '48px' })
  })

  it('applies the depth prop to the rows it renders', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[makeNode({ id: 'deep', name: 'DeepRow', sort_order: 0 })]}
          depth={2}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )
    expect(screen.getByText('DeepRow').parentElement).toHaveStyle({ paddingLeft: '48px' })
  })

  it('renders children in sorted order when canManage is true', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[nodeAlpha, nodeZeta, nodeMu]}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )
    const names = screen.getAllByText(/Alpha|Zeta|Mu/).map((el) => el.textContent)
    expect(names).toEqual(['Mu', 'Zeta', 'Alpha'])
  })

  it('renders no rows for an empty list when canManage is true', () => {
    const Wrapper = createWrapper()
    const { container } = render(
      <Wrapper>
        <CategoryTree
          nodes={[]}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
        />
      </Wrapper>,
    )
    expect(container.querySelectorAll('.group')).toHaveLength(0)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})

// --------------------------------------------------------------------------
// Tests: drag reorder (handleDragEnd)
// --------------------------------------------------------------------------

describe('CategoryTree — drag reorder', () => {
  // Sort orders 0 / 10 / 20 leave room between neighbours
  const a = makeNode({ id: 'a', name: 'Alpha', sort_order: 0 })
  const b = makeNode({ id: 'b', name: 'Bravo', sort_order: 10 })
  const c = makeNode({ id: 'c', name: 'Charlie', sort_order: 20 })

  function renderReorderable(
    nodes: CategoryNode[],
    onReorder?: (parentId: string | null, nodeId: string, newSortOrder: number) => void,
  ) {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={nodes}
          canManage={true}
          onAddChild={noop}
          onEdit={noop}
          onDelete={noop}
          onReorder={onReorder}
        />
      </Wrapper>,
    )
  }

  it('places a node after the last node as previous sort_order + 1', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('a', 'c')

    expect(onReorder).toHaveBeenCalledTimes(1)
    expect(onReorder).toHaveBeenCalledWith(null, 'a', 21)
  })

  it('places a node before the first node as next sort_order - 1', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('c', 'a')

    expect(onReorder).toHaveBeenCalledWith(null, 'c', -1)
  })

  it('places a node between two neighbours at the rounded average of their sort_orders', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('a', 'b')

    // a moves between b (10) and c (20): round((10 + 20) / 2) = 15
    expect(onReorder).toHaveBeenCalledWith(null, 'a', 15)
  })

  it('falls back to the target index when the averaged value collides with a neighbour', () => {
    const onReorder = vi.fn()
    const x = makeNode({ id: 'x', name: 'X', sort_order: 0 })
    const y = makeNode({ id: 'y', name: 'Y', sort_order: 1 })
    const z = makeNode({ id: 'z', name: 'Z', sort_order: 2 })
    renderReorderable([x, y, z], onReorder)

    fireDragEnd('x', 'y')

    // Average of y (1) and z (2) rounds to 2, which equals z's sort_order, so the target index (1) is used
    expect(onReorder).toHaveBeenCalledWith(null, 'x', 1)
  })

  it('uses the sorted order, not the input order, to resolve drag indices', () => {
    const onReorder = vi.fn()
    renderReorderable([c, a, b], onReorder)

    fireDragEnd('c', 'a')

    expect(onReorder).toHaveBeenCalledWith(null, 'c', -1)
  })

  it('reports the parent_id of the siblings as the parentId', () => {
    const onReorder = vi.fn()
    const sa = makeNode({ id: 'sa', name: 'SibA', sort_order: 0, parent_id: 'parent-1' })
    const sb = makeNode({ id: 'sb', name: 'SibB', sort_order: 10, parent_id: 'parent-1' })
    const sc = makeNode({ id: 'sc', name: 'SibC', sort_order: 20, parent_id: 'parent-1' })
    renderReorderable([sa, sb, sc], onReorder)

    fireDragEnd('sa', 'sc')

    expect(onReorder).toHaveBeenCalledWith('parent-1', 'sa', 21)
  })

  it('does nothing when the node is dropped outside the list', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('a', null)

    expect(onReorder).not.toHaveBeenCalled()
  })

  it('does nothing when the node is dropped on itself', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('b', 'b')

    expect(onReorder).not.toHaveBeenCalled()
  })

  it('does nothing when the dragged id is not among the siblings', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('ghost', 'a')

    expect(onReorder).not.toHaveBeenCalled()
  })

  it('does nothing when the drop target id is not among the siblings', () => {
    const onReorder = vi.fn()
    renderReorderable([a, b, c], onReorder)

    fireDragEnd('a', 'ghost')

    expect(onReorder).not.toHaveBeenCalled()
  })

  it('does not throw on drop when no onReorder callback is provided', () => {
    renderReorderable([a, b, c])

    expect(() => fireDragEnd('a', 'c')).not.toThrow()
  })
})

// --------------------------------------------------------------------------
// Tests: parent picker exclusion (via getDescendantIds + flattenCategories)
// --------------------------------------------------------------------------

describe('CategoryTree — parent picker exclusion logic', () => {
  it('getDescendantIds returns all nested descendant IDs', async () => {
    const { getDescendantIds } = await import('@/api/categories')

    const grandchild = makeNode({ id: 'gc-1', name: 'Grandchild', sort_order: 0 })
    const child = makeNode({ id: 'c-1', name: 'Child', sort_order: 0, children: [grandchild] })
    const parent = makeNode({ id: 'p-1', name: 'Parent', sort_order: 0, children: [child] })

    const ids = getDescendantIds(parent)
    expect(ids).toContain('c-1')
    expect(ids).toContain('gc-1')
    expect(ids).not.toContain('p-1')
  })

  it('flattenCategories returns all nodes in DFS order', async () => {
    const { flattenCategories } = await import('@/api/categories')

    const child1 = makeNode({ id: 'child-1', name: 'Child1', sort_order: 0 })
    const child2 = makeNode({ id: 'child-2', name: 'Child2', sort_order: 1 })
    const root = makeNode({ id: 'root', name: 'Root', sort_order: 0, children: [child1, child2] })

    const flat = flattenCategories([root])
    const ids = flat.map((n) => n.id)
    expect(ids).toEqual(['root', 'child-1', 'child-2'])
  })

  it('parent picker excludes self and all descendants from options', async () => {
    const { flattenCategories, getDescendantIds } = await import('@/api/categories')

    const grandchild = makeNode({ id: 'gc', name: 'Grandchild', sort_order: 0 })
    const child = makeNode({ id: 'child', name: 'Child', sort_order: 0, children: [grandchild] })
    const sibling = makeNode({ id: 'sibling', name: 'Sibling', sort_order: 1 })
    const editingNode = makeNode({
      id: 'editing',
      name: 'Editing',
      sort_order: 0,
      children: [child],
    })
    const tree = [editingNode, sibling]

    const flat = flattenCategories(tree)
    const excluded = new Set([editingNode.id, ...getDescendantIds(editingNode)])
    const parentOptions = flat.filter((n) => !excluded.has(n.id))

    const optionIds = parentOptions.map((n) => n.id)
    // editingNode, child, grandchild all excluded
    expect(optionIds).not.toContain('editing')
    expect(optionIds).not.toContain('child')
    expect(optionIds).not.toContain('gc')
    // sibling is valid
    expect(optionIds).toContain('sibling')
  })
})
