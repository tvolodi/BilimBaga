import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  createAppQueryClient,
  isPasswordChangeRequired,
  PASSWORD_CHANGE_FLAG_KEY,
  PASSWORD_CHANGE_REQUIRED,
} from './passwordChangeRequired'
import { apiFetch } from '@/api/apiFetch'

function coded(code: string): Error {
  return Object.assign(new Error(code), { code })
}

describe('PASSWORD_CHANGE_REQUIRED handling (ISS-160)', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('detects the error code only', () => {
    expect(isPasswordChangeRequired(coded(PASSWORD_CHANGE_REQUIRED))).toBe(true)
    expect(isPasswordChangeRequired(coded('FORBIDDEN'))).toBe(false)
    expect(isPasswordChangeRequired(null)).toBe(false)
    expect(isPasswordChangeRequired(new Error('x'))).toBe(false)
  })

  it('a failing query raises the flag', async () => {
    const qc = createAppQueryClient()
    await qc.fetchQuery({ queryKey: ['x'], queryFn: () => Promise.reject(coded(PASSWORD_CHANGE_REQUIRED)), retry: false }).catch(() => {})
    expect(qc.getQueryData(PASSWORD_CHANGE_FLAG_KEY)).toBe(true)
  })

  it('a failing mutation raises the flag', async () => {
    const qc = createAppQueryClient()
    await qc
      .getMutationCache()
      .build(qc, { mutationFn: () => Promise.reject(coded(PASSWORD_CHANGE_REQUIRED)) })
      .execute(undefined)
      .catch(() => {})
    expect(qc.getQueryData(PASSWORD_CHANGE_FLAG_KEY)).toBe(true)
  })

  it('other errors leave the flag alone', async () => {
    const qc = createAppQueryClient()
    await qc.fetchQuery({ queryKey: ['y'], queryFn: () => Promise.reject(coded('NOT_FOUND')), retry: false }).catch(() => {})
    expect(qc.getQueryData(PASSWORD_CHANGE_FLAG_KEY)).toBeUndefined()
  })

  it('apiFetch surfaces the 403 envelope code so the global handler fires', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        new Response(
          JSON.stringify({ data: null, error: { code: PASSWORD_CHANGE_REQUIRED, message: 'change it' } }),
          { status: 403 },
        ),
      ),
    )
    const qc = createAppQueryClient()
    await qc
      .fetchQuery({ queryKey: ['z'], queryFn: () => apiFetch(qc, '/api/v1/exams'), retry: false })
      .catch(() => {})
    expect(qc.getQueryData(PASSWORD_CHANGE_FLAG_KEY)).toBe(true)
  })
})
