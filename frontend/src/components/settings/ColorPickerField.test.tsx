import { useState } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@/i18n'
import { ColorPickerField, contrastRatio } from './ColorPickerField'

function Harness({ onChange }: { onChange: (v: string) => void }) {
  const [value, setValue] = useState('#000000')
  return (
    <ColorPickerField
      label="Primary"
      value={value}
      onChange={(v) => {
        setValue(v)
        onChange(v)
      }}
      contrastAgainst="#ffffff"
    />
  )
}

describe('contrastRatio', () => {
  it('returns 21 for black against white', () => {
    expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 1)
  })

  it('returns 1 for the same color', () => {
    expect(contrastRatio('#0ea5e9', '#0ea5e9')).toBeCloseTo(1, 5)
  })

  it('produces a value below 4.5 for sky blue on white (fail)', () => {
    expect(contrastRatio('#0ea5e9', '#ffffff')).toBeLessThan(4.5)
  })

  it('produces a value above 4.5 for dark blue on white (pass)', () => {
    expect(contrastRatio('#003366', '#ffffff')).toBeGreaterThan(4.5)
  })
})

describe('ColorPickerField', () => {
  it('renders both a color input and a hex text input', () => {
    render(
      <ColorPickerField
        label="Primary"
        value="#0ea5e9"
        onChange={() => {}}
        contrastAgainst="#ffffff"
      />,
    )
    expect(screen.getByLabelText('Primary')).toBeInTheDocument()
    expect(screen.getByLabelText('Primary hex')).toBeInTheDocument()
  })

  it('shows a contrast warning when ratio is below 4.5:1', () => {
    render(
      <ColorPickerField
        label="Primary"
        value="#ffff00"
        onChange={() => {}}
        contrastAgainst="#ffffff"
      />,
    )
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('does not show contrast warning when ratio is at or above 4.5:1', () => {
    render(
      <ColorPickerField
        label="Primary"
        value="#003366"
        onChange={() => {}}
        contrastAgainst="#ffffff"
      />,
    )
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('emits a normalised hex when the user pastes a valid hex', async () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    const hexInput = screen.getByLabelText('Primary hex')
    await userEvent.clear(hexInput)
    await userEvent.click(hexInput)
    await userEvent.paste('#abcdef')
    expect(onChange).toHaveBeenLastCalledWith('#abcdef')
  })

  it('emits a normalised hex when a 3-char shorthand is entered', async () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    const hexInput = screen.getByLabelText('Primary hex')
    await userEvent.clear(hexInput)
    await userEvent.click(hexInput)
    await userEvent.paste('#abc')
    expect(onChange).toHaveBeenLastCalledWith('#aabbcc')
  })

  // Hand-typed values: each keystroke must leave the field showing what the user typed, and the parent
  // must end up with the normalised hex that carries the leading #.
  it.each(['2E6DB4', '#2E6DB4'])('keeps a hand-typed %s intact and sends the normalised hex', async (typed) => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    const hexInput = screen.getByLabelText('Primary hex')
    await userEvent.clear(hexInput)
    await userEvent.type(hexInput, typed)
    expect(onChange).toHaveBeenLastCalledWith('#2e6db4')
    await userEvent.tab()
    expect(hexInput).toHaveValue('#2e6db4')
  })

  it('emits the native colour picker value unchanged', () => {
    const onChange = vi.fn()
    render(
      <ColorPickerField
        label="Primary"
        value="#0ea5e9"
        onChange={onChange}
        contrastAgainst="#ffffff"
      />,
    )
    fireEvent.change(screen.getByLabelText('Primary', { selector: 'input[type="color"]' }), {
      target: { value: '#123456' },
    })
    expect(onChange).toHaveBeenCalledWith('#123456')
  })

  it('normalises an uppercase 6-digit hex to lowercase', () => {
    const onChange = vi.fn()
    render(
      <ColorPickerField
        label="Primary"
        value="#0ea5e9"
        onChange={onChange}
        contrastAgainst="#ffffff"
      />,
    )
    fireEvent.change(screen.getByLabelText('Primary hex'), { target: { value: '#ABCDEF' } })
    expect(onChange).toHaveBeenLastCalledWith('#abcdef')
  })

  it('passes an incomplete hex through unchanged so the user can keep typing', () => {
    const onChange = vi.fn()
    render(
      <ColorPickerField
        label="Primary"
        value="#0ea5e9"
        onChange={onChange}
        contrastAgainst="#ffffff"
      />,
    )
    fireEvent.change(screen.getByLabelText('Primary hex'), { target: { value: '#12' } })
    expect(onChange).toHaveBeenLastCalledWith('#12')
  })

  it('falls back to a black swatch when the stored value is not a valid hex', () => {
    render(
      <ColorPickerField
        label="Primary"
        value="not-a-colour"
        onChange={() => {}}
        contrastAgainst="#ffffff"
      />,
    )
    const swatch = screen.getByLabelText('Primary', { selector: 'input[type="color"]' }) as HTMLInputElement
    expect(swatch.value).toBe('#000000')
    expect(screen.getByLabelText('Primary hex')).toHaveValue('not-a-colour')
  })

  it('reports the contrast ratio to two decimals in the warning', () => {
    render(
      <ColorPickerField
        label="Primary"
        value="#ffff00"
        onChange={() => {}}
        contrastAgainst="#ffffff"
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('ratio: 1.07:1')
  })

  it('accepts the 3-digit shorthand for the contrast background', () => {
    render(
      <ColorPickerField
        label="Primary"
        value="#000"
        onChange={() => {}}
        contrastAgainst="#fff"
      />,
    )
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

describe('contrastRatio shorthand', () => {
  it('treats #fff and #ffffff as the same colour', () => {
    expect(contrastRatio('#000', '#fff')).toBeCloseTo(21, 1)
  })
})
