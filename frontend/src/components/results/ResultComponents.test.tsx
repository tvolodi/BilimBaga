import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import '@/i18n'
import { PassFailBanner } from './PassFailBanner'
import { ScoreDial } from './ScoreDial'
import { TimeTakenBadge } from './TimeTakenBadge'
import { QuestionBreakdownTable } from './QuestionBreakdownTable'
import type { QuestionBreakdownItem } from '@/api/sessions'

describe('PassFailBanner (FR-BB45)', () => {
  it('shows Passed when passed', () => {
    render(<PassFailBanner passed />)
    expect(screen.getByTestId('pass-fail-banner')).toHaveTextContent('Passed')
  })

  it('shows Failed when not passed', () => {
    render(<PassFailBanner passed={false} />)
    expect(screen.getByTestId('pass-fail-banner')).toHaveTextContent('Failed')
  })
})

describe('ScoreDial (FR-BB45)', () => {
  const circumference = 2 * Math.PI * 54

  function progressOffset() {
    const circles = document.querySelectorAll('circle')
    return Number(circles[1].getAttribute('stroke-dashoffset'))
  }

  it('labels the dial with the rounded percentage and fills the arc proportionally', () => {
    render(<ScoreDial scorePct={72.4} primaryColor="#123456" />)
    expect(screen.getByRole('img', { name: '72%' })).toBeInTheDocument()
    expect(progressOffset()).toBeCloseTo(circumference * (1 - 0.724), 3)
    expect(document.querySelectorAll('circle')[1]).toHaveAttribute('stroke', '#123456')
  })

  it('clamps out-of-range scores to 0..100', () => {
    const { unmount } = render(<ScoreDial scorePct={140} primaryColor="#000" />)
    expect(screen.getByRole('img', { name: '100%' })).toBeInTheDocument()
    expect(progressOffset()).toBeCloseTo(0, 5)
    unmount()

    render(<ScoreDial scorePct={-5} primaryColor="#000" />)
    expect(screen.getByRole('img', { name: '0%' })).toBeInTheDocument()
    expect(progressOffset()).toBeCloseTo(circumference, 5)
  })
})

describe('TimeTakenBadge (FR-BB45)', () => {
  it('renders nothing when time taken is unknown', () => {
    const { container } = render(<TimeTakenBadge seconds={null} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('formats seconds as minutes and seconds', () => {
    render(<TimeTakenBadge seconds={754} />)
    expect(screen.getByText('12m 34s')).toBeInTheDocument()
    expect(screen.getByText(/Time Taken/)).toBeInTheDocument()
  })

  it('renders 0 seconds (not treated as unknown)', () => {
    render(<TimeTakenBadge seconds={0} />)
    expect(screen.getByText('0m 0s')).toBeInTheDocument()
  })
})

function item(overrides: Partial<QuestionBreakdownItem> = {}): QuestionBreakdownItem {
  return {
    question_id: 'q1',
    stem: 'What is 2+2?',
    employee_answer: ['3'],
    correct_answer: ['4'],
    points_earned: 0,
    max_points: 1,
    explanation: null,
    manual_feedback: null,
    ...overrides,
  }
}

describe('QuestionBreakdownTable (FR-BB45)', () => {
  it('renders nothing when the breakdown is hidden by the exam settings (undefined)', () => {
    const { container } = render(<QuestionBreakdownTable breakdown={undefined} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('shows answers, correct answers and points per question', () => {
    render(
      <QuestionBreakdownTable
        breakdown={[
          item(),
          item({
            question_id: 'q2',
            stem: 'Pick colours',
            employee_answer: ['red', 'blue'],
            correct_answer: ['red', 'blue'],
            points_earned: 2,
            max_points: 2,
          }),
        ]}
      />,
    )
    const rows = screen.getAllByRole('row').slice(1)
    expect(within(rows[0]).getByText('3')).toBeInTheDocument()
    expect(within(rows[0]).getByText('4')).toBeInTheDocument()
    expect(within(rows[0]).getByText('0 / 1')).toBeInTheDocument()
    expect(within(rows[1]).getAllByText('red, blue')).toHaveLength(2)
    expect(within(rows[1]).getByText('2 / 2')).toBeInTheDocument()
  })

  it('shows a dash for unanswered questions and missing explanation / feedback', () => {
    render(<QuestionBreakdownTable breakdown={[item({ employee_answer: [] })]} />)
    const row = screen.getAllByRole('row')[1]
    // answer, explanation, feedback
    expect(within(row).getAllByText('—')).toHaveLength(3)
  })

  it('truncates long stems to 120 characters', () => {
    const stem = 'S'.repeat(130)
    render(<QuestionBreakdownTable breakdown={[item({ stem })]} />)
    expect(screen.getByText(`${'S'.repeat(120)}…`)).toBeInTheDocument()
    expect(screen.queryByText(stem)).not.toBeInTheDocument()
  })

  it('collapses long explanations and expands/collapses them on demand', () => {
    const explanation = 'E'.repeat(250)
    render(<QuestionBreakdownTable breakdown={[item({ explanation })]} />)
    expect(screen.queryByText(explanation, { exact: false })).not.toBeInTheDocument()
    expect(screen.getByText(/E{200}…/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Show more' }))
    expect(screen.getByText(new RegExp(`^${explanation}`))).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Show less' }))
    expect(screen.getByRole('button', { name: 'Show more' })).toBeInTheDocument()
  })

  it('does not offer expand for short explanations and shows examiner feedback', () => {
    render(
      <QuestionBreakdownTable
        breakdown={[item({ explanation: 'Because maths', manual_feedback: 'Check your sums' })]}
      />,
    )
    expect(screen.getByText('Because maths')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /show/i })).not.toBeInTheDocument()
    expect(screen.getByText('Check your sums')).toBeInTheDocument()
  })
})
