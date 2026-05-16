import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { ScoreDistributionChart } from '../ScoreDistributionChart'
import type { BucketCount } from '@/api/analytics'

const distribution: BucketCount[] = [
  { bucket: '0–10%', count: 2 },
  { bucket: '10–20%', count: 5 },
  { bucket: '20–30%', count: 8 },
  { bucket: '30–40%', count: 12 },
  { bucket: '40–50%', count: 15 },
  { bucket: '50–60%', count: 20 },
  { bucket: '60–70%', count: 18 },
  { bucket: '70–80%', count: 14 },
  { bucket: '80–90%', count: 9 },
  { bucket: '90–100%', count: 4 },
]

describe('ScoreDistributionChart', () => {
  it('renders without crashing', () => {
    const { container } = render(
      <ScoreDistributionChart distribution={distribution} primaryColor="#6366f1" />,
    )
    expect(container.firstChild).toBeTruthy()
  })

  it('renders with empty distribution', () => {
    const { container } = render(
      <ScoreDistributionChart distribution={[]} primaryColor="#6366f1" />,
    )
    expect(container.firstChild).toBeTruthy()
  })

  it('renders with custom primary color', () => {
    const { container } = render(
      <ScoreDistributionChart distribution={distribution} primaryColor="#ff0000" />,
    )
    expect(container.firstChild).toBeTruthy()
  })

  it('renders recharts responsive container', () => {
    const { container } = render(
      <ScoreDistributionChart distribution={distribution} primaryColor="#6366f1" />,
    )
    // recharts renders a ResponsiveContainer div in jsdom
    const rc = container.querySelector('.recharts-responsive-container')
    expect(rc).toBeTruthy()
  })

  it('renders the recharts wrapper element', () => {
    const { container } = render(
      <ScoreDistributionChart distribution={distribution} primaryColor="#6366f1" />,
    )
    // In jsdom recharts renders a div but not the svg (no ResizeObserver)
    expect(container.querySelector('.recharts-responsive-container')).toBeTruthy()
  })
})
