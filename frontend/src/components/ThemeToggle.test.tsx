import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@/i18n'
import { ThemeToggle } from './ThemeToggle'
import type { ThemePreference } from './ThemeProvider'

const theme = vi.hoisted(() => ({
  preference: 'system' as ThemePreference,
  setPreference: vi.fn(),
}))

vi.mock('@/components/ThemeProvider', () => ({
  useTheme: () => ({
    preference: theme.preference,
    resolved: 'light',
    setPreference: theme.setPreference,
  }),
}))

beforeEach(() => {
  theme.preference = 'system'
  theme.setPreference.mockReset()
})

function radios() {
  return {
    light: screen.getByRole('radio', { name: 'Light' }),
    system: screen.getByRole('radio', { name: 'System' }),
    dark: screen.getByRole('radio', { name: 'Dark' }),
  }
}

describe('ThemeToggle', () => {
  it('renders a labelled radio group with Light, System and Dark in that order', () => {
    render(<ThemeToggle />)
    expect(screen.getByRole('radiogroup', { name: 'Theme' })).toBeInTheDocument()
    const names = screen.getAllByRole('radio').map((radio) => radio.textContent)
    expect(names).toEqual(['Light', 'System', 'Dark'])
  })

  it('marks the current preference as checked and gives it the only tab stop', () => {
    theme.preference = 'dark'
    render(<ThemeToggle />)
    const { light, system, dark } = radios()
    expect(dark).toHaveAttribute('aria-checked', 'true')
    expect(dark).toHaveAttribute('tabindex', '0')
    for (const radio of [light, system]) {
      expect(radio).toHaveAttribute('aria-checked', 'false')
      expect(radio).toHaveAttribute('tabindex', '-1')
    }
  })

  it('keeps the 44 px touch target below the 640 px breakpoint', () => {
    render(<ThemeToggle />)
    for (const radio of screen.getAllByRole('radio')) {
      expect(radio).toHaveClass('max-sm:min-h-11', 'max-sm:min-w-11')
    }
  })

  it('calls setPreference with the chosen value on click', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    await user.click(radios().dark)
    expect(theme.setPreference).toHaveBeenCalledWith('dark')
  })

  it('moves selection and focus to the next and previous radio with the arrow keys', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    const { light, system, dark } = radios()

    system.focus()
    await user.keyboard('{ArrowRight}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('dark')
    expect(dark).toHaveFocus()

    await user.keyboard('{ArrowLeft}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('system')
    expect(system).toHaveFocus()

    await user.keyboard('{ArrowLeft}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('light')
    expect(light).toHaveFocus()
  })

  it('wraps around at both ends and supports Up, Down, Home and End', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    const { light, dark } = radios()

    light.focus()
    await user.keyboard('{ArrowLeft}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('dark')
    expect(dark).toHaveFocus()

    await user.keyboard('{ArrowRight}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('light')
    expect(light).toHaveFocus()

    await user.keyboard('{ArrowDown}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('system')

    await user.keyboard('{End}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('dark')
    expect(dark).toHaveFocus()

    await user.keyboard('{Home}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('light')
    expect(light).toHaveFocus()

    await user.keyboard('{ArrowUp}')
    expect(theme.setPreference).toHaveBeenLastCalledWith('dark')
  })

  it('ignores other keys', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)
    radios().system.focus()
    await user.keyboard('a')
    expect(theme.setPreference).not.toHaveBeenCalled()
  })
})
