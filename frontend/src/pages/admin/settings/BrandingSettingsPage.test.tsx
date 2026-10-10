import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
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

  it('blocks save and shows the contrast ratio when the primary colour fails 4.5:1 (FR-BB320 AC-1)', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const hex = (await screen.findByLabelText('Primary Colour hex')) as HTMLInputElement
    fireEvent.change(hex, { target: { value: '#0ea5e9' } })

    expect(screen.getByText(/ratio: 2\.77:1/)).toBeInTheDocument()
    expect(screen.getByTestId('primary-color-blocked')).toHaveTextContent(
      /save is blocked until the primary colour reaches 4\.5:1/i,
    )
    expect(screen.getByTestId('primary-color-blocked')).toHaveTextContent('2.77:1')
    expect(screen.getByRole('button', { name: /save changes/i })).toBeDisabled()
  })

  it('blocks save when the primary colour is not a valid hex value', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const hex = (await screen.findByLabelText('Primary Colour hex')) as HTMLInputElement
    fireEvent.change(hex, { target: { value: 'not-a-colour' } })

    expect(screen.getByTestId('primary-color-blocked')).toHaveTextContent(/enter a valid hex colour/i)
    expect(screen.getByRole('button', { name: /save changes/i })).toBeDisabled()
  })

  it('saves a primary colour that reaches 4.5:1 and sends it as the only change', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const hex = (await screen.findByLabelText('Primary Colour hex')) as HTMLInputElement
    fireEvent.change(hex, { target: { value: '#2e6db4' } })

    expect(screen.queryByTestId('primary-color-blocked')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(lastPutBody).not.toBeNull()
    })
    // #498: the saved colour is the canonical uppercase form, the same as the seeded default.
    expect(lastPutBody).toEqual({ primary_color: '#2E6DB4' })
  })

  // #498: however the colour is typed, the saved value is the same string as the default, #2E6DB4.
  it.each(['2E6DB4', '#2E6DB4'])('saves a hand-typed %s as #2E6DB4', async (typed) => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const hex = (await screen.findByLabelText('Primary Colour hex')) as HTMLInputElement
    await userEvent.clear(hex)
    await userEvent.type(hex, typed)

    await userEvent.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(lastPutBody).not.toBeNull()
    })
    expect(lastPutBody).toEqual({ primary_color: '#2E6DB4' })
  })

  it('does not block other fields when the stored primary colour already fails', async () => {
    server.use(
      http.get('/api/v1/tenant/config', () =>
        HttpResponse.json({
          data: { ...initialConfig, primary_color: '#0ea5e9' },
          error: null,
        }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <BrandingSettingsPage />
      </Wrapper>,
    )
    const appName = (await screen.findByLabelText('Application Name')) as HTMLInputElement
    await userEvent.clear(appName)
    await userEvent.type(appName, 'NewName')

    expect(screen.queryByTestId('primary-color-blocked')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(lastPutBody).not.toBeNull()
    })
    expect(lastPutBody).toEqual({ app_name: 'NewName' })
  })
})
