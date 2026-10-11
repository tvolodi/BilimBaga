import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { Step1BasicSettings } from './Step1BasicSettings'
import type { ExamDetail } from '@/api/exams'

// Only the update hook is replaced, so the test can read the payload Step 1 sends (FR-BB73 AC-22).
const { updateExamMock } = vi.hoisted(() => ({ updateExamMock: vi.fn() }))
vi.mock('@/api/exams', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/exams')>()
  return { ...actual, useUpdateExam: () => ({ mutateAsync: updateExamMock, isPending: false }) }
})

function makeExam(overrides: Partial<ExamDetail> = {}): ExamDetail {
  return {
    id: 'exam-1',
    title: 'Test Exam',
    description: null,
    status: 'draft',
    time_limit_minutes: 60,
    passing_score_pct: 70,
    max_attempts: 1,
    available_from: null,
    available_until: null,
    shuffle_questions: false,
    shuffle_options: false,
    show_answers: 'never',
    on_tab_switch: 'log',
    certificate_enabled: false,
    ai_grading_mode: 'suggest',
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    sections: [],
    rules: [],
    ...overrides,
  }
}

function renderStep1(exam: ExamDetail | null, onDone = () => {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <Step1BasicSettings exam={exam} onDone={onDone} resolvedSectionId={null} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Step1BasicSettings', () => {
  it('renders all inputs as enabled for a draft exam', () => {
    renderStep1(makeExam({ status: 'draft' }))

    expect(screen.getByRole('textbox', { name: /exam title/i })).not.toBeDisabled()
    expect(screen.getByRole('spinbutton', { name: /time limit/i })).not.toBeDisabled()
    expect(screen.getByRole('spinbutton', { name: /passing score/i })).not.toBeDisabled()
    expect(screen.getByRole('spinbutton', { name: /max attempts/i })).not.toBeDisabled()
    expect(screen.getByRole('combobox', { name: /show answers/i })).not.toBeDisabled()
    expect(screen.getByRole('combobox', { name: /tab switch/i })).not.toBeDisabled()
  })

  it('does not show the read-only banner for a draft exam', () => {
    renderStep1(makeExam({ status: 'draft' }))
    expect(screen.queryByText(/unpublish it to make changes/i)).not.toBeInTheDocument()
  })

  it('disables all form inputs when exam status is active', () => {
    renderStep1(makeExam({ status: 'active' }))

    expect(screen.getByRole('textbox', { name: /exam title/i })).toBeDisabled()
    expect(screen.getByRole('spinbutton', { name: /time limit/i })).toBeDisabled()
    expect(screen.getByRole('spinbutton', { name: /passing score/i })).toBeDisabled()
    expect(screen.getByRole('spinbutton', { name: /max attempts/i })).toBeDisabled()
    expect(screen.getByRole('combobox', { name: /show answers/i })).toBeDisabled()
    expect(screen.getByRole('combobox', { name: /tab switch/i })).toBeDisabled()
  })

  it('disables all switch toggles when exam status is active', () => {
    renderStep1(makeExam({ status: 'active' }))

    const switches = screen.getAllByRole('switch')
    expect(switches.length).toBeGreaterThanOrEqual(3)
    switches.forEach((sw) => {
      expect(sw).toBeDisabled()
    })
  })

  it('keeps the Next button enabled for active exams to allow navigation', () => {
    renderStep1(makeExam({ status: 'active' }))

    const submitBtn = screen.getByRole('button', { name: /next/i })
    expect(submitBtn).not.toBeDisabled()
  })

  it('calls onDone directly without saving when Next is clicked for an active exam', () => {
    const onDone = vi.fn()
    renderStep1(makeExam({ status: 'active', id: 'exam-active-1' }), onDone)

    const submitBtn = screen.getByRole('button', { name: /next/i })
    fireEvent.click(submitBtn)

    expect(onDone).toHaveBeenCalledWith('exam-active-1', null)
  })

  it('shows the read-only banner when exam status is active', () => {
    renderStep1(makeExam({ status: 'active' }))

    expect(
      screen.getByText(/unpublish it to make changes/i),
    ).toBeInTheDocument()
  })

  it('renders inputs as enabled when no exam is passed (create mode)', () => {
    renderStep1(null)

    expect(screen.getByRole('textbox', { name: /exam title/i })).not.toBeDisabled()
    expect(screen.queryByText(/unpublish it to make changes/i)).not.toBeInTheDocument()
  })

  it('shows the exam AI grading mode with the help text of that mode (FR-BB73 AC-22)', () => {
    renderStep1(makeExam({ ai_grading_mode: 'auto' }))

    expect(screen.getByRole('combobox', { name: /ai grading/i })).toHaveValue('auto')
    expect(screen.getByTestId('ai-grading-mode-help')).toHaveTextContent(/cannot decide pass or fail/i)
  })

  it('sends the chosen AI grading mode with the save (FR-BB73 AC-22)', async () => {
    updateExamMock.mockReset()
    updateExamMock.mockResolvedValue(makeExam())
    const onDone = vi.fn()
    renderStep1(makeExam({ ai_grading_mode: 'suggest' }), onDone)

    fireEvent.change(screen.getByRole('combobox', { name: /ai grading/i }), { target: { value: 'off' } })
    fireEvent.click(screen.getByRole('button', { name: /next/i }))

    await waitFor(() => expect(onDone).toHaveBeenCalled())
    expect(updateExamMock).toHaveBeenCalledWith(expect.objectContaining({ ai_grading_mode: 'off' }))
  })
})
