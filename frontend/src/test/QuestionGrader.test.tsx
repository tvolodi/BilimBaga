import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { QuestionGrader } from '@/components/grading/QuestionGrader'
import type { GradingQuestion } from '@/api/grading'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

function makeQuestion(overrides: Partial<GradingQuestion> = {}): GradingQuestion {
  return {
    question_id: 'q-1',
    stem: 'What is the capital of France?',
    text_answer: 'Paris',
    grading_status: 'pending_manual',
    current_score_pct: null,
    manual_feedback: null,
    ai_reasoning: null,
    ...overrides,
  }
}

describe('QuestionGrader — AI grading display (FR-BB73)', () => {
  it('shows "AI Graded" badge when grading_status is ai_graded', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'ai_graded', ai_reasoning: 'Mostly correct' })}
        score={80}
        feedback=""
        onChange={vi.fn()}
      />
    )
    expect(screen.getByText('grading.ai_graded')).toBeInTheDocument()
  })

  it('does NOT show AI badge when grading_status is pending_manual', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'pending_manual' })}
        score={null}
        feedback=""
        onChange={vi.fn()}
      />
    )
    expect(screen.queryByText('grading.ai_graded')).not.toBeInTheDocument()
  })

  it('does NOT show AI badge when grading_status is graded', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'graded' })}
        score={100}
        feedback=""
        onChange={vi.fn()}
      />
    )
    expect(screen.queryByText('grading.ai_graded')).not.toBeInTheDocument()
  })

  it('shows AI reasoning toggle when ai_graded and ai_reasoning present', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'ai_graded', ai_reasoning: 'Partial credit: answer omits arrondissement.' })}
        score={70}
        feedback=""
        onChange={vi.fn()}
      />
    )
    // The "show" toggle button should be visible
    expect(screen.getByText('grading.ai_reasoning_show')).toBeInTheDocument()
    // Reasoning text itself should be hidden initially
    expect(screen.queryByText('Partial credit: answer omits arrondissement.')).not.toBeInTheDocument()
  })

  it('expands AI reasoning when toggle is clicked', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'ai_graded', ai_reasoning: 'Good answer!' })}
        score={100}
        feedback=""
        onChange={vi.fn()}
      />
    )
    const toggleBtn = screen.getByText('grading.ai_reasoning_show')
    fireEvent.click(toggleBtn)
    expect(screen.getByText('Good answer!')).toBeInTheDocument()
    expect(screen.getByText('grading.ai_reasoning_hide')).toBeInTheDocument()
  })

  it('collapses AI reasoning when toggle is clicked again', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'ai_graded', ai_reasoning: 'Good answer!' })}
        score={100}
        feedback=""
        onChange={vi.fn()}
      />
    )
    const toggleBtn = screen.getByText('grading.ai_reasoning_show')
    fireEvent.click(toggleBtn)
    fireEvent.click(screen.getByText('grading.ai_reasoning_hide'))
    expect(screen.queryByText('Good answer!')).not.toBeInTheDocument()
  })

  it('does NOT show reasoning toggle when ai_reasoning is null', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'ai_graded', ai_reasoning: null })}
        score={80}
        feedback=""
        onChange={vi.fn()}
      />
    )
    expect(screen.queryByText('grading.ai_reasoning_show')).not.toBeInTheDocument()
  })
})

describe('QuestionGrader — score input (FR-BB47 AC-6)', () => {
  it('keeps slider and numeric field in sync with the score', () => {
    render(<QuestionGrader question={makeQuestion()} score={64} feedback="" onChange={vi.fn()} />)
    expect(screen.getByRole('slider')).toHaveValue('64')
    expect(screen.getByRole('spinbutton')).toHaveValue(64)
  })

  it('reports slider and numeric edits through onChange, and null for an emptied field', () => {
    const onChange = vi.fn()
    render(<QuestionGrader question={makeQuestion()} score={10} feedback="fb" onChange={onChange} />)
    fireEvent.change(screen.getByRole('slider'), { target: { value: '55' } })
    expect(onChange).toHaveBeenLastCalledWith(55, 'fb')
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '' } })
    expect(onChange).toHaveBeenLastCalledWith(null, 'fb')
  })

  it.each([101, -3])('shows an inline error for out-of-range score %i', (bad) => {
    render(<QuestionGrader question={makeQuestion()} score={bad} feedback="" onChange={vi.fn()} />)
    expect(screen.getByText('grading.score_error')).toBeInTheDocument()
  })

  it('shows no error for boundary scores 0 and 100', () => {
    const { rerender } = render(
      <QuestionGrader question={makeQuestion()} score={0} feedback="" onChange={vi.fn()} />,
    )
    expect(screen.queryByText('grading.score_error')).not.toBeInTheDocument()
    rerender(<QuestionGrader question={makeQuestion()} score={100} feedback="" onChange={vi.fn()} />)
    expect(screen.queryByText('grading.score_error')).not.toBeInTheDocument()
  })
})
