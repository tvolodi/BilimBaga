/**
 * Regression test for ISS-036:
 * QuestionDisplay must render the SaveIndicator near the answer area
 * when saveStatus is not 'idle'.
 */
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { QuestionDisplay } from '../QuestionDisplay'
import type { SessionQuestion } from '@/api/sessions'

// Minimal i18n mock
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: 'en' },
  }),
}))

const baseQuestion: SessionQuestion = {
  id: 'q1',
  sort_order: 1,
  stem: 'What is 2+2?',
  type: 'single_choice',
  options: [
    { id: 'o1', text: '3' },
    { id: 'o2', text: '4' },
  ],
}

const noop = () => {}

describe('QuestionDisplay — SaveIndicator (ISS-036 regression)', () => {
  it('does not render a save indicator when saveStatus is idle (default)', () => {
    render(
      <QuestionDisplay
        question={baseQuestion}
        savedAnswer={undefined}
        onAnswer={noop}
        isFlagged={false}
        onToggleFlag={noop}
      />,
    )
    expect(screen.queryByText('exam.taking.save.saving')).toBeNull()
    expect(screen.queryByText('exam.taking.save.saved')).toBeNull()
    expect(screen.queryByText('exam.taking.save.error')).toBeNull()
  })

  it('renders "Saving…" indicator near the answer area when saveStatus is saving', () => {
    render(
      <QuestionDisplay
        question={baseQuestion}
        savedAnswer={undefined}
        onAnswer={noop}
        isFlagged={false}
        onToggleFlag={noop}
        saveStatus="saving"
      />,
    )
    expect(screen.getByText('exam.taking.save.saving')).toBeInTheDocument()
  })

  it('renders "Saved ✓" indicator near the answer area when saveStatus is saved', () => {
    render(
      <QuestionDisplay
        question={baseQuestion}
        savedAnswer={undefined}
        onAnswer={noop}
        isFlagged={false}
        onToggleFlag={noop}
        saveStatus="saved"
      />,
    )
    expect(screen.getByText('exam.taking.save.saved')).toBeInTheDocument()
  })

  it('renders "Connection lost" indicator near the answer area when saveStatus is error', () => {
    render(
      <QuestionDisplay
        question={baseQuestion}
        savedAnswer={undefined}
        onAnswer={noop}
        isFlagged={false}
        onToggleFlag={noop}
        saveStatus="error"
      />,
    )
    expect(screen.getByText('exam.taking.save.error')).toBeInTheDocument()
  })
})
