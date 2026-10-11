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
    ai_suggested_score_pct: null,
    ai_suggested_at: null,
    ai_reasoning: null,
    ai_flags: null,
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

describe('QuestionGrader — AI suggestion of a waiting answer (FR-BB73 AC-18)', () => {
  const waiting = {
    grading_status: 'pending_manual' as const,
    ai_suggested_score_pct: 60,
    ai_suggested_at: '2026-10-11T06:00:00Z',
    ai_reasoning: 'Covers the main point.',
    ai_flags: ['decisive_pass'],
  }

  it('shows the suggestion panel with the flag chips and the reasoning', () => {
    render(<QuestionGrader question={makeQuestion(waiting)} score={null} feedback="" onChange={vi.fn()} onAccept={vi.fn()} />)
    expect(screen.getByText('grading.suggestion.title')).toBeInTheDocument()
    expect(screen.getByText('grading.flag.decisive_pass')).toBeInTheDocument()
    expect(screen.getByText('Covers the main point.')).toBeInTheDocument()
  })

  it('calls onAccept when Accept is clicked', () => {
    const onAccept = vi.fn()
    render(<QuestionGrader question={makeQuestion(waiting)} score={null} feedback="" onChange={vi.fn()} onAccept={onAccept} />)
    fireEvent.click(screen.getByRole('button', { name: 'grading.suggestion.accept' }))
    expect(onAccept).toHaveBeenCalledTimes(1)
  })

  it('shows no suggestion panel when the answer has no suggestion', () => {
    render(<QuestionGrader question={makeQuestion()} score={null} feedback="" onChange={vi.fn()} onAccept={vi.fn()} />)
    expect(screen.queryByText('grading.suggestion.title')).not.toBeInTheDocument()
  })

  it('renders the reasoning as plain text: no HTML, no markdown', () => {
    const hostile = '<img src=x onerror=alert(1)> and **bold**'
    const { container } = render(
      <QuestionGrader question={makeQuestion({ ...waiting, ai_reasoning: hostile })} score={null} feedback="" onChange={vi.fn()} onAccept={vi.fn()} />,
    )
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('strong, b')).toBeNull()
    expect(screen.getByText(hostile)).toBeInTheDocument()
  })

  it('hides Accept in a read-only view', () => {
    render(
      <QuestionGrader question={makeQuestion(waiting)} score={null} feedback="" onChange={vi.fn()} onAccept={vi.fn()} readOnly />,
    )
    expect(screen.queryByRole('button', { name: 'grading.suggestion.accept' })).not.toBeInTheDocument()
  })

  it('shows the AI score and reasoning of an ai_graded answer', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'ai_graded', current_score_pct: 40, ai_reasoning: 'Partly right.' })}
        score={40}
        feedback=""
        onChange={vi.fn()}
      />,
    )
    expect(screen.getByText('grading.suggestion.ai_score')).toBeInTheDocument()
    expect(screen.getByText('grading.ai_reasoning_show')).toBeInTheDocument()
  })
})

describe('QuestionGrader — read-only graded answer (FR-BB47 AC-5)', () => {
  it('disables the score and the feedback', () => {
    render(
      <QuestionGrader
        question={makeQuestion({ grading_status: 'graded', current_score_pct: 75, manual_feedback: 'ok' })}
        score={75}
        feedback="ok"
        onChange={vi.fn()}
        readOnly
      />,
    )
    expect(screen.getByRole('slider')).toBeDisabled()
    expect(screen.getByRole('spinbutton')).toBeDisabled()
    expect(screen.getByDisplayValue('ok')).toHaveAttribute('readonly')
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
