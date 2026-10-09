import { describe, it, expect } from 'vitest'
import { roleErrorKey, ROLE_NAME_PATTERN, NON_ASSIGNABLE_PERMISSIONS } from './roles'

describe('roleErrorKey', () => {
  it.each(['ROLE_NAME_TAKEN', 'ROLE_SYSTEM_IMMUTABLE', 'VALIDATION_ERROR', 'NOT_FOUND', 'FORBIDDEN'])(
    'maps %s to its own key',
    (code) => {
      expect(roleErrorKey({ code, message: 'x' })).toEqual({ key: `roles.errors.${code}` })
    },
  )

  it('extracts the user count for ROLE_IN_USE', () => {
    expect(roleErrorKey({ code: 'ROLE_IN_USE', message: 'role is assigned to 12 user(s)' })).toEqual({
      key: 'roles.errors.ROLE_IN_USE',
      values: { count: 12 },
    })
  })

  it('falls back to a generic key for unknown codes and non-errors', () => {
    expect(roleErrorKey({ code: 'WAT', message: 'x' }).key).toBe('roles.errors.generic')
    expect(roleErrorKey(new Error('network')).key).toBe('roles.errors.generic')
    expect(roleErrorKey(null).key).toBe('roles.errors.generic')
  })
})

describe('role constants mirror backend rules', () => {
  it('name pattern', () => {
    expect(ROLE_NAME_PATTERN.test('qa_reviewer')).toBe(true)
    expect(ROLE_NAME_PATTERN.test('QA Reviewer')).toBe(false)
    expect(ROLE_NAME_PATTERN.test('ab')).toBe(false)
    expect(ROLE_NAME_PATTERN.test('1abc')).toBe(false)
  })
  it('non-assignable permissions', () => {
    expect([...NON_ASSIGNABLE_PERMISSIONS].sort()).toEqual(['roles:manage', 'roles:read', 'tenant:manage'])
  })
})
