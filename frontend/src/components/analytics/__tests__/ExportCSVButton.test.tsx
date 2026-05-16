import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/i18n'
import { ExportCSVButton } from '../ExportCSVButton'

function setup(examId = 'exam-123') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const { container } = render(
    <QueryClientProvider client={qc}>
      <ExportCSVButton examId={examId} />
    </QueryClientProvider>,
  )
  return container
}

describe('ExportCSVButton', () => {
  it('renders the export button', () => {
    setup()
    const buttons = screen.getAllByRole('button')
    expect(buttons.length).toBeGreaterThanOrEqual(1)
  })

  it('button is enabled initially', () => {
    setup()
    const buttons = screen.getAllByRole('button')
    expect(buttons[0]).not.toBeDisabled()
  })

  it('calls fetch on button click with correct URL', async () => {
    const mockBlob = new Blob(['csv,data'], { type: 'text/csv' })
    // Use direct assignment like other tests in the codebase
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => mockBlob,
      headers: { get: vi.fn(() => 'attachment; filename="results.csv"') },
    })
    URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    URL.revokeObjectURL = vi.fn()

    const appendSpy = vi.spyOn(document.body, 'appendChild').mockImplementation((el) => el)
    const removeSpy = vi.spyOn(document.body, 'removeChild').mockImplementation((el) => el)

    const container = setup('exam-abc')
    // Use container to find button - avoids any screen isolation issue
    const btn = container.querySelector('button')
    expect(btn).toBeTruthy()
    fireEvent.click(btn!)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v1/admin/exams/exam-abc/results/export',
        expect.objectContaining({ credentials: 'include' }),
      )
    })

    appendSpy.mockRestore()
    removeSpy.mockRestore()
  })

  it('disables button during download', async () => {
    let resolveBlob!: (value: Blob) => void
    const blobPromise = new Promise<Blob>((resolve) => {
      resolveBlob = resolve
    })
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      blob: () => blobPromise,
      headers: { get: vi.fn(() => null) },
    })
    URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    URL.revokeObjectURL = vi.fn()

    const appendSpy = vi.spyOn(document.body, 'appendChild').mockImplementation((el) => el)
    const removeSpy = vi.spyOn(document.body, 'removeChild').mockImplementation((el) => el)

    const container = setup()
    const btn = container.querySelector('button')
    expect(btn).toBeTruthy()
    fireEvent.click(btn!)

    await waitFor(() => {
      expect(btn).toBeDisabled()
    })

    resolveBlob(new Blob(['data']))

    await waitFor(() => {
      expect(btn).not.toBeDisabled()
    })

    appendSpy.mockRestore()
    removeSpy.mockRestore()
  })
})
