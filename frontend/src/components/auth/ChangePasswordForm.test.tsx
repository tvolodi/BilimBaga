import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ChangePasswordForm } from './ChangePasswordForm'
import '../../i18n'

const noop = () => {}

describe('ChangePasswordForm', () => {
  it('renders all three password fields and submit button', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={false} error={null} />)
    expect(screen.getByLabelText('Current password')).toBeInTheDocument()
    expect(screen.getByLabelText('New password')).toBeInTheDocument()
    expect(screen.getByLabelText('Confirm new password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /change password/i })).toBeInTheDocument()
  })

  it('shows mismatch error when new and confirm passwords differ', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={false} error={null} />)

    fireEvent.change(screen.getByLabelText('Current password'), {
      target: { value: 'oldpass' },
    })
    fireEvent.change(screen.getByLabelText('New password'), {
      target: { value: 'newpass1' },
    })
    fireEvent.change(screen.getByLabelText('Confirm new password'), {
      target: { value: 'newpass2' },
    })
    fireEvent.click(screen.getByRole('button', { name: /change password/i }))

    expect(screen.getByRole('alert').textContent).toMatch(/do not match/i)
  })

  it('does not call onSubmit when passwords mismatch', () => {
    const onSubmit = vi.fn()
    render(<ChangePasswordForm onSubmit={onSubmit} isPending={false} error={null} />)

    fireEvent.change(screen.getByLabelText('New password'), {
      target: { value: 'newpass1' },
    })
    fireEvent.change(screen.getByLabelText('Confirm new password'), {
      target: { value: 'different' },
    })
    fireEvent.click(screen.getByRole('button', { name: /change password/i }))

    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('calls onSubmit with correct payload when passwords match', () => {
    const onSubmit = vi.fn()
    render(<ChangePasswordForm onSubmit={onSubmit} isPending={false} error={null} />)

    fireEvent.change(screen.getByLabelText('Current password'), {
      target: { value: 'oldpass' },
    })
    fireEvent.change(screen.getByLabelText('New password'), {
      target: { value: 'newpass' },
    })
    fireEvent.change(screen.getByLabelText('Confirm new password'), {
      target: { value: 'newpass' },
    })
    fireEvent.click(screen.getByRole('button', { name: /change password/i }))

    expect(onSubmit).toHaveBeenCalledWith({
      current_password: 'oldpass',
      new_password: 'newpass',
    })
  })

  it('shows INVALID_CREDENTIALS server error', () => {
    render(
      <ChangePasswordForm
        onSubmit={noop}
        isPending={false}
        error={{ code: 'INVALID_CREDENTIALS', message: 'wrong password' }}
      />,
    )
    expect(screen.getByRole('alert').textContent).toMatch(/current password is incorrect/i)
  })

  it('disables submit button while pending', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={true} error={null} />)
    const submitBtn = document.querySelector('button[type="submit"]') as HTMLButtonElement
    expect(submitBtn).toBeDisabled()
  })

  it('toggles current-password visibility when eye icon is clicked', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={false} error={null} />)
    const input = document.getElementById('current-password')!
    expect(input).toHaveAttribute('type', 'password')

    const toggleBtns = screen.getAllByRole('button', { name: /show password/i })
    fireEvent.click(toggleBtns[0])
    expect(input).toHaveAttribute('type', 'text')

    fireEvent.click(screen.getAllByRole('button', { name: /hide password/i })[0])
    expect(input).toHaveAttribute('type', 'password')
  })

  it('toggles new-password visibility when eye icon is clicked', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={false} error={null} />)
    const input = document.getElementById('new-password')!
    expect(input).toHaveAttribute('type', 'password')

    const toggleBtns = screen.getAllByRole('button', { name: /show password/i })
    fireEvent.click(toggleBtns[1])
    expect(input).toHaveAttribute('type', 'text')

    fireEvent.click(screen.getAllByRole('button', { name: /hide password/i })[0])
    expect(input).toHaveAttribute('type', 'password')
  })

  it('toggles confirm-password visibility when eye icon is clicked', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={false} error={null} />)
    const input = document.getElementById('confirm-password')!
    expect(input).toHaveAttribute('type', 'password')

    const toggleBtns = screen.getAllByRole('button', { name: /show password/i })
    fireEvent.click(toggleBtns[2])
    expect(input).toHaveAttribute('type', 'text')

    fireEvent.click(screen.getAllByRole('button', { name: /hide password/i })[0])
    expect(input).toHaveAttribute('type', 'password')
  })

  it('current-password field has autoComplete off to prevent stale autofill', () => {
    render(<ChangePasswordForm onSubmit={noop} isPending={false} error={null} />)
    expect(document.getElementById('current-password')).toHaveAttribute('autocomplete', 'off')
  })
})
