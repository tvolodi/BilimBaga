import { describe, it, expect } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { endRevokedSession, SESSION_REVOKED_KEY } from './sessionRevoked'

describe('endRevokedSession (ISS-249, AC-11)', () => {
  it('drops the token, the user and every per-user cache, and raises the notice', () => {
    const qc = new QueryClient()
    qc.setQueryData(['auth', 'accessToken'], 'tok')
    qc.setQueryData(['auth', 'currentUser'], { id: 'u' })
    qc.setQueryData(['users', 'me'], { id: 'u', preferred_locale: 'ru' })
    qc.setQueryData(['users', 'roles', 'u'], [])
    qc.setQueryData(['roles'], [{ id: 'r' }])
    qc.setQueryData(['roles', 'permissions'], ['users:read'])
    qc.setQueryData(['portal', 'exams'], [])
    qc.setQueryData(['my-results', 1, 'date', 'desc'], [])
    qc.setQueryData(['tenant', 'config'], { app_name: 'BB' })

    endRevokedSession(qc)

    expect(qc.getQueryData(['auth', 'accessToken'])).toBeNull()
    expect(qc.getQueryData(['auth', 'currentUser'])).toBeNull()
    expect(qc.getQueryData(['users', 'me'])).toBeUndefined()
    expect(qc.getQueryData(['users', 'roles', 'u'])).toBeUndefined()
    expect(qc.getQueryData(['roles'])).toBeUndefined()
    expect(qc.getQueryData(['roles', 'permissions'])).toBeUndefined()
    expect(qc.getQueryData(['portal', 'exams'])).toBeUndefined()
    expect(qc.getQueryData(['my-results', 1, 'date', 'desc'])).toBeUndefined()
    // Not per-user: kept.
    expect(qc.getQueryData(['tenant', 'config'])).toEqual({ app_name: 'BB' })
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBe(true)
  })

  it('is idempotent', () => {
    const qc = new QueryClient()
    endRevokedSession(qc)
    endRevokedSession(qc)
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBe(true)
    expect(qc.getQueryData(['auth', 'accessToken'])).toBeNull()
  })
})
