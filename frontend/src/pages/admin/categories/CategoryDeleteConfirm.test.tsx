import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@/i18n'
import { CategoryDeleteConfirm } from './CategoryDeleteConfirm'

describe('CategoryDeleteConfirm', () => {
  it('renders the title and warning description when open', () => {
    render(
      <CategoryDeleteConfirm open onClose={vi.fn()} onConfirm={vi.fn()} isPending={false} />,
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Delete category')).toBeInTheDocument()
    expect(screen.getByText('This cannot be undone.')).toBeInTheDocument()
  })

  it('renders nothing when closed', () => {
    render(
      <CategoryDeleteConfirm open={false} onClose={vi.fn()} onConfirm={vi.fn()} isPending={false} />,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('calls onConfirm when Delete is clicked and does not close on its own', async () => {
    const onConfirm = vi.fn()
    const onClose = vi.fn()
    render(
      <CategoryDeleteConfirm open onClose={onClose} onConfirm={onConfirm} isPending={false} />,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('calls onClose when Cancel is clicked and not onConfirm', async () => {
    const onConfirm = vi.fn()
    const onClose = vi.fn()
    render(
      <CategoryDeleteConfirm open onClose={onClose} onConfirm={onConfirm} isPending={false} />,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('calls onClose when the dialog is dismissed with Escape', async () => {
    const onClose = vi.fn()
    render(
      <CategoryDeleteConfirm open onClose={onClose} onConfirm={vi.fn()} isPending={false} />,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('disables both buttons and shows a placeholder while the delete is pending', () => {
    render(
      <CategoryDeleteConfirm open onClose={vi.fn()} onConfirm={vi.fn()} isPending />,
    )
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    const confirm = screen.getByRole('button', { name: '…' })
    expect(confirm).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  })
})
