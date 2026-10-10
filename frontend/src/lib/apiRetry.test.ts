import { describe, it, expect } from 'vitest'
import { isNotFoundError, retryUnlessNotFound } from './apiRetry'

function apiError(status: number | undefined, code: string) {
  return Object.assign(new Error(code), { status, code })
}

describe('isNotFoundError (#470)', () => {
  it('recognises a 404 and the EXAM_NOT_FOUND code', () => {
    expect(isNotFoundError(apiError(404, 'EXAM_NOT_FOUND'))).toBe(true)
    expect(isNotFoundError(apiError(undefined, 'EXAM_NOT_FOUND'))).toBe(true)
    expect(isNotFoundError(apiError(404, 'ERR_HTTP'))).toBe(true)
  })

  it('does not treat other failures as not found', () => {
    expect(isNotFoundError(apiError(500, 'INTERNAL_ERROR'))).toBe(false)
    expect(isNotFoundError(apiError(403, 'FORBIDDEN'))).toBe(false)
    expect(isNotFoundError(new Error('network down'))).toBe(false)
    expect(isNotFoundError(null)).toBe(false)
  })
})

describe('retryUnlessNotFound (#470)', () => {
  it('does not retry a not-found answer, however early the failure', () => {
    expect(retryUnlessNotFound(0, apiError(404, 'EXAM_NOT_FOUND'))).toBe(false)
  })

  it('keeps the default of three retries for other failures', () => {
    const transient = apiError(500, 'INTERNAL_ERROR')
    expect(retryUnlessNotFound(0, transient)).toBe(true)
    expect(retryUnlessNotFound(2, transient)).toBe(true)
    expect(retryUnlessNotFound(3, transient)).toBe(false)
  })
})
