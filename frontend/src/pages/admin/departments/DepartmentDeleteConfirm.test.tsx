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

  it('does not render when closed', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open={false} department={dept} isPending={false} onClose={() => {}} onConfirm={() => {}} />
      </Wrapper>,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
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

  it('disables buttons when isPending is true', () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <DepartmentDeleteConfirm open department={dept} isPending={true} onClose={() => {}} onConfirm={() => {}} />
      </Wrapper>,
    )
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled()
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
})
