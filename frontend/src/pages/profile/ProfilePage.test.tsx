import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import i18n from '@/i18n'
import { ProfilePage, ProfileRedirect } from './ProfilePage'

const meBody = {
  id: 'u1',
  email: 'ana@example.com',
  full_name: 'Ana Test',
  department_id: 'd1',
  department_name: 'Engineering',
  role_id: 'r1',
  role_name: 'employee',
  status: 'active',
  force_password_change: false,
  is_locked: false,
  preferred_locale: null,
  created_at: '2026-01-01T00:00:00Z',
}

function json(status: number, data: unknown, error: unknown = null) {
  return new Response(JSON.stringify({ data, error }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

// Per-test responses for the two endpoints the page touches.
let meResponse: () => Response
let patchResponse: () => Response
const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
  const url = String(input)
  if (url === '/api/v1/users/me' && init?.method === 'PATCH') return patchResponse()
  if (url === '/api/v1/users/me') return meResponse()
  return json(404, null, { code: 'NOT_FOUND', message: 'not mocked' })
})

function patchCalls() {
  return fetchMock.mock.calls.filter(([, init]) => init?.method === 'PATCH')
}

function renderProfile(path = '/admin/profile') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], 'tok-1')
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/admin/profile" element={<ProfilePage />} />
          <Route path="/change-password" element={<div>Change password screen</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  localStorage.clear()
  await i18n.changeLanguage('en')
  document.documentElement.lang = 'en'
  meResponse = () => json(200, meBody)
  patchResponse = () => json(200, { ...meBody, preferred_locale: 'ru' })
  fetchMock.mockClear()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ProfilePage: identity (FR-BB116 AC-5)', () => {
  it('renders the caller identity, the role read-only, and a change-password link', async () => {
    renderProfile()

    expect(await screen.findByRole('heading', { name: 'My profile' })).toBeInTheDocument()
    expect(screen.getByText('Ana Test')).toBeInTheDocument()
    expect(screen.getByText('ana@example.com')).toBeInTheDocument()
    expect(screen.getByText('Engineering')).toBeInTheDocument()
    expect(screen.getByText('Employee')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Change password' })).toHaveAttribute('href', '/change-password')
  })

  it('shows the no-department text when the caller has no department', async () => {
    meResponse = () => json(200, { ...meBody, department_id: null, department_name: null })
    renderProfile()

    expect(await screen.findByText('No department')).toBeInTheDocument()
  })

  it('offers the three languages with native labels and labels the control', async () => {
    renderProfile()

    const select = await screen.findByLabelText('Language')
    expect(select).toHaveValue('en')
    expect(screen.getByRole('option', { name: 'Қазақша' })).toHaveAttribute('value', 'kk')
    expect(screen.getByRole('option', { name: 'Русский' })).toHaveAttribute('value', 'ru')
    expect(screen.getByRole('option', { name: 'English' })).toHaveAttribute('value', 'en')
  })

  it('shows a localized alert when the profile cannot be loaded', async () => {
    meResponse = () => json(500, null, { code: 'INTERNAL_ERROR', message: 'boom' })
    renderProfile()

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load your profile. Please try again later.')
  })
})

describe('ProfilePage: change language (FR-BB116 AC-6)', () => {
  it('PATCHes /users/me with the chosen code, switches the UI and persists it', async () => {
    const user = userEvent.setup()
    renderProfile()

    await user.selectOptions(await screen.findByLabelText('Language'), 'ru')

    await waitFor(() => expect(i18n.language).toBe('ru'))
    expect(localStorage.getItem('i18n-lang')).toBe('ru')
    expect(document.documentElement.lang).toBe('ru')

    const calls = patchCalls()
    expect(calls).toHaveLength(1)
    const [url, init] = calls[0]
    expect(url).toBe('/api/v1/users/me')
    expect(JSON.parse(init?.body as string)).toEqual({ preferred_locale: 'ru' })
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok-1')

    // The success notice is localized into the newly selected language.
    expect(await screen.findByRole('status')).toHaveTextContent('Язык обновлён.')
  })

  it('restores the previous language and shows a localized error when the save fails', async () => {
    patchResponse = () => json(500, null, { code: 'INTERNAL_ERROR', message: 'boom' })
    const user = userEvent.setup()
    renderProfile()

    const select = await screen.findByLabelText('Language')
    await user.selectOptions(select, 'kk')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not save your language. The previous language has been restored.',
    )
    expect(patchCalls()).toHaveLength(1)
    expect(i18n.language).toBe('en')
    expect(localStorage.getItem('i18n-lang')).toBe('en')
    expect(document.documentElement.lang).toBe('en')
    expect(select).toHaveValue('en')
  })
})

describe('ProfileRedirect: /profile', () => {
  function renderRedirect(role: string, profileStatus = 200) {
    meResponse = () =>
      profileStatus === 200
        ? json(200, { ...meBody, role_name: role })
        : json(profileStatus, null, { code: 'INTERNAL_ERROR', message: 'boom' })
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/profile']}>
          <Routes>
            <Route path="/profile" element={<ProfileRedirect />} />
            <Route path="/portal/profile" element={<div>Portal profile</div>} />
            <Route path="/admin/profile" element={<div>Admin profile</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
  }

  it('sends an employee to the portal profile', async () => {
    renderRedirect('employee')
    expect(await screen.findByText('Portal profile')).toBeInTheDocument()
  })

  it('sends admin roles to the admin profile', async () => {
    renderRedirect('department_admin')
    expect(await screen.findByText('Admin profile')).toBeInTheDocument()
  })

  it('shows a localized error instead of redirecting when the profile cannot be loaded', async () => {
    renderRedirect('employee', 500)
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load your profile. Please try again later.')
    expect(screen.queryByText('Portal profile')).not.toBeInTheDocument()
  })
})
