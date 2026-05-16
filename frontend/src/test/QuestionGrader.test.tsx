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
