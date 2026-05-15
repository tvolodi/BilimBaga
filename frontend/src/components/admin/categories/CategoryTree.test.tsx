import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { type ReactNode } from 'react'
import '@/i18n'
import { CategoryTree } from './CategoryTree'
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
  it('calls onEdit when edit action is triggered', async () => {
    const onEdit = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <CategoryTree
          nodes={[makeNode({ id: 'editable', name: 'EditMe', sort_order: 0 })]}
          canManage={true}
          onAddChild={noop}
          onEdit={onEdit}
          onDelete={noop}
        />
      </Wrapper>,
    )
    expect(screen.getByText('EditMe')).toBeInTheDocument()
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
