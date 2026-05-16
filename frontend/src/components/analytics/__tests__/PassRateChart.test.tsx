import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@/i18n'
import { PassRateChart } from '../PassRateChart'

describe('PassRateChart', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <PassRateChart passRate={0.75} primaryColor="#6366f1" />,
    )
    expect(container.firstChild).toBeTruthy()
  })

  it('displays the pass rate percentage', () => {
    render(<PassRateChart passRate={0.75} primaryColor="#6366f1" />)
    expect(screen.getByText(/75\.0%/)).toBeInTheDocument()
  })

  it('renders with 100% pass rate', () => {
    render(<PassRateChart passRate={1.0} primaryColor="#6366f1" />)
    expect(screen.getByText(/100\.0%/)).toBeInTheDocument()
  })

  it('renders with 0% pass rate', () => {
    render(<PassRateChart passRate={0} primaryColor="#6366f1" />)
    expect(screen.getByText(/0\.0%/)).toBeInTheDocument()
  })

  it('renders recharts svg element', () => {
    const { container } = render(
      <PassRateChart passRate={0.6} primaryColor="#ff0000" />,
    )
    const svg = container.querySelector('svg')
    expect(svg).toBeTruthy()
  })
})
