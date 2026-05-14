import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { LanguageSelector } from './LanguageSelector'
import '../../i18n'

const LOCALES = ['kk', 'ru', 'en']

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  localStorage.clear()
})

describe('LanguageSelector', () => {
  it('renders a select with the provided locales', () => {
    render(<LanguageSelector availableLocales={LOCALES} />)
    const select = screen.getByRole('combobox', { name: /language/i })
    expect(select).toBeInTheDocument()
    expect(screen.getByRole('option', { name: /Қазақша/i })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: /Русский/i })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: /English/i })).toBeInTheDocument()
  })

  it('persists language choice to localStorage on change', () => {
    render(<LanguageSelector availableLocales={LOCALES} />)
    const select = screen.getByRole('combobox', { name: /language/i })
    fireEvent.change(select, { target: { value: 'ru' } })
    expect(localStorage.getItem('i18n-lang')).toBe('ru')
  })

  it('persists kk choice to localStorage', () => {
    render(<LanguageSelector availableLocales={LOCALES} />)
    const select = screen.getByRole('combobox', { name: /language/i })
    fireEvent.change(select, { target: { value: 'kk' } })
    expect(localStorage.getItem('i18n-lang')).toBe('kk')
  })
})
