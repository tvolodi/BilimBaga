import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@/i18n'
import { StatsSummaryRow } from '../StatsSummaryRow'

describe('StatsSummaryRow', () => {
  it('renders all 4 stat cards', () => {
    render(
      <StatsSummaryRow
        avgScore={0.75}
        medianScore={0.78}
        totalAttempts={120}
        uniqueParticipants={95}
      />,
    )
    expect(screen.getByText('Average Score')).toBeInTheDocument()
    expect(screen.getByText('Median Score')).toBeInTheDocument()
    expect(screen.getByText('Total Attempts')).toBeInTheDocument()
    expect(screen.getByText('Unique Participants')).toBeInTheDocument()
  })

  it('renders numeric values correctly', () => {
    render(
      <StatsSummaryRow
        avgScore={0.75}
        medianScore={0.78}
        totalAttempts={120}
        uniqueParticipants={95}
      />,
    )
    expect(screen.getByText('75.0%')).toBeInTheDocument()
    expect(screen.getByText('78.0%')).toBeInTheDocument()
    expect(screen.getByText('120')).toBeInTheDocument()
    expect(screen.getByText('95')).toBeInTheDocument()
  })

  it('renders dash for null scores', () => {
    render(
      <StatsSummaryRow
        avgScore={null}
        medianScore={null}
        totalAttempts={0}
        uniqueParticipants={0}
      />,
    )
    // Should render "—" for null scores
    const dashes = screen.getAllByText('—')
    expect(dashes.length).toBeGreaterThanOrEqual(2)
  })

  it('renders 4 card elements', () => {
    const { container } = render(
      <StatsSummaryRow
        avgScore={0.5}
        medianScore={0.5}
        totalAttempts={10}
        uniqueParticipants={10}
      />,
    )
    // Grid has 4 children
    const grid = container.firstChild as HTMLElement
    expect(grid.children).toHaveLength(4)
  })
})
