// Regression test for ISS-046: SectionScores must not render entries with empty titles
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@/i18n'
import { SectionScores } from './SectionScores'
import type { SectionScore } from '@/api/sessions'

describe('SectionScores', () => {
  it('renders nothing when sections array is empty', () => {
    const { container } = render(<SectionScores sections={[]} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders nothing when all sections have empty titles (flat exam — ISS-046)', () => {
    const sections: SectionScore[] = [
      { section_id: 'rule-uuid-1', title: '', score_pct: 82.5 },
    ]
    const { container } = render(<SectionScores sections={sections} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders nothing when all sections have whitespace-only titles', () => {
    const sections: SectionScore[] = [
      { section_id: 'rule-uuid-1', title: '   ', score_pct: 50.0 },
    ]
    const { container } = render(<SectionScores sections={sections} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders section cards for valid (non-empty title) sections', () => {
    const sections: SectionScore[] = [
      { section_id: 'sect-1', title: 'Theory', score_pct: 90.0 },
      { section_id: 'sect-2', title: 'Practical', score_pct: 75.0 },
    ]
    render(<SectionScores sections={sections} />)
    expect(screen.getByText('Theory')).toBeInTheDocument()
    expect(screen.getByText('Practical')).toBeInTheDocument()
    expect(screen.getByText('90%')).toBeInTheDocument()
    expect(screen.getByText('75%')).toBeInTheDocument()
  })

  it('filters out empty-title entries and renders only valid ones', () => {
    const sections: SectionScore[] = [
      { section_id: 'rule-uuid-1', title: '', score_pct: 60.0 },
      { section_id: 'sect-1', title: 'Theory', score_pct: 90.0 },
    ]
    render(<SectionScores sections={sections} />)
    expect(screen.getByText('Theory')).toBeInTheDocument()
    // The spurious empty-title entry must not produce a score card with just a number
    expect(screen.queryByText('60%')).toBeNull()
  })
})
