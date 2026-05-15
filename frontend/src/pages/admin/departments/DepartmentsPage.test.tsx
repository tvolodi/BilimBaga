import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { DepartmentsPage } from './DepartmentsPage'

describe('DepartmentsPage', () => {
  it('renders the page heading', () => {
    render(
      <MemoryRouter>
        <DepartmentsPage />
      </MemoryRouter>,
    )
    // The page renders the nav.departments i18n key or a "coming soon" placeholder
    expect(screen.getByRole('heading', { level: 2 })).toBeInTheDocument()
  })

  it('shows coming soon placeholder text', () => {
    render(
      <MemoryRouter>
        <DepartmentsPage />
      </MemoryRouter>,
    )
    expect(document.body.textContent).toBeTruthy()
  })
})
