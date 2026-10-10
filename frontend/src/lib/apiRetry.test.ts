import { describe, it, expect } from 'vitest'
import { isNotFoundError, retryUnlessNotFound } from './apiRetry'
import { ExamApiError } from '@/api/exams'

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

// #487: the exam hooks throw ExamApiError (httpStatus, not status), and the API answers an unknown exam with 404
// and ERR_NOT_FOUND. These are the real error type and the real code, not a hand-built shape.
describe('isNotFoundError with the error the exam API layer throws (#487)', () => {
  it('recognises the ExamApiError that examsFetch throws for a 404 ERR_NOT_FOUND', () => {
    expect(isNotFoundError(new ExamApiError('exam not found', 'ERR_NOT_FOUND', 404))).toBe(true)
  })

  it('recognises a not-found answer by httpStatus alone and by the code alone', () => {
    expect(isNotFoundError(new ExamApiError('exam not found', 'ERR_UNKNOWN', 404))).toBe(true)
    expect(isNotFoundError(new ExamApiError('exam not found', 'ERR_NOT_FOUND', 0))).toBe(true)
  })

  it('does not treat other ExamApiError answers as not found', () => {
    expect(isNotFoundError(new ExamApiError('boom', 'INTERNAL_ERROR', 500))).toBe(false)
  })

  it('does not retry the ExamApiError not-found answer', () => {
    expect(retryUnlessNotFound(0, new ExamApiError('exam not found', 'ERR_NOT_FOUND', 404))).toBe(false)
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
