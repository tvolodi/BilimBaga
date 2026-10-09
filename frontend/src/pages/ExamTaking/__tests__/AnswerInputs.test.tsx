import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { LikertInput } from '../LikertInput'
import { SingleChoiceInput } from '../SingleChoiceInput'
import { ShortTextInput } from '../ShortTextInput'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

const options = [
  { id: 'a', text: 'Alpha' },
  { id: 'b', text: 'Beta' },
  { id: 'c', text: '' },
]

describe('LikertInput', () => {
  it('marks only the selected option as checked', () => {
    render(<LikertInput options={options} selected="b" onChange={() => {}} />)
    const radios = screen.getAllByRole('radio')
    expect(radios.map((r) => r.getAttribute('aria-checked'))).toEqual(['false', 'true', 'false'])
  })

  it('falls back to the 1-based index when an option has no text', () => {
    render(<LikertInput options={options} selected={null} onChange={() => {}} />)
    expect(screen.getByRole('radio', { name: '3' })).toBeInTheDocument()
  })

  it('reports the clicked option id', async () => {
    const onChange = vi.fn()
    render(<LikertInput options={options} selected={null} onChange={onChange} />)
    await userEvent.click(screen.getByRole('radio', { name: 'Alpha' }))
    expect(onChange).toHaveBeenCalledWith('a')
  })
})

describe('SingleChoiceInput', () => {
  it('checks the selected option and leaves the rest unchecked', () => {
    render(<SingleChoiceInput options={options} selected="a" onChange={() => {}} />)
    expect(screen.getByLabelText('Alpha')).toBeChecked()
    expect(screen.getByLabelText('Beta')).not.toBeChecked()
  })

  it('reports the chosen option id', async () => {
    const onChange = vi.fn()
    render(<SingleChoiceInput options={options} selected={null} onChange={onChange} />)
    await userEvent.click(screen.getByLabelText('Beta'))
    expect(onChange).toHaveBeenCalledWith('b')
  })

  it('groups radios under one name so only one can be selected', () => {
    render(<SingleChoiceInput options={options} selected={null} onChange={() => {}} />)
    const names = new Set(screen.getAllByRole('radio').map((r) => r.getAttribute('name')))
    expect(names.size).toBe(1)
  })
})

describe('ShortTextInput', () => {
  it('shows the current value and a localized placeholder', () => {
    render(<ShortTextInput value="hello" onChange={() => {}} />)
    const box = screen.getByRole('textbox')
    expect(box).toHaveValue('hello')
    expect(box).toHaveAttribute('placeholder', 'exam.taking.shortTextPlaceholder')
  })

  it('emits the typed text', async () => {
    const onChange = vi.fn()
    render(<ShortTextInput value="" onChange={onChange} />)
    await userEvent.type(screen.getByRole('textbox'), 'x')
    expect(onChange).toHaveBeenCalledWith('x')
  })
})
