import { useState } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
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
})
