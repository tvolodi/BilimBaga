import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { type ReactNode } from 'react'
import '@/i18n'
import { DepartmentDeleteConfirm } from './DepartmentDeleteConfirm'
import { type Department } from '@/api/departments'

const dept: Department = {
  id: 'dept-1',
  name: 'Engineering',
  parent_id: null,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
}

function createWrapper() {
  const queryClient = new QueryClient()
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('DepartmentDeleteConfirm', () => {
  it('renders department name in confirmation dialog', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={false} onClose={() => {}} onConfirm={() => {}} />
      </Wrapper>,
    )
    expect(screen.getByText('Engineering')).toBeInTheDocument()
  })

  it('renders the title and the irreversible-action warning', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={false} onClose={() => {}} onConfirm={() => {}} />
      </Wrapper>,
    )
    expect(screen.getByRole('dialog')).toHaveTextContent('Delete Department')
    expect(screen.getByRole('dialog')).toHaveTextContent('This action cannot be undone')
  })

  it('does not render when closed', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open={false} department={dept} isPending={false} onClose={() => {}} onConfirm={() => {}} />
      </Wrapper>,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows no department name when no department is selected', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={null} isPending={false} onClose={() => {}} onConfirm={() => {}} />
      </Wrapper>,
    )
    expect(screen.getByRole('dialog')).not.toHaveTextContent('Engineering')
    expect(screen.getByRole('button', { name: /delete/i })).toBeEnabled()
  })

  it('calls onConfirm when delete button is clicked', async () => {
    const onConfirm = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={false} onClose={() => {}} onConfirm={onConfirm} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /delete/i }))
    expect(onConfirm).toHaveBeenCalledOnce()
  })

  it('disables buttons and replaces the delete label while the delete is pending', () => {
    const onConfirm = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={true} onClose={() => {}} onConfirm={onConfirm} />
      </Wrapper>,
    )
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled()
    const pendingButton = screen.getByRole('button', { name: '…' })
    expect(pendingButton).toBeDisabled()
    expect(screen.queryByRole('button', { name: /delete/i })).not.toBeInTheDocument()
  })

  it('does not call onConfirm when the pending delete button is clicked', async () => {
    const onConfirm = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={true} onClose={() => {}} onConfirm={onConfirm} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: '…' }))
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('calls onClose when cancel is clicked', async () => {
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={false} onClose={onClose} onConfirm={() => {}} />
      </Wrapper>,
    )
    await userEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('calls onClose when the dialog is dismissed with the Escape key', async () => {
    const onClose = vi.fn()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={false} onClose={onClose} onConfirm={() => {}} />
      </Wrapper>,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
