import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { BrandingSettingsPage } from './BrandingSettingsPage'

const initialConfig = {
  app_name: 'BilimBaga',
  primary_color: '#003366',
  accent_color: '#f59e0b',
  default_locale: 'en',
  available_locales: ['kk', 'ru', 'en'],
}

let lastPutBody: Record<string, unknown> | null = null

const server = setupServer(
  http.get('/api/v1/tenant/config', () =>
    HttpResponse.json({ data: initialConfig, error: null }),
  ),
  http.put('/api/v1/tenant/config', async ({ request }) => {
    lastPutBody = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({ data: {}, error: null })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  lastPutBody = null
})
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('BrandingSettingsPage', () => {
  it('pre-populates the app name input from the loaded config (AC-2)', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    await waitFor(() => {
      const input = screen.getByLabelText('Application Name') as HTMLInputElement
      expect(input.value).toBe('BilimBaga')
    })
  })

  it('renders the live preview with the current colours (AC-6)', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const header = await screen.findByTestId('branding-preview-header')
    expect(header).toHaveStyle({ backgroundColor: 'rgb(0, 51, 102)' })
  })

  it('only PUTs the changed fields on save (AC-9)', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const appName = (await screen.findByLabelText('Application Name')) as HTMLInputElement
    await userEvent.clear(appName)
    await userEvent.type(appName, 'NewName')

    const saveButton = screen.getByRole('button', { name: /save changes/i })
    await userEvent.click(saveButton)

    await waitFor(() => {
      expect(lastPutBody).not.toBeNull()
    })
    expect(lastPutBody).toEqual({ app_name: 'NewName' })
  })

  it('blocks save when the default locale is removed from available locales (AC-8)', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    await screen.findByLabelText('Application Name')

    const englishCheckbox = screen.getByLabelText('English')
    expect((englishCheckbox as HTMLInputElement).checked).toBe(true)
    await userEvent.click(englishCheckbox)

    expect(screen.getByText(/default language must be in the available languages list/i))
      .toBeInTheDocument()
    const saveButton = screen.getByRole('button', { name: /save changes/i })
    expect(saveButton).toBeDisabled()
  })

  it('shows a success message after a successful save', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const appName = (await screen.findByLabelText('Application Name')) as HTMLInputElement
    await userEvent.clear(appName)
    await userEvent.type(appName, 'NewName')
    await userEvent.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(screen.getByText(/branding updated successfully/i)).toBeInTheDocument()
    })
  })
})
