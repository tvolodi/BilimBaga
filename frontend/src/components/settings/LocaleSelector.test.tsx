import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18n from '@/i18n'
import { LocaleSelector } from './LocaleSelector'

const LOCALE_ERROR = 'Default language must be in the available languages list.'

function renderSelector(
  overrides: Partial<{
    availableLocales: string[]
    defaultLocale: string
    onAvailableChange: (locales: string[]) => void
    onDefaultChange: (locale: string) => void
  }> = {},
) {
  const props = {
    availableLocales: ['kk', 'ru', 'en'],
    defaultLocale: 'en',
    onAvailableChange: vi.fn(),
    onDefaultChange: vi.fn(),
    ...overrides,
  }
  render(<LocaleSelector {...props} />)
  return props
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

describe('LocaleSelector', () => {
  it('lists every supported language as a checkbox with its native name', () => {
    renderSelector()

    expect(screen.getByRole('checkbox', { name: 'Қазақша' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Русский' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'English' })).toBeInTheDocument()
  })

  it('checks exactly the languages that are currently available', () => {
    renderSelector({ availableLocales: ['ru', 'en'], defaultLocale: 'en' })

    expect(screen.getByRole('checkbox', { name: 'Қазақша' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Русский' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'English' })).toBeChecked()
  })

  it('adds an unchecked language to the available list', async () => {
    const user = userEvent.setup()
    const { onAvailableChange } = renderSelector({ availableLocales: ['en'], defaultLocale: 'en' })

    await user.click(screen.getByRole('checkbox', { name: 'Русский' }))

    expect(onAvailableChange).toHaveBeenCalledWith(['en', 'ru'])
  })

  it('removes a checked language from the available list', async () => {
    const user = userEvent.setup()
    const { onAvailableChange } = renderSelector({ availableLocales: ['kk', 'en'], defaultLocale: 'en' })

    await user.click(screen.getByRole('checkbox', { name: 'Қазақша' }))

    expect(onAvailableChange).toHaveBeenCalledWith(['en'])
  })

  it('does not show the default-language error while the default is available', () => {
    renderSelector({ availableLocales: ['kk', 'en'], defaultLocale: 'en' })

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('shows an error when the default language is not in the available list', () => {
    renderSelector({ availableLocales: ['kk', 'ru'], defaultLocale: 'en' })

    expect(screen.getByRole('alert')).toHaveTextContent(LOCALE_ERROR)
  })

  it('shows an error when no language is available at all', () => {
    renderSelector({ availableLocales: [], defaultLocale: 'ru' })

    expect(screen.getByRole('alert')).toHaveTextContent(LOCALE_ERROR)
    expect(screen.getByRole('checkbox', { name: 'Русский' })).not.toBeChecked()
  })

  it('offers every supported language in the default-language select with the current value selected', () => {
    renderSelector({ availableLocales: ['kk', 'ru', 'en'], defaultLocale: 'ru' })

    const select = screen.getByRole('combobox', { name: 'Default Language' }) as HTMLSelectElement
    expect(select.value).toBe('ru')
    const options = within(select).getAllByRole('option')
    expect(options.map((o) => o.textContent)).toEqual(['Қазақша', 'Русский', 'English'])
  })

  it('reports the newly chosen default language', async () => {
    const user = userEvent.setup()
    const { onDefaultChange } = renderSelector({ availableLocales: ['kk', 'ru', 'en'], defaultLocale: 'en' })

    await user.selectOptions(screen.getByRole('combobox', { name: 'Default Language' }), 'kk')

    expect(onDefaultChange).toHaveBeenCalledWith('kk')
  })
})
