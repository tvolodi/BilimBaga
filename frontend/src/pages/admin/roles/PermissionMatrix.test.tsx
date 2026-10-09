import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@/i18n'
import { PermissionMatrix } from './PermissionMatrix'
import type { CataloguePermission } from '@/api/roles'

const catalogue: CataloguePermission[] = [
  { id: 'p-users-read', resource: 'users', action: 'read' },
  { id: 'p-users-manage', resource: 'users', action: 'manage' },
  { id: 'p-exams-read', resource: 'exams', action: 'read' },
  { id: 'p-roles-read', resource: 'roles', action: 'read' },
  { id: 'p-roles-manage', resource: 'roles', action: 'manage' },
  { id: 'p-tenant-manage', resource: 'tenant', action: 'manage' },
]

describe('PermissionMatrix', () => {
  it('groups permissions by resource with a labelled checkbox per action', () => {
    render(<PermissionMatrix catalogue={catalogue} selected={new Set()} onChange={vi.fn()} />)
    expect(screen.getByRole('group', { name: 'Users' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Exams' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: /Users: View/ })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: /Users: Manage/ })).toBeInTheDocument()
  })

  it('calls onChange with the toggled id (check and uncheck)', async () => {
    const onChange = vi.fn()
    const { rerender } = render(<PermissionMatrix catalogue={catalogue} selected={new Set()} onChange={onChange} />)
    await userEvent.click(screen.getByRole('checkbox', { name: /Exams: View/ }))
    expect(onChange).toHaveBeenLastCalledWith(new Set(['p-exams-read']))

    rerender(<PermissionMatrix catalogue={catalogue} selected={new Set(['p-exams-read'])} onChange={onChange} />)
    await userEvent.click(screen.getByRole('checkbox', { name: /Exams: View/ }))
    expect(onChange).toHaveBeenLastCalledWith(new Set())
  })

  it('disables roles:read, roles:manage and tenant:manage so they cannot be selected', async () => {
    const onChange = vi.fn()
    render(<PermissionMatrix catalogue={catalogue} selected={new Set()} onChange={onChange} />)
    for (const id of ['perm-p-roles-read', 'perm-p-roles-manage', 'perm-p-tenant-manage']) {
      const box = document.getElementById(id) as HTMLInputElement
      expect(box).toBeDisabled()
      await userEvent.click(box)
    }
    expect(onChange).not.toHaveBeenCalled()
    // assignable ones stay enabled
    expect(document.getElementById('perm-p-users-read')).toBeEnabled()
  })

  it('is fully read-only for system roles but shows what is granted', () => {
    render(
      <PermissionMatrix catalogue={catalogue} selected={new Set(['p-users-read', 'p-roles-read'])} onChange={vi.fn()} readOnly />,
    )
    for (const box of screen.getAllByRole('checkbox')) expect(box).toBeDisabled()
    expect(document.getElementById('perm-p-users-read')).toBeChecked()
    expect(document.getElementById('perm-p-roles-read')).toBeChecked()
    expect(document.getElementById('perm-p-exams-read')).not.toBeChecked()
  })
})
