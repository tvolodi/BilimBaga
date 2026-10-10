import { afterEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@/i18n'
import { ThemeProvider, THEME_STORAGE_KEY } from './ThemeProvider'
import { TenantLogo } from './TenantLogo'

afterEach(() => {
  window.localStorage.removeItem(THEME_STORAGE_KEY)
  document.documentElement.classList.remove('dark')
})

describe('TenantLogo (FR-BB321 AC-9)', () => {
  it('leaves the image unchanged in the light theme', () => {
    render(<TenantLogo appName="Acme" />)
    expect(screen.getByRole('img', { name: 'Acme' })).not.toHaveClass('bg-white')
  })

  it('puts the white plate on the image itself in the dark theme', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    render(
      <ThemeProvider switchEnabled>
        <TenantLogo appName="Acme" />
      </ThemeProvider>,
    )
    expect(screen.getByRole('img', { name: 'Acme' })).toHaveClass('bg-white', 'rounded-md', 'p-1')
  })

  it('keeps the text fallback as the next sibling of the image in the dark theme', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    render(
      <ThemeProvider switchEnabled>
        <TenantLogo appName="Acme" />
      </ThemeProvider>,
    )
    const image = screen.getByRole('img', { name: 'Acme' })
    expect(image.nextElementSibling).toHaveTextContent('Acme')
  })
})
