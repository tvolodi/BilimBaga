import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ComponentProps } from 'react'
import '@/i18n'
import { DepartmentTreeSelect } from './DepartmentTreeSelect'
import type { Department } from '@/api/departments'

const useDepartmentsMock = vi.fn()
const refetchMock = vi.fn()

vi.mock('@/api/departments', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/departments')>()),
  useDepartments: () => useDepartmentsMock(),
}))

function node(id: string, name: string, children: Department[] = []): Department {
  return {
    id,
    name,
    parent_id: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    children,
  }
}

const TREE: Department[] = [
  node('d-eng', 'Engineering', [
    node('d-back', 'Backend', [node('d-plat', 'Platform')]),
    node('d-front', 'Frontend'),
  ]),
  node('d-fin', 'Finance'),
]

function mockQuery(state: { data?: Department[]; isLoading?: boolean; isError?: boolean }) {
  useDepartmentsMock.mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: refetchMock,
    ...state,
  })
}

function renderSelect(props: Partial<ComponentProps<typeof DepartmentTreeSelect>> = {}) {
  const onChange = vi.fn()
  const utils = render(
    <DepartmentTreeSelect value={null} onChange={onChange} aria-label="Department" {...props} />,
  )
  return { onChange, ...utils }
}

async function openPopover(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('combobox', { name: 'Department' }))
}

async function openTree(user: ReturnType<typeof userEvent.setup>) {
  await openPopover(user)
  return screen.getByRole('tree')
}

function item(name: string) {
  return screen.getByRole('treeitem', { name })
}

function chevronOf(name: string) {
  const el = item(name)
  const chevron = el.querySelector('.lucide-chevron-right, .lucide-chevron-down')
  if (!chevron) throw new Error(`no chevron on ${name}`)
  return chevron as Element
}

beforeEach(() => {
  useDepartmentsMock.mockReset()
  refetchMock.mockReset()
  mockQuery({ data: TREE })
})

describe('DepartmentTreeSelect trigger', () => {
  it('shows the default placeholder when nothing is selected', () => {
    renderSelect()
    expect(screen.getByRole('combobox', { name: 'Department' })).toHaveTextContent('Select department')
  })

  it('shows a custom placeholder when nothing is selected', () => {
    renderSelect({ placeholder: 'Pick a unit' })
    expect(screen.getByRole('combobox', { name: 'Department' })).toHaveTextContent('Pick a unit')
  })

  it('shows the name of a nested selected department', () => {
    renderSelect({ value: 'd-plat' })
    expect(screen.getByRole('combobox', { name: 'Department' })).toHaveTextContent('Platform')
  })

  it('falls back to the placeholder when the value matches no department', () => {
    renderSelect({ value: 'missing-id', placeholder: 'Pick a unit' })
    expect(screen.getByRole('combobox', { name: 'Department' })).toHaveTextContent('Pick a unit')
  })

  it('passes aria-label to the combobox and to the tree', async () => {
    const user = userEvent.setup()
    renderSelect({ 'aria-label': 'Owning department' })
    await user.click(screen.getByRole('combobox', { name: 'Owning department' }))
    expect(screen.getByRole('tree', { name: 'Owning department' })).toBeInTheDocument()
  })

  it('does not open when disabled', async () => {
    const user = userEvent.setup()
    renderSelect({ disabled: true })
    const trigger = screen.getByRole('combobox', { name: 'Department' })
    expect(trigger).toBeDisabled()
    await user.click(trigger)
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()
  })
})

describe('DepartmentTreeSelect loading, empty and error states', () => {
  it('disables the trigger and shows a loading label while departments load', () => {
    mockQuery({ isLoading: true })
    renderSelect({ value: 'd-fin' })
    const trigger = screen.getByRole('combobox', { name: 'Department' })
    expect(trigger).toBeDisabled()
    expect(trigger).toHaveTextContent('Loading departments...')
    expect(trigger).not.toHaveTextContent('Finance')
  })

  it('shows skeleton rows when the query starts loading while the tree is open', async () => {
    const user = userEvent.setup()
    const { rerender } = renderSelect()
    await openTree(user)
    expect(document.querySelectorAll('.animate-pulse')).toHaveLength(0)

    mockQuery({ isLoading: true })
    rerender(<DepartmentTreeSelect value={null} onChange={vi.fn()} aria-label="Department" />)

    expect(document.querySelectorAll('.animate-pulse')).toHaveLength(3)
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()
  })

  it('shows the empty message when there are no departments', async () => {
    const user = userEvent.setup()
    mockQuery({ data: [] })
    renderSelect()
    await openPopover(user)
    expect(screen.getByText('No departments found')).toBeInTheDocument()
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()
  })

  it('shows an error message and retries the query from the retry button', async () => {
    const user = userEvent.setup()
    mockQuery({ isError: true })
    renderSelect()
    await openPopover(user)
    expect(screen.getByText('Failed to load departments')).toBeInTheDocument()
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(refetchMock).toHaveBeenCalledTimes(1)
  })
})

describe('DepartmentTreeSelect tree rendering and expansion', () => {
  it('lists top-level departments with children collapsed', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'false')
    expect(item('Finance')).not.toHaveAttribute('aria-expanded')
    expect(screen.queryByRole('treeitem', { name: 'Backend' })).not.toBeInTheDocument()
    expect(screen.queryByRole('treeitem', { name: 'Frontend' })).not.toBeInTheDocument()
  })

  it('expands a node through its chevron without selecting it', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)

    await user.click(chevronOf('Engineering'))

    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'true')
    const group = screen.getByRole('group')
    expect(within(group).getByRole('treeitem', { name: 'Backend' })).toBeInTheDocument()
    expect(within(group).getByRole('treeitem', { name: 'Frontend' })).toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it('collapses an expanded node again through its chevron', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)
    await user.click(chevronOf('Engineering'))
    await user.click(chevronOf('Engineering'))

    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('group')).not.toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it('expands with ArrowRight and collapses with ArrowLeft', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    item('Engineering').focus()
    await user.keyboard('{ArrowRight}')
    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'true')

    await user.keyboard('{ArrowRight}')
    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'true')

    await user.keyboard('{ArrowLeft}')
    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'false')

    await user.keyboard('{ArrowLeft}')
    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'false')
  })

  it('ignores ArrowLeft and ArrowRight on a leaf department', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    item('Finance').focus()
    await user.keyboard('{ArrowRight}')
    await user.keyboard('{ArrowLeft}')

    expect(item('Finance')).not.toHaveAttribute('aria-expanded')
    expect(item('Finance')).toHaveFocus()
  })

  it('renders nested levels inside their parent group', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)
    await user.click(chevronOf('Engineering'))
    await user.click(chevronOf('Backend'))

    const groups = screen.getAllByRole('group')
    expect(groups).toHaveLength(2)
    expect(within(groups[1]).getByRole('treeitem', { name: 'Platform' })).toBeInTheDocument()
    expect(item('Platform')).not.toHaveAttribute('aria-expanded')
  })
})

describe('DepartmentTreeSelect selection', () => {
  it('calls onChange with the clicked department and closes the tree', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)

    await user.click(item('Finance'))

    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith('d-fin')
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Department' })).toHaveAttribute('aria-expanded', 'false')
  })

  it('selects a nested department that was revealed by expanding its parent', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)
    await user.click(chevronOf('Engineering'))
    await user.click(item('Frontend'))

    expect(onChange).toHaveBeenCalledWith('d-front')
  })

  it('selects a parent department without selecting its children', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)
    await user.click(item('Engineering'))

    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith('d-eng')
  })

  it('selects the focused department with Enter and Space', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)

    item('Finance').focus()
    await user.keyboard('{Enter}')
    expect(onChange).toHaveBeenLastCalledWith('d-fin')

    await openTree(user)
    item('Finance').focus()
    await user.keyboard(' ')
    expect(onChange).toHaveBeenLastCalledWith('d-fin')
    expect(onChange).toHaveBeenCalledTimes(2)
  })

  it('marks the current value as aria-selected', async () => {
    const user = userEvent.setup()
    renderSelect({ value: 'd-fin' })
    await openTree(user)

    expect(item('Finance')).toHaveAttribute('aria-selected', 'true')
    expect(item('Engineering')).toHaveAttribute('aria-selected', 'false')
  })
})

describe('DepartmentTreeSelect clear button', () => {
  it('clears the selection without opening the tree', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect({ value: 'd-fin' })

    await user.click(screen.getByLabelText('Clear selection'))

    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith(null)
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()
  })

  it('is hidden when nothing is selected', () => {
    renderSelect({ value: null })
    expect(screen.queryByLabelText('Clear selection')).not.toBeInTheDocument()
  })

  it('is hidden when clearable is false', () => {
    renderSelect({ value: 'd-fin', clearable: false })
    expect(screen.queryByLabelText('Clear selection')).not.toBeInTheDocument()
  })

  it('is hidden when the control is disabled', () => {
    renderSelect({ value: 'd-fin', disabled: true })
    expect(screen.queryByLabelText('Clear selection')).not.toBeInTheDocument()
  })
})

describe('DepartmentTreeSelect keyboard focus', () => {
  it('moves focus down and up through visible departments and stops at both ends', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)
    await user.click(chevronOf('Engineering'))

    item('Engineering').focus()
    await user.keyboard('{ArrowDown}')
    expect(item('Backend')).toHaveFocus()

    await user.keyboard('{ArrowDown}')
    expect(item('Frontend')).toHaveFocus()

    await user.keyboard('{ArrowDown}')
    expect(item('Finance')).toHaveFocus()

    await user.keyboard('{ArrowDown}')
    expect(item('Finance')).toHaveFocus()

    await user.keyboard('{ArrowUp}')
    expect(item('Frontend')).toHaveFocus()

    await user.keyboard('{ArrowUp}')
    await user.keyboard('{ArrowUp}')
    expect(item('Engineering')).toHaveFocus()
  })

  it('skips collapsed children when moving focus', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    item('Engineering').focus()
    await user.keyboard('{ArrowDown}')
    expect(item('Finance')).toHaveFocus()
  })

  it('moves the roving tab stop to the focused department and keeps exactly one tab stop on blur', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    expect(item('Engineering')).toHaveAttribute('tabindex', '0')
    act(() => item('Engineering').focus())
    expect(item('Engineering')).toHaveAttribute('tabindex', '0')
    expect(item('Engineering').className).toMatch(/ring-2/)

    act(() => item('Finance').focus())
    expect(item('Finance')).toHaveAttribute('tabindex', '0')
    expect(item('Engineering')).toHaveAttribute('tabindex', '-1')

    act(() => item('Finance').blur())
    expect(item('Engineering')).toHaveAttribute('tabindex', '0')
    expect(item('Finance')).toHaveAttribute('tabindex', '-1')
  })

  it('gives exactly one tab stop when the tree opens, on the first department (#360)', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    const tabStops = screen.getAllByRole('treeitem').filter((el) => el.getAttribute('tabindex') === '0')
    expect(tabStops).toHaveLength(1)
    expect(tabStops[0]).toBe(item('Engineering'))
  })

  it('puts the tab stop on the selected department when it is visible (#360)', async () => {
    const user = userEvent.setup()
    renderSelect({ value: 'd-fin' })
    await openTree(user)

    expect(item('Finance')).toHaveAttribute('tabindex', '0')
    expect(item('Engineering')).toHaveAttribute('tabindex', '-1')
  })

  it('lets Tab from the search input reach the tree tab stop (#360)', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    expect(screen.getByRole('textbox', { name: 'Search departments...' })).toHaveFocus()
    await user.tab()
    expect(item('Engineering')).toHaveFocus()
  })
})

describe('DepartmentTreeSelect search', () => {
  it('shows only matching departments and expands the ancestors of a match', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    await user.type(screen.getByRole('textbox', { name: 'Search departments...' }), 'platform')

    expect(item('Engineering')).toHaveAttribute('aria-expanded', 'true')
    expect(item('Backend')).toHaveAttribute('aria-expanded', 'true')
    expect(item('Platform')).toBeInTheDocument()
    expect(screen.queryByRole('treeitem', { name: 'Frontend' })).not.toBeInTheDocument()
    expect(screen.queryByRole('treeitem', { name: 'Finance' })).not.toBeInTheDocument()
  })

  it('matches names case-insensitively', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    await user.type(screen.getByRole('textbox', { name: 'Search departments...' }), 'FIN')

    expect(item('Finance')).toBeInTheDocument()
    expect(screen.queryByRole('treeitem', { name: 'Engineering' })).not.toBeInTheDocument()
  })

  it('renders an empty tree when nothing matches the query', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)

    await user.type(screen.getByRole('textbox', { name: 'Search departments...' }), 'zzz')

    expect(screen.queryAllByRole('treeitem')).toHaveLength(0)
    expect(screen.getByRole('tree')).toBeInTheDocument()
  })

  it('clears the query and the search filter when the popover is dismissed with Escape', async () => {
    const user = userEvent.setup()
    renderSelect()
    await openTree(user)
    await user.type(screen.getByRole('textbox', { name: 'Search departments...' }), 'fin')
    expect(screen.queryByRole('treeitem', { name: 'Engineering' })).not.toBeInTheDocument()

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()

    await openTree(user)
    expect(screen.getByRole('textbox', { name: 'Search departments...' })).toHaveValue('')
    expect(item('Engineering')).toBeInTheDocument()
  })

  it('clears the query after a department is selected', async () => {
    const user = userEvent.setup()
    const { onChange } = renderSelect()
    await openTree(user)
    await user.type(screen.getByRole('textbox', { name: 'Search departments...' }), 'fin')
    await user.click(item('Finance'))

    expect(onChange).toHaveBeenCalledWith('d-fin')
    await openTree(user)
    expect(screen.getByRole('textbox', { name: 'Search departments...' })).toHaveValue('')
  })
})
