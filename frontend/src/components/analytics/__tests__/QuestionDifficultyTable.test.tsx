import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import '@/i18n'
import { QuestionDifficultyTable } from '../QuestionDifficultyTable'
import type { QuestionStat } from '@/api/analytics'

const questions: QuestionStat[] = [
  {
    question_id: 'q1',
    stem_preview: 'What is the capital of France?',
    correct_rate: 0.85,
    avg_time_seconds: 12.5,
    answer_distribution: [
      { option_id: 'o1', option_text: 'Paris', select_count: 85 },
      { option_id: 'o2', option_text: 'London', select_count: 10 },
      { option_id: 'o3', option_text: 'Berlin', select_count: 5 },
    ],
  },
  {
    question_id: 'q2',
    stem_preview: 'A very easy question that everyone gets right',
    correct_rate: 0.98,
    avg_time_seconds: 5.0,
    answer_distribution: [],
  },
  {
    question_id: 'q3',
    stem_preview: 'An extremely difficult question that few answer correctly',
    correct_rate: 0.25,
    avg_time_seconds: 45.0,
    answer_distribution: [],
  },
]

function setup(
  props: Partial<{
    questions: QuestionStat[]
    sortCol: 'correct_rate' | 'avg_time_seconds'
    sortDir: 'asc' | 'desc'
    onSort: (col: 'correct_rate' | 'avg_time_seconds') => void
  }> = {},
) {
  const onSort = props.onSort ?? vi.fn()
  render(
    <QuestionDifficultyTable
      questions={props.questions ?? questions}
      sortCol={props.sortCol ?? 'correct_rate'}
      sortDir={props.sortDir ?? 'asc'}
      onSort={onSort}
    />,
  )
  return { onSort }
}

describe('QuestionDifficultyTable', () => {
  it('renders all question rows', () => {
    setup()
    expect(screen.getByText('What is the capital of France?')).toBeInTheDocument()
    expect(screen.getByText(/A very easy question/)).toBeInTheDocument()
    expect(screen.getByText(/An extremely difficult/)).toBeInTheDocument()
  })

  it('renders correct rate as percentage', () => {
    setup()
    expect(screen.getByText('85.0%')).toBeInTheDocument()
    expect(screen.getByText('98.0%')).toBeInTheDocument()
    expect(screen.getByText('25.0%')).toBeInTheDocument()
  })

  it('applies red border class for low correct_rate (< 0.40)', () => {
    const { container } = render(
      <QuestionDifficultyTable
        questions={questions}
        sortCol="correct_rate"
        sortDir="asc"
        onSort={vi.fn()}
      />,
    )
    const rows = container.querySelectorAll('tbody tr')
    // Third question has correct_rate 0.25 — should have red border
    const redRow = rows[2]
    expect(redRow.className).toMatch(/border-l-red-500/)
  })

  it('applies amber border class for high correct_rate (> 0.95)', () => {
    const { container } = render(
      <QuestionDifficultyTable
        questions={questions}
        sortCol="correct_rate"
        sortDir="asc"
        onSort={vi.fn()}
      />,
    )
    const rows = container.querySelectorAll('tbody tr')
    // Second question has correct_rate 0.98 — should have amber border
    const amberRow = rows[1]
    expect(amberRow.className).toMatch(/border-l-amber-400/)
  })

  it('does not apply border for normal correct_rate', () => {
    const { container } = render(
      <QuestionDifficultyTable
        questions={questions}
        sortCol="correct_rate"
        sortDir="asc"
        onSort={vi.fn()}
      />,
    )
    const rows = container.querySelectorAll('tbody tr')
    // First question has correct_rate 0.85 — no coloured border
    const normalRow = rows[0]
    expect(normalRow.className).not.toMatch(/border-l-red-500/)
    expect(normalRow.className).not.toMatch(/border-l-amber-400/)
  })

  it('shows "Consider Revising" badge for low correct_rate', () => {
    setup()
    expect(screen.getByText('Consider Revising')).toBeInTheDocument()
  })

  it('shows "Consider Removing" badge for high correct_rate', () => {
    setup()
    expect(screen.getByText('Consider Removing')).toBeInTheDocument()
  })

  it('calls onSort when clicking correct rate header', () => {
    const { onSort } = setup()
    fireEvent.click(screen.getByText('Correct Rate'))
    expect(onSort).toHaveBeenCalledWith('correct_rate')
  })

  it('calls onSort when clicking avg time header', () => {
    const { onSort } = setup()
    fireEvent.click(screen.getByText('Avg Time (s)'))
    expect(onSort).toHaveBeenCalledWith('avg_time_seconds')
  })

  it('renders avg time values', () => {
    setup()
    expect(screen.getByText('12.5s')).toBeInTheDocument()
    expect(screen.getByText('5.0s')).toBeInTheDocument()
    expect(screen.getByText('45.0s')).toBeInTheDocument()
  })
})
