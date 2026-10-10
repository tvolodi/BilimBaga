import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'
import '@/i18n'
import { DesignGalleryPage } from './index'
import { GALLERY_SECTIONS } from './states'

// FR-BB320 AC-9: the gallery shows every section, can show a section in a .dark wrapper, and opens a
// dialog on demand.
describe('DesignGalleryPage (FR-BB320 AC-9)', () => {
  it('renders one section per gallery component', () => {
    render(<DesignGalleryPage />)
    for (const section of GALLERY_SECTIONS) {
      expect(screen.getByTestId(`gallery-section-${section.component}`)).toBeInTheDocument()
    }
  })

  it('shows a section inside a .dark wrapper when Dark is pressed, and back to light', () => {
    render(<DesignGalleryPage />)
    const section = screen.getByTestId('gallery-section-button')
    const states = within(section).getByTestId('gallery-states-button')
    expect(states.classList.contains('dark')).toBe(false)

    fireEvent.click(within(section).getByRole('button', { name: 'Dark' }))
    expect(states.classList.contains('dark')).toBe(true)

    fireEvent.click(within(section).getByRole('button', { name: 'Light' }))
    expect(states.classList.contains('dark')).toBe(false)
  })

  it('opens the dialog on demand and closes it again', () => {
    render(<DesignGalleryPage />)
    const section = screen.getByTestId('gallery-section-dialog')
    // The open Popover is also role=dialog, so the dialog is found by its title.
    expect(screen.queryByRole('dialog', { name: 'Dialog title' })).toBeNull()

    fireEvent.click(within(section).getByRole('button', { name: 'Open' }))
    expect(screen.getByRole('dialog', { name: 'Dialog title' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog', { name: 'Dialog title' })).toBeNull()
  })
})
