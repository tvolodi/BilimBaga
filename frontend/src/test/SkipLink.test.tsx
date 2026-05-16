import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { SkipLink } from '@/components/SkipLink'

// i18next is not initialised in unit tests; the component will fall back to the key
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

describe('SkipLink', () => {
  it('renders an anchor element', () => {
    render(<SkipLink />)
    const link = screen.getByRole('link')
    expect(link).toBeInTheDocument()
  })

  it('has href pointing to #main-content', () => {
    render(<SkipLink />)
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '#main-content')
  })

  it('renders using the i18n key for its label', () => {
    render(<SkipLink />)
    // When t() returns the key, the text should be the i18n key
    expect(screen.getByText('common.skipToMain')).toBeInTheDocument()
  })

  it('is visually hidden by default (sr-only class)', () => {
    render(<SkipLink />)
    const link = screen.getByRole('link')
    expect(link.className).toMatch(/sr-only/)
  })
})
