import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@/i18n'
import { StatsSummaryRow } from '../StatsSummaryRow'

describe('StatsSummaryRow', () => {
  it('renders all 4 stat cards', () => {
    render(
      <StatsSummaryRow
        avgScore={75.0}
        medianScore={78.0}
        totalAttempts={120}
        uniqueParticipants={95}
      />,
    )
    expect(screen.getByText('Average Score')).toBeInTheDocument()
    expect(screen.getByText('Median Score')).toBeInTheDocument()
    expect(screen.getByText('Total Attempts')).toBeInTheDocument()
    expect(screen.getByText('Unique Participants')).toBeInTheDocument()
  })

  // avgScore and medianScore come from the API already as percentage values (e.g. 75.0 = 75.0%)
  // ISS-049: removing the ×100 multiplication that was inflating displayed values
  it('renders numeric values correctly', () => {
    render(
      <StatsSummaryRow
        avgScore={75.0}
        medianScore={78.0}
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
        avgScore={50.0}
        medianScore={50.0}
        totalAttempts={10}
        uniqueParticipants={10}
      />,
    )
    // Grid has 4 children
    const grid = container.firstChild as HTMLElement
    expect(grid.children).toHaveLength(4)
  })
})
