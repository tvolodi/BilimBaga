import { describe, it, expect } from 'vitest'
import i18n from '@/i18n'
import { parseInsufficientDetails, describeExamError, safeMessage } from './examErrors'

const t = i18n.t.bind(i18n)

describe('parseInsufficientDetails', () => {
  it('reads an object payload', () => {
    expect(
      parseInsufficientDetails({ rule_id: 'abc', difficulty: 'easy', required: 5, available: 2 }),
    ).toEqual({ ruleId: 'abc', difficulty: 'easy', required: 5, available: 2 })
  })
  it('reads the first object of an array', () => {
    expect(parseInsufficientDetails([{ required: 3, available: 1 }]).required).toBe(3)
  })
  it.each([undefined, null, 'oops', 42, [], ['x'], { required: '5', available: NaN }])(
    'returns no counts for malformed input %#',
    (input) => {
      const r = parseInsufficientDetails(input)
      expect(r.required).toBeUndefined()
      expect(r.available).toBeUndefined()
    },
  )
})

describe('safeMessage', () => {
  it('never stringifies objects', () => {
    expect(safeMessage({ a: 1 })).toBe('')
    expect(safeMessage('x')).toBe('x')
  })
})

describe('describeExamError', () => {
  it('interpolates counts and difficulty for adaptive object details', () => {
    const msg = describeExamError(
      { code: 'INSUFFICIENT_ADAPTIVE_QUESTIONS', details: { rule_id: 'abcdef123', difficulty: 'hard', required: 5, available: 2 } },
      t,
    )
    expect(msg).toMatch(/only 2 Hard questions, 5 are required/)
    expect(msg).toContain('abcdef12')
  })
  it('uses counts without difficulty when it is missing', () => {
    expect(describeExamError({ code: 'INSUFFICIENT_ADAPTIVE_QUESTIONS', details: { required: 5, available: 1 } }, t)).toMatch(/only 1 matching/)
  })
  it.each([undefined, 'string details', { foo: 'bar' }])('falls back for adaptive details %#', (details) => {
    const msg = describeExamError({ code: 'INSUFFICIENT_ADAPTIVE_QUESTIONS', details }, t)
    expect(msg).toMatch(/at least 5 questions per difficulty/)
    expect(msg).not.toContain('[object Object]')
  })
  it('maps INSUFFICIENT_QUESTIONS regardless of details', () => {
    expect(describeExamError({ code: 'INSUFFICIENT_QUESTIONS', details: { x: 1 } }, t)).toMatch(/no questions/)
  })
  it('falls back to message or generic text, never an object', () => {
    expect(describeExamError({ code: 'X', message: 'boom' }, t)).toBe('boom')
    expect(describeExamError({ code: 'X', message: { a: 1 } }, t)).toBe('The operation failed. Please try again.')
    expect(describeExamError(null, t)).toBe('The operation failed. Please try again.')
  })
})
