import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { LoginForm } from './LoginForm'
import '../../i18n'

const noop = () => {}

describe('LoginForm', () => {
  it('renders email, password fields and submit button', () => {
    render(<LoginForm onSubmit={noop} isPending={false} error={null} />)
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /email/i })).toBeInTheDocument()
    expect(document.getElementById('password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('calls onSubmit with email and password values', () => {
    const onSubmit = vi.fn()
    render(<LoginForm onSubmit={onSubmit} isPending={false} error={null} />)

    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: 'user@example.com' },
    })
    fireEvent.change(document.getElementById('password')!, {
      target: { value: 'secret123' },
    })
    fireEvent.click(screen.getByRole('button', { name: /sign in/i }))

    expect(onSubmit).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret123',
    })
  })

  it('shows INVALID_CREDENTIALS error without clearing the email field', () => {
    render(
      <LoginForm
        onSubmit={noop}
        isPending={false}
        error={{ code: 'INVALID_CREDENTIALS', message: 'invalid email or password' }}
      />,
    )

    const emailInput = screen.getByLabelText(/email/i)
    fireEvent.change(emailInput, { target: { value: 'user@example.com' } })

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('alert').textContent).toMatch(/incorrect email or password/i)
    // Email field value preserved
    expect(emailInput).toHaveValue('user@example.com')
  })

  it('shows ACCOUNT_LOCKED error with a formatted timestamp', () => {
    render(
      <LoginForm
        onSubmit={noop}
        isPending={false}
        error={{
          code: 'ACCOUNT_LOCKED',
          message: 'account locked, try again after 2099-06-15T10:30:00Z',
        }}
      />,
    )

    expect(screen.getByRole('alert')).toBeInTheDocument()
    // Should contain formatted date, not raw UTC
    expect(screen.getByRole('alert').textContent).not.toMatch(/2099-06-15T10:30:00Z/)
  })

  it('shows network error for unknown error codes', () => {
    render(
      <LoginForm
        onSubmit={noop}
        isPending={false}
        error={{ code: 'INTERNAL_ERROR', message: 'server exploded' }}
      />,
    )

    expect(screen.getByRole('alert').textContent).toMatch(/service unavailable/i)
  })

  it('disables submit button while pending', () => {
    render(<LoginForm onSubmit={noop} isPending={true} error={null} />)
    expect(screen.getByRole('button', { name: /…/ })).toBeDisabled()
  })

  it('toggles password visibility when eye icon is clicked', () => {
    render(<LoginForm onSubmit={noop} isPending={false} error={null} />)
    const passwordInput = document.getElementById('password')!
    expect(passwordInput).toHaveAttribute('type', 'password')

    const toggleBtn = screen.getByRole('button', { name: /show password/i })
    fireEvent.click(toggleBtn)
    expect(passwordInput).toHaveAttribute('type', 'text')

    fireEvent.click(screen.getByRole('button', { name: /hide password/i }))
    expect(passwordInput).toHaveAttribute('type', 'password')
  })
})
