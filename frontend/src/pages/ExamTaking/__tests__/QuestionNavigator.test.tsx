import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { QuestionNavigator } from '../QuestionNavigator'
import type { SessionQuestion } from '@/api/sessions'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

const questions = [{ id: 'q1' }, { id: 'q2' }, { id: 'q3' }] as SessionQuestion[]

describe('QuestionNavigator', () => {
  it('renders one numbered button per question', () => {
    render(
      <QuestionNavigator questions={questions} answeredIds={new Set()} flaggedIds={new Set()} onNavigate={() => {}} />,
    )
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual(['1', '2', '3'])
  })

  it('navigates to the clicked question', async () => {
    const onNavigate = vi.fn()
    render(
      <QuestionNavigator questions={questions} answeredIds={new Set()} flaggedIds={new Set()} onNavigate={onNavigate} />,
    )
    await userEvent.click(screen.getByRole('button', { name: '2' }))
    expect(onNavigate).toHaveBeenCalledWith('q2')
  })

  it('styles flagged over answered over unanswered', () => {
    render(
      <QuestionNavigator
        questions={questions}
        answeredIds={new Set(['q1', 'q2'])}
        flaggedIds={new Set(['q2'])}
        onNavigate={() => {}}
      />,
    )
    expect(screen.getByRole('button', { name: '1' }).className).toContain('bg-info')
    expect(screen.getByRole('button', { name: '2' }).className).toContain('bg-bg-warning')
    expect(screen.getByRole('button', { name: '3' }).className).not.toMatch(/bg-info|bg-bg-warning/)
  })

  it('renders the legend', () => {
    render(
      <QuestionNavigator questions={[]} answeredIds={new Set()} flaggedIds={new Set()} onNavigate={() => {}} />,
    )
    expect(screen.getByText('exam.taking.navigator.unanswered')).toBeInTheDocument()
    expect(screen.getByText('exam.taking.navigator.answered')).toBeInTheDocument()
    expect(screen.getByText('exam.taking.navigator.flagged')).toBeInTheDocument()
  })
})
