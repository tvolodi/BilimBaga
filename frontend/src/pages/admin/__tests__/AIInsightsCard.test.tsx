import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { AIInsightsCard } from '@/pages/admin/ExamAnalyticsPage'

// Mock the AI insights hook.
vi.mock('@/api/ai', () => ({
  useAIInsights: vi.fn(),
}))

import { useAIInsights } from '@/api/ai'
const mockUseAIInsights = vi.mocked(useAIInsights)

function buildRefetch(fn?: () => void) {
  return vi.fn(() => {
    fn?.()
    return Promise.resolve({ data: undefined, error: null })
  })
}

function setup(examId = 'exam-1') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], 'fake-token')
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <AIInsightsCard examId={examId} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('AIInsightsCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the card header and regenerate button', () => {
    mockUseAIInsights.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
      refetch: buildRefetch(),
    } as ReturnType<typeof useAIInsights>)

    setup()

    expect(screen.getByText('AI Insights')).toBeInTheDocument()
    expect(screen.getByText('Regenerate')).toBeInTheDocument()
  })

  it('shows skeleton while loading (card expanded)', async () => {
    mockUseAIInsights.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      refetch: buildRefetch(),
    } as ReturnType<typeof useAIInsights>)

    setup()

    // Expand the card.
    const toggleBtn = screen.getByRole('button', { name: /toggle expand/i })
    fireEvent.click(toggleBtn)

    await waitFor(() => {
      expect(screen.getByTestId('ai-insights-skeleton')).toBeInTheDocument()
    })
  })

  it('shows error message when query fails (card expanded)', async () => {
    mockUseAIInsights.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      refetch: buildRefetch(),
    } as ReturnType<typeof useAIInsights>)

    setup()

    const toggleBtn = screen.getByRole('button', { name: /toggle expand/i })
    fireEvent.click(toggleBtn)

    await waitFor(() => {
      expect(screen.getByTestId('ai-insights-error')).toBeInTheDocument()
    })
  })

  it('renders insight bullet list when data is available (card expanded)', async () => {
    const insights = ['Insight one.', 'Insight two.', 'Insight three.']
    mockUseAIInsights.mockReturnValue({
      data: {
        insights,
        generated_at: '2026-05-14T10:30:00Z',
        cached: true,
      },
      isLoading: false,
      isError: false,
      refetch: buildRefetch(),
    } as ReturnType<typeof useAIInsights>)

    setup()

    const toggleBtn = screen.getByRole('button', { name: /toggle expand/i })
    fireEvent.click(toggleBtn)

    await waitFor(() => {
      expect(screen.getByTestId('ai-insights-list')).toBeInTheDocument()
    })
    insights.forEach((txt) => {
      expect(screen.getByText(txt)).toBeInTheDocument()
    })
  })

  it('calls refetch when Regenerate is clicked', async () => {
    const refetch = buildRefetch()
    mockUseAIInsights.mockReturnValue({
      data: {
        insights: ['Insight.'],
        generated_at: '2026-05-14T10:30:00Z',
        cached: false,
      },
      isLoading: false,
      isError: false,
      refetch,
    } as ReturnType<typeof useAIInsights>)

    setup()

    const regenerateBtn = screen.getByText('Regenerate')
    fireEvent.click(regenerateBtn)

    await waitFor(() => {
      expect(refetch).toHaveBeenCalledTimes(1)
    })
  })
})
