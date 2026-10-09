import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ErrorBoundary } from './ErrorBoundary'

vi.mock('i18next', () => ({
  default: { t: (key: string) => key },
}))

function Boom({ message }: { message: string }): never {
  throw new Error(message)
}

describe('ErrorBoundary', () => {
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders children when nothing throws', () => {
    render(
      <ErrorBoundary>
        <p>all good</p>
      </ErrorBoundary>,
    )
    expect(screen.getByText('all good')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('shows a recovery screen with the error message when a child throws', () => {
    render(
      <ErrorBoundary>
        <Boom message="kaboom" />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('common.errorBoundary.title')
    expect(screen.getByText('common.errorBoundary.message')).toBeInTheDocument()
    expect(screen.getByText('kaboom')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'common.errorBoundary.reload' })).toBeInTheDocument()
  })

  it('logs the caught error for observability', () => {
    render(
      <ErrorBoundary>
        <Boom message="logged" />
      </ErrorBoundary>,
    )
    expect(console.error).toHaveBeenCalledWith(
      '[ErrorBoundary] Uncaught render error:',
      expect.any(Error),
      expect.any(String),
    )
  })
})
