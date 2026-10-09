import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { Step4Review } from './Step4Review'
import type { ExamDetail, QuestionRuleDetail } from '@/api/exams'

const useExam = vi.fn()
const useEligibleCounts = vi.fn()

vi.mock('@/api/exams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/exams')>()),
  useExam: (...a: unknown[]) => useExam(...a),
  useEligibleCounts: (...a: unknown[]) => useEligibleCounts(...a),
  usePublishExam: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUnpublishExam: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useArchiveExam: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

function rule(id: string, count: number, overrides: Partial<QuestionRuleDetail> = {}): QuestionRuleDetail {
  return {
    id,
    section_id: null,
    mode: 'random',
    category_id: null,
    tag_ids: [],
    difficulty: null,
    count,
    sort_order: 0,
    ...overrides,
  }
}

function makeExam(rules: QuestionRuleDetail[]): ExamDetail {
  return {
    id: 'exam-1',
    title: 'Security Basics',
    description: null,
    status: 'draft',
    time_limit_minutes: 30,
    passing_score_pct: 70,
    max_attempts: 1,
    available_from: null,
    available_until: null,
    shuffle_questions: false,
    shuffle_options: false,
    show_answers: 'never',
    on_tab_switch: 'log',
    certificate_enabled: false,
    created_by: 'u1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    sections: [],
    rules,
  }
}

function renderStep4() {
  return render(
    <MemoryRouter>
      <Step4Review examId="exam-1" onBack={vi.fn()} onPublished={vi.fn()} />
    </MemoryRouter>,
  )
}

/** The rule list item containing the given rule summary text. */
function ruleItem(n: number) {
  return screen.getByText(new RegExp(`^Rule ${n}:`)).closest('li') as HTMLElement
}

describe('Step4Review eligible counts (FR-BB315)', () => {
  beforeEach(() => {
    useExam.mockReset()
    useEligibleCounts.mockReset()
  })

  it('shows a green badge when eligible questions cover the rule count', () => {
    useExam.mockReturnValue({ data: makeExam([rule('r1', 5)]), isLoading: false })
    useEligibleCounts.mockReturnValue({
      data: { counts: [{ rule_id: 'r1', eligible: 8 }] },
      isLoading: false,
      isError: false,
    })
    renderStep4()
    const badge = ruleItem(1).querySelector('[class*="bg-green"]')
    expect(badge).toHaveTextContent('8 eligible')
  })

  it('treats eligible == required as sufficient (green)', () => {
    useExam.mockReturnValue({ data: makeExam([rule('r1', 5)]), isLoading: false })
    useEligibleCounts.mockReturnValue({
      data: { counts: [{ rule_id: 'r1', eligible: 5 }] },
      isLoading: false,
      isError: false,
    })
    renderStep4()
    expect(ruleItem(1).querySelector('[class*="bg-green"]')).toHaveTextContent('5 eligible')
  })

  it('shows amber when some but not enough questions are eligible, and red when none', () => {
    useExam.mockReturnValue({
      data: makeExam([rule('r1', 5), rule('r2', 3, { sort_order: 1 })]),
      isLoading: false,
    })
    useEligibleCounts.mockReturnValue({
      data: {
        counts: [
          { rule_id: 'r1', eligible: 2 },
          { rule_id: 'r2', eligible: 0 },
        ],
      },
      isLoading: false,
      isError: false,
    })
    renderStep4()
    expect(ruleItem(1).querySelector('[class*="bg-amber"]')).toHaveTextContent('2 eligible')
    expect(ruleItem(2).querySelector('[class*="bg-red"]')).toHaveTextContent('0 eligible')
  })

  it('matches counts to rules by rule_id, not by position', () => {
    useExam.mockReturnValue({
      data: makeExam([rule('r1', 5), rule('r2', 5, { sort_order: 1 })]),
      isLoading: false,
    })
    useEligibleCounts.mockReturnValue({
      data: {
        counts: [
          { rule_id: 'r2', eligible: 9 },
          { rule_id: 'r1', eligible: 1 },
        ],
      },
      isLoading: false,
      isError: false,
    })
    renderStep4()
    expect(ruleItem(1)).toHaveTextContent('1 eligible')
    expect(ruleItem(2)).toHaveTextContent('9 eligible')
  })

  it('shows no badge for a rule absent from the counts response', () => {
    useExam.mockReturnValue({ data: makeExam([rule('r1', 5)]), isLoading: false })
    useEligibleCounts.mockReturnValue({ data: { counts: [] }, isLoading: false, isError: false })
    renderStep4()
    expect(ruleItem(1)).not.toHaveTextContent('eligible')
  })

  it('shows a spinner (no count) while eligible counts load', () => {
    useExam.mockReturnValue({ data: makeExam([rule('r1', 5)]), isLoading: false })
    useEligibleCounts.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    renderStep4()
    expect(ruleItem(1).querySelector('svg.animate-spin')).not.toBeNull()
    expect(ruleItem(1)).not.toHaveTextContent('eligible')
  })

  it('shows an inline error per rule when counts fail to load', () => {
    useExam.mockReturnValue({
      data: makeExam([rule('r1', 5), rule('r2', 2, { sort_order: 1 })]),
      isLoading: false,
    })
    useEligibleCounts.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderStep4()
    expect(screen.getAllByText('Could not load counts')).toHaveLength(2)
  })

  it('includes difficulty in the rule summary when set', () => {
    useExam.mockReturnValue({ data: makeExam([rule('r1', 4, { difficulty: 'hard' })]), isLoading: false })
    useEligibleCounts.mockReturnValue({ data: { counts: [] }, isLoading: false, isError: false })
    renderStep4()
    expect(ruleItem(1)).toHaveTextContent('Rule 1: random, 4 questions, hard')
  })

  it('shows a dash instead of rules when the exam has none', () => {
    useExam.mockReturnValue({ data: makeExam([]), isLoading: false })
    useEligibleCounts.mockReturnValue({ data: { counts: [] }, isLoading: false, isError: false })
    renderStep4()
    expect(screen.queryByText(/^Rule \d+:/)).not.toBeInTheDocument()
  })
})
