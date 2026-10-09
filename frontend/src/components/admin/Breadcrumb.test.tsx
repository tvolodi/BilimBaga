import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Breadcrumb } from './Breadcrumb'

const at = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Breadcrumb />
    </MemoryRouter>,
  )

describe('Breadcrumb', () => {
  it('renders nothing on the admin root', () => {
    const { container } = at('/admin')
    expect(container).toBeEmptyDOMElement()
  })

  it('links ancestors and renders the last crumb as plain text', () => {
    at('/admin/exams/new')
    expect(screen.getByRole('link', { name: 'Exams' }).getAttribute('href')).toBe('/admin/exams')
    expect(screen.getByText('New').tagName).toBe('SPAN')
    expect(screen.queryByRole('link', { name: 'New' })).toBeNull()
  })

  it('falls back to the raw segment for unknown routes', () => {
    at('/admin/users/abc-123')
    expect(screen.getByText('abc-123')).toBeTruthy()
  })
})
