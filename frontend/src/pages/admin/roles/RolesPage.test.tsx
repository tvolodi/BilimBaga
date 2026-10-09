import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@/i18n'
import { RolesPage } from './RolesPage'
import type { AdminRole, CataloguePermission } from '@/api/roles'

const hooks = vi.hoisted(() => ({
  roles: { data: [] as unknown[], isLoading: false, isError: false },
  catalogue: { data: [] as unknown[], isLoading: false, isError: false },
  create: { mutateAsync: vi.fn(), reset: vi.fn(), isPending: false },
  update: { mutateAsync: vi.fn(), reset: vi.fn(), isPending: false },
  del: { mutateAsync: vi.fn(), reset: vi.fn(), isPending: false },
}))

vi.mock('@/api/roles', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/roles')>()
  return {
    ...actual,
    useRolesAdmin: () => hooks.roles,
    usePermissionsCatalogue: () => hooks.catalogue,
    useCreateRole: () => hooks.create,
    useUpdateRole: () => hooks.update,
    useDeleteRole: () => hooks.del,
  }
})

const catalogue: CataloguePermission[] = [
  { id: 'p1', resource: 'users', action: 'read' },
  { id: 'p2', resource: 'exams', action: 'read' },
  { id: 'p3', resource: 'reports', action: 'read' },
  { id: 'p4', resource: 'tenant', action: 'manage' },
]

const system: AdminRole = {
  id: 'r-sys', name: 'examiner', description: 'Built in', is_system: true, user_count: 3,
  permissions: ['exams:read', 'reports:read'], created_at: '2026-01-01T00:00:00Z',
}
const custom: AdminRole = {
  id: 'r-qa', name: 'qa_reviewer', description: 'QA', is_system: false, user_count: 0,
  permissions: ['exams:read'], created_at: '2026-01-02T00:00:00Z',
}

function codedError(code: string, message = code, details?: Record<string, unknown>) {
  return Object.assign(new Error(message), { code, details })
}

beforeEach(() => {
  vi.clearAllMocks()
  hooks.roles = { data: [system, custom], isLoading: false, isError: false }
  hooks.catalogue = { data: catalogue, isLoading: false, isError: false }
  hooks.create.mutateAsync.mockResolvedValue({})
  hooks.update.mutateAsync.mockResolvedValue({})
  hooks.del.mutateAsync.mockResolvedValue(undefined)
})

describe('RolesPage list', () => {
  it('shows name, description, type badge, user and permission counts', () => {
    render(<RolesPage />)
    const sysRow = screen.getByRole('row', { name: /examiner/ })
    expect(within(sysRow).getByText('System')).toBeInTheDocument()
    expect(within(sysRow).getByText('3')).toBeInTheDocument()
    expect(within(sysRow).getByText('2')).toBeInTheDocument()
    const customRow = screen.getByRole('row', { name: /qa_reviewer/ })
    expect(within(customRow).getByText('Custom')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create role' })).toBeInTheDocument()
  })

  it('protects system rows: View instead of Edit and no Delete', () => {
    render(<RolesPage />)
    const sysRow = screen.getByRole('row', { name: /examiner/ })
    expect(within(sysRow).getByRole('button', { name: 'View role examiner' })).toBeInTheDocument()
    expect(within(sysRow).queryByRole('button', { name: /Edit role/ })).not.toBeInTheDocument()
    expect(within(sysRow).queryByRole('button', { name: /Delete role/ })).not.toBeInTheDocument()
    const customRow = screen.getByRole('row', { name: /qa_reviewer/ })
    expect(within(customRow).getByRole('button', { name: 'Edit role qa_reviewer' })).toBeInTheDocument()
    expect(within(customRow).getByRole('button', { name: 'Delete role qa_reviewer' })).toBeInTheDocument()
  })

  it('opens a read-only dialog for a system role', async () => {
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'View role examiner' }))
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByLabelText('Name')).toBeDisabled()
    expect(within(dialog).getByLabelText('Description')).toBeDisabled()
    for (const box of within(dialog).getAllByRole('checkbox')) expect(box).toBeDisabled()
    expect(within(dialog).getByLabelText(/Exams: View/)).toBeChecked()
    expect(within(dialog).queryByRole('button', { name: 'Save' })).not.toBeInTheDocument()
  })

  it('shows a translated load error', () => {
    hooks.roles = { data: [], isLoading: false, isError: true }
    render(<RolesPage />)
    expect(screen.getByRole('alert')).toHaveTextContent('Failed to load')
  })
})

describe('RolesPage create flow', () => {
  async function openCreate() {
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'Create role' }))
    return screen.getByRole('dialog')
  }

  it('submits name, description and the selected permission ids, then shows a toast', async () => {
    const dialog = await openCreate()
    await userEvent.type(within(dialog).getByLabelText('Name'), 'qa_lead')
    await userEvent.type(within(dialog).getByLabelText('Description'), 'Lead')
    await userEvent.click(within(dialog).getByLabelText(/Users: View/))
    await userEvent.click(within(dialog).getByLabelText(/Reports: View/))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    await waitFor(() =>
      expect(hooks.create.mutateAsync).toHaveBeenCalledWith({
        name: 'qa_lead',
        description: 'Lead',
        permissions: ['p1', 'p3'],
      }),
    )
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByRole('status')).toHaveTextContent('Role "qa_lead" created.')
  })

  it('does not select tenant:manage (not assignable)', async () => {
    const dialog = await openCreate()
    expect(within(dialog).getByLabelText(/Tenant: Manage/)).toBeDisabled()
  })

  it('validates the name client-side without calling the API', async () => {
    const dialog = await openCreate()
    await userEvent.type(within(dialog).getByLabelText('Name'), 'QA Reviewer')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(hooks.create.mutateAsync).not.toHaveBeenCalled()
    expect(within(dialog).getByRole('alert')).toHaveTextContent(/lowercase letters/)
  })

  it.each([
    ['ROLE_NAME_TAKEN', 'A role with this name already exists.'],
    ['VALIDATION_ERROR', 'The role data is invalid.'],
    ['SOMETHING_NEW', 'Something went wrong. Please try again.'],
  ])('maps server error %s to translated inline text and keeps the dialog open', async (code, text) => {
    hooks.create.mutateAsync.mockRejectedValue(codedError(code, 'raw english server text'))
    const dialog = await openCreate()
    await userEvent.type(within(dialog).getByLabelText('Name'), 'qa_lead')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(within(dialog).getByRole('alert')).toHaveTextContent(text))
    expect(within(dialog).queryByText('raw english server text')).not.toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})

describe('RolesPage edit flow', () => {
  it('keeps the name disabled, preselects held permissions and sends replace-semantics PUT', async () => {
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit role qa_reviewer' }))
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).getByLabelText('Name')).toBeDisabled()
    expect(within(dialog).getByLabelText('Name')).toHaveValue('qa_reviewer')
    expect(within(dialog).getByLabelText(/Exams: View/)).toBeChecked()
    await userEvent.click(within(dialog).getByLabelText(/Users: View/))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(hooks.update.mutateAsync).toHaveBeenCalledWith({
        id: 'r-qa',
        description: 'QA',
        permissions: expect.arrayContaining(['p1', 'p2']),
      }),
    )
    expect(hooks.update.mutateAsync.mock.calls[0][0].permissions).toHaveLength(2)
    expect(await screen.findByRole('status')).toHaveTextContent('Role "qa_reviewer" updated.')
  })

  it('shows ROLE_SYSTEM_IMMUTABLE translated', async () => {
    hooks.update.mutateAsync.mockRejectedValue(codedError('ROLE_SYSTEM_IMMUTABLE'))
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit role qa_reviewer' }))
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(within(dialog).getByRole('alert')).toHaveTextContent('System roles cannot be changed'))
  })
})

describe('RolesPage delete flow', () => {
  it('asks for confirmation, deletes, and shows a toast', async () => {
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete role qa_reviewer' }))
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveTextContent('Delete the role "qa_reviewer"?')
    expect(hooks.del.mutateAsync).not.toHaveBeenCalled()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(hooks.del.mutateAsync).toHaveBeenCalledWith('r-qa'))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByRole('status')).toHaveTextContent('Role "qa_reviewer" deleted.')
  })

  it('cancel does not delete', async () => {
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete role qa_reviewer' }))
    await userEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Cancel' }))
    expect(hooks.del.mutateAsync).not.toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('ROLE_IN_USE shows the translated message with the user count and keeps the role listed', async () => {
    hooks.del.mutateAsync.mockRejectedValue(codedError('ROLE_IN_USE', 'role is assigned to 7 user(s)', { count: 2 }))
    render(<RolesPage />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete role qa_reviewer' }))
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(within(dialog).getByRole('alert')).toHaveTextContent('(count: 2)'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('row', { name: /qa_reviewer/ })).toBeInTheDocument()
  })
})
